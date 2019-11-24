package api

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

// TODO: make this the real one
type ContentGenerateFunc2 func(ctx context.Context, state octant.State, contentPath string) (component.ContentResponse, error)

type ContentStreamer interface {
	Stream(ctx context.Context)
}

type contentStreamer struct {
	channelID            string
	state                octant.State
	contentPath          string
	contentGeneratorFunc ContentGenerateFunc2
	timerExpiration      time.Duration
}

var _ ContentStreamer = (*contentStreamer)(nil)

func newContentStreamer(channelID, contentPath string, state octant.State, genFunc ContentGenerateFunc2) *contentStreamer {
	return &contentStreamer{
		channelID:            channelID,
		state:                state,
		contentPath:          contentPath,
		contentGeneratorFunc: genFunc,
		timerExpiration:      1 * time.Second,
	}
}

func (cs *contentStreamer) Stream(ctx context.Context) {
	timer := time.NewTimer(0)

	stateClient := cs.state.Client()

	previousHash := ""
	done := false
	for !done {
		select {
		case <-ctx.Done():
			done = true
			timer.Stop()
		case <-timer.C:
			contentResponse, err := cs.contentGeneratorFunc(ctx, cs.state, cs.contentPath)
			if err != nil {
				stateClient.Send(octant.CreateErrorEvent(err))
				continue
			}

			data, err := json.Marshal(contentResponse)
			if err != nil {
				stateClient.Send(octant.CreateErrorEvent(err))
				continue
			}
			hash := fmt.Sprintf("%x", sha1.Sum(data))
			if hash != previousHash {
				e := CreateChannelContentEvent(
					contentResponse,
					cs.state.GetNamespace(),
					cs.contentPath,
					cs.channelID,
					cs.state.GetQueryParams())
				stateClient.Send(e)
			}
			previousHash = hash

			timer.Reset(cs.timerExpiration)
		}
	}
}
