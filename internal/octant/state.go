/*
 * Copyright (c) 2019 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package octant

import (
	"context"

	"github.com/vmware-tanzu/octant/pkg/action"
)

//go:generate mockgen -destination=./fake/mock_state.go -package=fake github.com/vmware-tanzu/octant/internal/octant State
//go:generate mockgen -destination=./fake/mock_state_client.go -package=fake github.com/vmware-tanzu/octant/internal/octant StateClient
//go:generate mockgen -destination=./fake/mock_state_manager.go -package=fake github.com/vmware-tanzu/octant/internal/octant StateManager

// UpdateCancelFunc cancels the update.
type UpdateCancelFunc func()

// QueryParams are query parameters.
type QueryParams map[string][]string

// State represents Octant's view state.
type State interface {
	// Client returns the state client.
	Client() StateClient

	// DefaultContentPath returns the default content path.
	DefaultContentPath() string

	// SetContentPath sets the content path.
	SetContentPath(string)
	// GetContentPath returns the content path.
	GetContentPath() string
	// OnNamespaceUpdate registers a function to be called with the content path
	// is changed.
	OnContentPathUpdate(fn ContentPathUpdateFunc) UpdateCancelFunc
	// GetQueryParams returns the query params.
	GetQueryParams() QueryParams
	// SetNamespace sets the namespace.
	SetNamespace(namespace string)
	// GetNamespace returns the namespace.
	GetNamespace() string
	// OnNamespaceUpdate returns a function to be called when the namespace
	// is changed.
	OnNamespaceUpdate(fun NamespaceUpdateFunc) UpdateCancelFunc
	// AddFilter adds a label to filtered.
	AddFilter(filter Filter)
	// RemoveFilter removes a filter.
	RemoveFilter(filter Filter)
	// GetFilters returns a slice of filters.
	GetFilters() []Filter
	// SetFilters replaces the current filters with a slice of filters.
	// The slice can be empty.
	SetFilters(filters []Filter)
	// SetContext sets the current context.
	SetContext(requestedContext string)
	// Dispatch dispatches a payload for an action.
	Dispatch(ctx context.Context, actionName string, payload action.Payload) error
	// SendAlert sends an alert.
	SendAlert(alert action.Alert)
}

// ContentPathUpdateFunc is a function that is called when content path is updated.
type ContentPathUpdateFunc func(contentPath string)

// NamespaceUpdateFunc is a function that is called when namespace is updated.
type NamespaceUpdateFunc func(namespace string)

// StateClient is an StateClient.
type StateClient interface {
	Send(event Event)
	ID() string
}

// StateManager manages states for WebsocketState.
type StateManager interface {
	Handlers() []ClientRequestHandler
	Start(ctx context.Context, state State, s StateClient)
}
