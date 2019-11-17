/*
 * Copyright (c) 2019 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package api

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/octant/internal/log"
	"github.com/vmware-tanzu/octant/internal/module"
	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/pkg/action"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

const (
	RequestSetContentPath       = "setContentPath"
	RequestSetNamespace         = "setNamespace"
	RequestCreateContentStream  = "createContentStream"
	RequestDestroyContentStream = "destroyContentStream"
)

// ContentManagerOption is an option for configuring ContentManager.
type ContentManagerOption func(manager *ContentManager)

// ContentGenerateFunc is a function that generates content. It returns `rerun=true`
// if the action should be be immediately rerun.
type ContentGenerateFunc func(ctx context.Context, state octant.State) (component.ContentResponse, bool, error)

// WithContentGenerator configures the content generate function.
func WithContentGenerator(fn ContentGenerateFunc) ContentManagerOption {
	return func(manager *ContentManager) {
		manager.contentGenerateFunc = fn
	}
}

// WithContentGeneratorPoller configures the poller.
func WithContentGeneratorPoller(poller Poller) ContentManagerOption {
	return func(manager *ContentManager) {
		manager.poller = poller
	}
}

// ContentManager manages content for websockets.
type ContentManager struct {
	moduleManager       module.ManagerInterface
	logger              log.Logger
	contentGenerateFunc ContentGenerateFunc
	poller              Poller
	updateContentCh     chan struct{}
	contentStreamBus    *contentStreamBus
}

// NewContentManager creates an instance of ContentManager.
func NewContentManager(moduleManager module.ManagerInterface, logger log.Logger, options ...ContentManagerOption) *ContentManager {
	cm := &ContentManager{
		moduleManager:    moduleManager,
		logger:           logger,
		poller:           NewInterruptiblePoller("content"),
		updateContentCh:  make(chan struct{}, 1),
		contentStreamBus: initContentStreamBus(),
	}
	cm.contentGenerateFunc = cm.generateContent

	for _, option := range options {
		option(cm)
	}

	return cm
}

var _ octant.StateManager = (*ContentManager)(nil)

// Start starts the manager.
func (cm *ContentManager) Start(ctx context.Context, state octant.State, s octant.StateClient) {
	defer func() {
		close(cm.updateContentCh)
	}()

	updateCancel := state.OnContentPathUpdate(func(contentPath string) {
		cm.updateContentCh <- struct{}{}
	})
	defer updateCancel()

	// cm.poller.Run(ctx, cm.updateContentCh, cm.runUpdate(state, s), event.DefaultScheduleDelay)
}

func (cm *ContentManager) runUpdate(state octant.State, s octant.StateClient) PollerFunc {
	return func(ctx context.Context) bool {
		contentPath := state.GetContentPath()
		if contentPath == "" {
			return false
		}

		contentResponse, _, err := cm.contentGenerateFunc(ctx, state)
		if err != nil {
			return false
		}

		if ctx.Err() == nil {
			s.Send(CreateContentEvent(contentResponse, state.GetNamespace(), contentPath, state.GetQueryParams()))
		}

		return false
	}
}

func (cm *ContentManager) generateContent(ctx context.Context, state octant.State) (component.ContentResponse, bool, error) {
	contentPath := state.GetContentPath()
	return cm.generateContentForPath(ctx, state, contentPath)
}

func (cm *ContentManager) generateContentForPath(ctx context.Context, state octant.State, contentPath string) (component.ContentResponse, bool, error) {
	logger := cm.logger.With("contentPath", contentPath)

	now := time.Now()
	defer func() {
		logger.With("elapsed", time.Since(now)).Debugf("generating content")
	}()

	m, ok := cm.moduleManager.ModuleForContentPath(contentPath)
	if !ok {
		return component.EmptyContentResponse, false, errors.Errorf("unable to find module for content path %q", contentPath)
	}
	modulePath := strings.TrimPrefix(contentPath, m.Name())
	options := module.ContentOptions{
		LabelSet: FiltersToLabelSet(state.GetFilters()),
	}
	contentResponse, err := m.Content(ctx, modulePath, options)
	if err != nil {
		if nfe, ok := err.(notFound); ok && nfe.NotFound() {
			logger.Debugf("path not found, redirecting to parent")
			state.SetContentPath(notFoundRedirectPath(contentPath))
			return component.EmptyContentResponse, true, nil
		} else {
			return component.EmptyContentResponse, false, errors.Wrap(err, "generate content")
		}
	}

	return contentResponse, false, nil

}

// Handlers returns a slice of client request handlers.
func (cm *ContentManager) Handlers() []octant.ClientRequestHandler {
	return []octant.ClientRequestHandler{
		{
			RequestType: RequestCreateContentStream,
			Handler:     cm.RequestCreateContentStream,
		},
		{
			RequestType: RequestDestroyContentStream,
			Handler:     cm.RequestDestroyContentStream,
		},
	}
}

// SetQueryParams sets the current query params.
func (cm *ContentManager) SetQueryParams(state octant.State, payload action.Payload) error {
	if params, ok := payload["params"].(map[string]interface{}); ok {
		// handle filters
		if filters, ok := params["filters"]; ok {
			list, err := FiltersFromQueryParams(filters)
			if err != nil {
				return errors.Wrap(err, "extract filters from query params")
			}
			state.SetFilters(list)
		}
	}

	return nil
}

// SetNamespace sets the current namespace.
func (cm *ContentManager) SetNamespace(state octant.State, payload action.Payload) error {
	namespace, err := payload.String("namespace")
	if err != nil {
		return errors.Wrap(err, "extract namespace from payload")
	}
	state.SetNamespace(namespace)
	return nil
}

// SetContentPath sets the current content path.
func (cm *ContentManager) SetContentPath(state octant.State, payload action.Payload) error {
	contentPath, err := payload.String("contentPath")
	if err != nil {
		return errors.Wrap(err, "extract contentPath from payload")
	}
	if err := cm.SetQueryParams(state, payload); err != nil {
		return errors.Wrap(err, "extract query params from payload")
	}

	state.SetContentPath(contentPath)
	return nil
}

func (cm *ContentManager) RequestCreateContentStream(state octant.State, payload action.Payload) error {
	contentPath, err := payload.String("contentPath")
	if err != nil {
		contentPath = state.DefaultContentPath()
	}

	namespace, err := payload.String("namespace")
	if err != nil {
		return fmt.Errorf("extract namespace from payload: %w", err)
	}

	contentPath = updateNamespaceInContentPath(contentPath, namespace)

	channelID, err := payload.String("channelID")
	if err != nil {
		channelID = contentPath
	}

	// TODO verify contentPath and channelID

	// TODO support filters

	go func() {
		cm.logger.With("channelID", channelID).Infof("creating stream channel")
		ctx := cm.contentStreamBus.createChannel(channelID)

		timer := time.NewTimer(0)

		previousHash := ""
		done := false
		for !done {
			select {
			case <-ctx.Done():
				done = true
				timer.Stop()
			case <-timer.C:
				contentResponse, _, err := cm.generateContentForPath(ctx, state, contentPath)
				if err != nil {
					// TODO: figure out how to handle this error
				}

				stateClient := state.Client()

				data, err := json.Marshal(contentResponse)
				if err != nil {
					// TODO: handle this error
				}
				hash := fmt.Sprintf("%x", sha1.Sum(data))
				if hash != previousHash {
					e := CreateChannelContentEvent(
						contentResponse,
						state.GetNamespace(),
						contentPath,
						channelID,
						state.GetQueryParams())
					stateClient.Send(e)
				}
				previousHash = hash

				timer.Reset(1 * time.Second)
			}
		}
	}()

	return nil
}

func (cm *ContentManager) RequestDestroyContentStream(state octant.State, payload action.Payload) error {
	channelID, err := payload.String("channelID")
	if err != nil {
		return fmt.Errorf("get channel id: %w", err)
	}

	cm.logger.With("channelID", channelID).Infof("destroying channel")
	cm.contentStreamBus.deleteChannel(channelID)

	e := CreateChannelDestroyEvent(channelID)
	state.Client().Send(e)

	return nil
}

type contentStreamCancelFunc func()

type contentStreamBus struct {
	channels map[string]contentStreamCancelFunc
	mu       sync.Mutex
}

func initContentStreamBus() *contentStreamBus {
	csb := &contentStreamBus{
		channels: make(map[string]contentStreamCancelFunc),
		mu:       sync.Mutex{},
	}

	return csb
}

func (csb *contentStreamBus) createChannel(channelID string) context.Context {
	csb.mu.Lock()
	defer csb.mu.Unlock()

	if cancel, ok := csb.channels[channelID]; ok {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())

	fn := func() {
		csb.mu.Lock()
		defer csb.mu.Unlock()

		cancel()
		delete(csb.channels, channelID)
	}

	csb.channels[channelID] = fn
	return ctx
}

func (csb *contentStreamBus) deleteChannel(channelID string) {
	if fn, ok := csb.channels[channelID]; ok {
		fn()
	}
}

type notFound interface {
	NotFound() bool
	Path() string
}

// CreateContentEvent creates a content event.
func CreateContentEvent(contentResponse component.ContentResponse, namespace, contentPath string, queryParams map[string][]string) octant.Event {
	return octant.Event{
		Type: octant.EventTypeContent,
		Data: map[string]interface{}{
			"content":     contentResponse,
			"namespace":   namespace,
			"contentPath": contentPath,
			"queryParams": queryParams,
		},
	}
}

func CreateChannelContentEvent(
	contentResponse component.ContentResponse,
	namespace string,
	contentPath string,
	channelID string,
	queryParams map[string][]string) octant.Event {
	return octant.Event{
		Type: octant.EventTypeChannelContent,
		Data: map[string]interface{}{
			"content":     contentResponse,
			"namespace":   namespace,
			"contentPath": contentPath,
			"channelID":   channelID,
			"queryParams": queryParams,
		},
	}
}

func CreateChannelDestroyEvent(channelID string) octant.Event {
	return octant.Event{
		Type: octant.EventTypeChannelDestroy,
		Data: map[string]interface{}{
			"channelID": channelID,
		},
	}
}
