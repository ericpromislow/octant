package queryer

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/vmware/octant/internal/config"
	"github.com/vmware/octant/internal/gvk"
	"github.com/vmware/octant/pkg/navigation"
)

//go:generate mockgen -source=resource.go -destination=./fake/group_version_factory.go -package=fake github.com/vmware/octant/internal/queryer GroupVersionFactory

type GroupVersionFactory interface {
	GroupVersions(ctx context.Context, dashConfig config.Dash) ([]schema.GroupVersion, error)
}

type Resource struct {
	GroupVersionKind schema.GroupVersionKind
	Verbs            metav1.Verbs
}

type ResourcesFactory func(ctx context.Context) ([]Resource, error)

func DefaultResourcesFactory(dashConfig config.Dash) ResourcesFactory {
	return func(ctx context.Context) ([]Resource, error) {
		return preferredResources(ctx, dashConfig, &defaultGroupVersions{}, &crdGroupVersions{})
	}
}

var (
	allowedGroupVersionKinds = []schema.GroupVersionKind{
		gvk.ClusterRoleBindingGVK,
		gvk.ClusterRoleGVK,
		gvk.ConfigMapGVK,
		gvk.CronJobGVK,
		gvk.CustomResourceDefinitionGVK,
		gvk.DaemonSetGVK,
		gvk.DeploymentGVK,
		gvk.ExtReplicaSet,
		gvk.IngressGVK,
		gvk.JobGVK,
		gvk.ServiceAccountGVK,
		gvk.SecretGVK,
		gvk.ServiceGVK,
		gvk.PodGVK,
		gvk.PersistentVolumeClaimGVK,
		gvk.ReplicationControllerGVK,
		gvk.StatefulSetGVK,
		gvk.RoleBindingGVK,
		gvk.RoleGVK,
	}
)

func preferredResources(ctx context.Context, dashConfig config.Dash, factories ...GroupVersionFactory) ([]Resource, error) {
	if dashConfig == nil {
		return nil, errors.New("octant config is required")
	}

	clusterClient := dashConfig.ClusterClient()
	discoveryClient, err := clusterClient.DiscoveryClient()
	if err != nil {
		return nil, err
	}

	var resourceLists []metav1.APIResourceList

	var groupVersions []schema.GroupVersion
	for _, factory := range factories {
		cur, err := factory.GroupVersions(ctx, dashConfig)
		if err != nil {
			return nil, err
		}

		groupVersions = append(groupVersions, cur...)
	}

	for _, groupVersion := range groupVersions {
		resourceList, err := discoveryClient.ServerResourcesForGroupVersion(groupVersion.String())
		if err != nil {
			return nil, err
		}

		resourceLists = append(resourceLists, *resourceList)

	}

	var resources []Resource

	for resourceListIndex := range resourceLists {
		resourceList := resourceLists[resourceListIndex]

		gv, err := schema.ParseGroupVersion(resourceList.GroupVersion)
		if err != nil {
			return nil, err
		}

		for i := range resourceList.APIResources {
			apiResource := resourceList.APIResources[i]

			cur := schema.GroupVersionKind{
				Group:   gv.Group,
				Version: gv.Version,
				Kind:    apiResource.Kind,
			}

			if !apiResource.Namespaced {
				continue
			}

			resource := Resource{
				GroupVersionKind: cur,
				Verbs:            apiResource.Verbs,
			}

			resources = append(resources, resource)
		}
	}

	return resources, nil
}

type defaultGroupVersions struct{}

func (defaultGroupVersions) GroupVersions(ctx context.Context, dashConfig config.Dash) ([]schema.GroupVersion, error) {
	m := make(map[schema.GroupVersion]bool)

	for _, groupVersionKind := range allowedGroupVersionKinds {
		m[groupVersionKind.GroupVersion()] = true
	}

	var groupVersions []schema.GroupVersion
	for k := range m {
		groupVersions = append(groupVersions, k)
	}

	return groupVersions, nil
}

type crdGroupVersions struct{}

func (crdGroupVersions) GroupVersions(ctx context.Context, dashConfig config.Dash) ([]schema.GroupVersion, error) {
	if dashConfig == nil {
		return nil, errors.New("octant config is nil")
	}

	crds, err := navigation.CustomResourceDefinitions(ctx, dashConfig.ObjectStore())
	if err != nil {
		return nil, err
	}

	var list []schema.GroupVersion

	for _, crd := range crds {
		for _, version := range crd.Spec.Versions {
			if !version.Served || !version.Storage {
				continue
			}
			groupVersion := schema.GroupVersion{
				Group:   crd.Spec.Group,
				Version: version.Name,
			}
			list = append(list, groupVersion)
		}
	}

	return list, nil

}
