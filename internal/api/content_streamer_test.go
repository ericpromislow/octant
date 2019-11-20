package api

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/vmware-tanzu/octant/internal/octant"
	"github.com/vmware-tanzu/octant/internal/octant/fake"
	"github.com/vmware-tanzu/octant/pkg/view/component"
)

func Test_contentStreamer_Stream(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	state := fake.NewMockState(controller)

	gen := func(ctx context.Context, state octant.State, contentPath string) (component.ContentResponse, error) {
		return component.ContentResponse{}, fmt.Errorf("broken")
	}
	cs := contentStreamer{
		channelID:            "channelID",
		state:                state,
		contentPath:          "/path",
		contentGeneratorFunc: gen,
	}
}
