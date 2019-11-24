package api

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/internal/octant/fake"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

func Test_contentStreamer_Stream(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan bool, 1)

	cr := component.ContentResponse{
		IconName: "icon",
	}

	client := fake.NewMockStateClient(controller)
	client.EXPECT().
		Send(gomock.Any()).Do(func(e octant.Event) {
		expected := octant.Event{
			Type: octant.EventTypeChannelContent,
			Data: ChannelContentResponse{
				Content:     cr,
				Namespace:   "testing",
				ContentPath: "/path",
				ChannelID:   "channelID",
				QueryParams: octant.QueryParams{},
			},
		}
		require.Equal(t, expected, e)

		cancel()
		done <- true
	})

	state := fake.NewMockState(controller)
	state.EXPECT().Client().Return(client).AnyTimes()

	// TODO: Figure out why this is being called
	state.EXPECT().GetNamespace().Return("testing").AnyTimes()

	// TODO: why are query params needed?
	queryParams := octant.QueryParams{}
	state.EXPECT().GetQueryParams().Return(queryParams).AnyTimes()

	gen := func(ctx context.Context, state octant.State, contentPath string) (component.ContentResponse, error) {
		return cr, nil
	}
	cs := contentStreamer{
		channelID:            "channelID",
		state:                state,
		contentPath:          "/path",
		contentGeneratorFunc: gen,
	}

	cs.Stream(ctx)

	<-done
}
