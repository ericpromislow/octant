/*
 * Copyright (c) 2019 the Octant contributors. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package api_test

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/vmware-tanzu/octant/internal/api"
	"github.com/vmware-tanzu/octant/internal/log"
	moduleFake "github.com/vmware-tanzu/octant/internal/module/fake"
	"github.com/vmware-tanzu/octant/internal/octant"
	octantFake "github.com/vmware-tanzu/octant/internal/octant/fake"
	"github.com/vmware-tanzu/octant/pkg/action"
)

func TestContentManager_Handlers(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	moduleManager := moduleFake.NewMockManagerInterface(controller)

	logger := log.NopLogger()

	manager := api.NewContentManager(moduleManager, logger)
	AssertHandlers(t, manager, []string{
		api.RequestCreateContentStream,
		api.RequestDestroyContentStream,
	})
}

func TestContentManager_SetNamespace(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	m := moduleFake.NewMockModule(controller)
	m.EXPECT().Name().Return("name").AnyTimes()

	moduleManager := moduleFake.NewMockManagerInterface(controller)

	state := octantFake.NewMockState(controller)
	state.EXPECT().SetNamespace("kube-system")

	logger := log.NopLogger()

	manager := api.NewContentManager(moduleManager, logger)

	payload := action.Payload{
		"namespace": "kube-system",
	}

	require.NoError(t, manager.SetNamespace(state, payload))
}

func TestContentManager_SetQueryParams(t *testing.T) {
	tests := []struct {
		name    string
		payload action.Payload
		setup   func(state *octantFake.MockState)
	}{
		{
			name: "single filter",
			payload: action.Payload{
				"params": map[string]interface{}{
					"filters": "foo:bar",
				},
			},
			setup: func(state *octantFake.MockState) {
				state.EXPECT().SetFilters([]octant.Filter{
					{Key: "foo", Value: "bar"},
				})
			},
		},
		{
			name: "multiple filters",
			payload: action.Payload{
				"params": map[string]interface{}{
					"filters": []interface{}{
						"foo:bar",
						"baz:qux",
					},
				},
			},
			setup: func(state *octantFake.MockState) {
				state.EXPECT().SetFilters([]octant.Filter{
					{Key: "foo", Value: "bar"},
					{Key: "baz", Value: "qux"},
				})
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			m := moduleFake.NewMockModule(controller)
			m.EXPECT().Name().Return("name").AnyTimes()

			moduleManager := moduleFake.NewMockManagerInterface(controller)

			state := octantFake.NewMockState(controller)
			require.NotNil(t, test.setup)
			test.setup(state)

			logger := log.NopLogger()

			manager := api.NewContentManager(moduleManager, logger)
			require.NoError(t, manager.SetQueryParams(state, test.payload))
		})
	}
}
