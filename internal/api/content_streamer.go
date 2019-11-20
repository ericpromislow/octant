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

type contentStreamer struct {
	channelID            string
	state                octant.State
	contentPath          string
	contentGeneratorFunc ContentGenerateFunc2
	quitCh               chan struct{}
	timerExpiration      time.Duration
}

func (cs *contentStreamer) Stream(ctx context.Context) {
	timer := time.NewTimer(0)

	previousHash := ""
	done := false
	for !done {
		select {
		case _, ok := <-cs.quitCh:
			if !ok {
				done = true
				timer.Stop()
			}

		case <-ctx.Done():
			done = true
			timer.Stop()
		case <-timer.C:
			contentResponse, err := cs.contentGeneratorFunc(ctx, cs.state, cs.contentPath)
			if err != nil {
				// TODO: send error event
			}

			stateClient := cs.state.Client()

			data, err := json.Marshal(contentResponse)
			if err != nil {
				// TODO: send error event
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
