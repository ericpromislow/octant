package queryer

import (
	"context"
	"sort"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	clusterFake "github.com/vmware/octant/internal/cluster/fake"
	configFake "github.com/vmware/octant/internal/config/fake"
	"github.com/vmware/octant/internal/queryer/fake"
	"github.com/vmware/octant/internal/testutil"
	"github.com/vmware/octant/pkg/store"
	storeFake "github.com/vmware/octant/pkg/store/fake"
)

func Test_preferredResources(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	groupVersion := schema.GroupVersion{
		Group:   "group",
		Version: "version",
	}
	resourceList := &metav1.APIResourceList{
		TypeMeta:     metav1.TypeMeta{},
		GroupVersion: groupVersion.String(),
		APIResources: []metav1.APIResource{
			{
				Kind:       "Kind1",
				Verbs:      []string{"list"},
				Namespaced: true,
			},
			{
				Kind:       "Kind2",
				Verbs:      []string{"list"},
				Namespaced: false,
			},
		},
	}
	discoveryClient := clusterFake.NewMockDiscoveryInterface(controller)
	discoveryClient.EXPECT().
		ServerResourcesForGroupVersion(groupVersion.String()).
		Return(resourceList, nil)
	clusterClient := clusterFake.NewMockClientInterface(controller)
	clusterClient.EXPECT().DiscoveryClient().Return(discoveryClient, nil)
	dashConfig := configFake.NewMockDash(controller)
	dashConfig.EXPECT().ClusterClient().Return(clusterClient)

	groupVersionFactory := fake.NewMockGroupVersionFactory(controller)
	groupVersionFactory.EXPECT().
		GroupVersions(gomock.Any(), dashConfig).
		Return([]schema.GroupVersion{groupVersion}, nil)

	ctx := context.Background()
	got, err := preferredResources(ctx, dashConfig, groupVersionFactory)
	require.NoError(t, err)

	expected := []Resource{
		{
			GroupVersionKind: schema.GroupVersionKind{
				Group:   "group",
				Version: "version",
				Kind:    "Kind1",
			},
			Verbs: []string{"list"},
		},
	}
	assert.Equal(t, expected, got)
}

func Test_defaultGroupVersions(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	dashConfig := configFake.NewMockDash(controller)
	ctx := context.Background()

	gdv := defaultGroupVersions{}
	got, err := gdv.GroupVersions(ctx, dashConfig)
	require.NoError(t, err)

	sort.Slice(got, func(i, j int) bool {
		return got[i].Group < got[j].Group
	})

	expected := []schema.GroupVersion{
		{Group: "", Version: "v1"},
		{Group: "apiextensions.k8s.io", Version: "v1beta1"},
		{Group: "apps", Version: "v1"},
		{Group: "batch", Version: "v1"},
		{Group: "batch", Version: "v1beta1"},
		{Group: "extensions", Version: "v1beta1"},
		{Group: "rbac.authorization.k8s.io", Version: "v1"},
	}
	assert.Equal(t, expected, got)

}

func Test_crdGroupVersions(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	objectStore := storeFake.NewMockStore(controller)
	crd := testutil.CreateCRD("crd")
	crd.Spec.Group = "group"
	crd.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{
		Name:    "v1",
		Served:  true,
		Storage: true,
	}}
	key := store.Key{
		APIVersion: crd.APIVersion,
		Kind:       crd.Kind,
	}
	objectStore.EXPECT().
		HasAccess(gomock.Any(), key, "list").
		Return(nil)
	objectStore.EXPECT().
		List(gomock.Any(), key).
		Return(testutil.ToUnstructuredList(t, crd), nil)

	dashConfig := configFake.NewMockDash(controller)
	dashConfig.EXPECT().ObjectStore().Return(objectStore)

	ctx := context.Background()

	cdv := crdGroupVersions{}
	got, err := cdv.GroupVersions(ctx, dashConfig)
	require.NoError(t, err)

	expected := []schema.GroupVersion{{
		Group:   "group",
		Version: "v1",
	}}
	assert.Equal(t, expected, got)
}
