package queryer

import (
	"context"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	appsv1 "k8s.io/api/apps/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/pkg/apis/apps"
	"k8s.io/kubernetes/pkg/apis/batch"
	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/vmware/octant/internal/config"
	"github.com/vmware/octant/internal/util/strings"
	"github.com/vmware/octant/pkg/store"
)

//go:generate mockgen -source=../../vendor/k8s.io/client-go/discovery/discovery_client.go -imports=openapi_v2=github.com/googleapis/gnostic/OpenAPIv2 -destination=./fake/mock_discovery.go -package=fake k8s.io/client-go/discovery DiscoveryInterface

type ObjectStoreQueryerOption func(q *ObjectStoreQueryer)

func SetResourcesFactory(fn ResourcesFactory) ObjectStoreQueryerOption {
	return func(q *ObjectStoreQueryer) {
		q.resourcesFactory = fn
	}
}

type ObjectStoreQueryer struct {
	objectStore store.Store

	children         *childrenCache
	podsForServices  *podsForServicesCache
	owner            *ownerCache
	resourcesFactory ResourcesFactory
}

var _ Queryer = (*ObjectStoreQueryer)(nil)

func New(dashConfig config.Dash, options ...ObjectStoreQueryerOption) (*ObjectStoreQueryer, error) {
	if dashConfig == nil {
		return nil, errors.New("octant config is nil")
	}

	q := &ObjectStoreQueryer{
		objectStore: dashConfig.ObjectStore(),

		children:        initChildrenCache(),
		podsForServices: initPodsForServicesCache(),
		owner:           initOwnerCache(),
	}

	for _, option := range options {
		option(q)
	}

	if q.resourcesFactory == nil {
		q.resourcesFactory = DefaultResourcesFactory(dashConfig)
	}

	return q, nil
}

func (osq *ObjectStoreQueryer) Children(ctx context.Context, owner metav1.Object) ([]runtime.Object, error) {
	if owner == nil {
		return []runtime.Object{}, nil
	}

	ctx, span := trace.StartSpan(ctx, "queryer:Children")
	defer span.End()

	stored, ok := osq.children.get(owner.GetUID())

	if ok {
		return stored, nil
	}

	var children []runtime.Object

	ch := make(chan runtime.Object)
	childrenProcessed := make(chan bool, 1)
	go func() {
		for child := range ch {
			if child == nil {
				continue
			}
			children = append(children, child)
		}
		childrenProcessed <- true
	}()

	var g errgroup.Group

	resources, err := osq.resourcesFactory(ctx)
	if err != nil {
		return nil, err
	}

	sem := semaphore.NewWeighted(10)

	for i := range resources {
		resource := resources[i]

		groupVersionKind := resource.GroupVersionKind

		key := store.Key{
			Namespace:  owner.GetNamespace(),
			APIVersion: groupVersionKind.GroupVersion().String(),
			Kind:       groupVersionKind.Kind,
		}

		if osq.canList(resource.Verbs) {
			continue
		}

		g.Go(func() error {
			if err := sem.Acquire(ctx, 1); err != nil {
				return err
			}
			defer sem.Release(1)

			objects, err := osq.objectStore.List(ctx, key)
			if err != nil {
				return errors.Wrapf(err, "unable to retrieve %+v", key)
			}

			for _, object := range objects {
				if metav1.IsControlledBy(object, owner) {
					ch <- object
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	close(ch)
	<-childrenProcessed
	close(childrenProcessed)

	osq.children.set(owner.GetUID(), children)

	return children, nil
}

func (osq *ObjectStoreQueryer) canList(verbs metav1.Verbs) bool {
	return !strings.Contains("watch", verbs) ||
		!strings.Contains("list", verbs)
}

func (osq *ObjectStoreQueryer) Events(ctx context.Context, object metav1.Object) ([]*corev1.Event, error) {
	if object == nil {
		return nil, errors.New("object is nil")
	}

	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(object)
	if err != nil {
		return nil, err
	}

	u := &unstructured.Unstructured{Object: m}

	key := store.Key{
		Namespace:  u.GetNamespace(),
		APIVersion: "v1",
		Kind:       "Event",
	}

	allEvents, err := osq.objectStore.List(ctx, key)
	if err != nil {
		return nil, err
	}

	var events []*corev1.Event
	for _, unstructuredEvent := range allEvents {
		event := &corev1.Event{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredEvent.Object, event)
		if err != nil {
			return nil, err
		}

		involvedObject := event.InvolvedObject
		if involvedObject.Namespace == u.GetNamespace() &&
			involvedObject.APIVersion == u.GetAPIVersion() &&
			involvedObject.Kind == u.GetKind() &&
			involvedObject.Name == u.GetName() {
			events = append(events, event)
		}
	}

	return events, nil
}

func (osq *ObjectStoreQueryer) IngressesForService(ctx context.Context, service *corev1.Service) ([]*v1beta1.Ingress, error) {
	if service == nil {
		return nil, errors.New("nil service")
	}

	key := store.Key{
		Namespace:  service.Namespace,
		APIVersion: "extensions/v1beta1",
		Kind:       "Ingress",
	}
	ul, err := osq.objectStore.List(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving ingresses")
	}

	var results []*v1beta1.Ingress

	for _, u := range ul {
		ingress := &v1beta1.Ingress{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, ingress)
		if err != nil {
			return nil, errors.Wrap(err, "converting unstructured ingress")
		}
		if err = copyObjectMeta(ingress, u); err != nil {
			return nil, errors.Wrap(err, "copying object metadata")
		}
		backends := osq.listIngressBackends(*ingress)
		if !containsBackend(backends, service.Name) {
			continue
		}

		results = append(results, ingress)
	}
	return results, nil
}

func (osq *ObjectStoreQueryer) listIngressBackends(ingress v1beta1.Ingress) []v1beta1.IngressBackend {
	var backends []v1beta1.IngressBackend

	if ingress.Spec.Backend != nil && ingress.Spec.Backend.ServiceName != "" {
		backends = append(backends, *ingress.Spec.Backend)
	}

	for _, rule := range ingress.Spec.Rules {
		if rule.IngressRuleValue.HTTP == nil {
			continue
		}
		for _, p := range rule.IngressRuleValue.HTTP.Paths {
			if p.Backend.ServiceName == "" {
				continue
			}
			backends = append(backends, p.Backend)
		}
	}

	return backends
}

func (osq *ObjectStoreQueryer) OwnerReference(ctx context.Context, namespace string, ownerReference metav1.OwnerReference) (runtime.Object, error) {
	key := store.Key{
		Namespace:  namespace,
		APIVersion: ownerReference.APIVersion,
		Kind:       ownerReference.Kind,
		Name:       ownerReference.Name,
	}

	object, ok := osq.owner.get(key)
	if ok {
		return object, nil
	}

	owner, err := osq.objectStore.Get(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "get owner from store")
	}

	osq.owner.set(key, owner)

	return owner, nil
}

func (osq *ObjectStoreQueryer) PodsForService(ctx context.Context, service *corev1.Service) ([]*corev1.Pod, error) {
	if service == nil {
		return nil, errors.New("nil service")
	}

	stored, ok := osq.podsForServices.get(service.UID)
	if ok {
		return stored, nil
	}

	key := store.Key{
		Namespace:  service.Namespace,
		APIVersion: "v1",
		Kind:       "Pod",
	}

	selector, err := osq.getSelector(service)
	if err != nil {
		return nil, errors.Wrapf(err, "creating pod selector for service: %v", service.Name)
	}
	pods, err := osq.loadPods(ctx, key, selector)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching pods for service: %v", service.Name)
	}

	osq.podsForServices.set(service.UID, pods)

	return pods, nil
}

func (osq *ObjectStoreQueryer) loadPods(ctx context.Context, key store.Key, labelSelector *metav1.LabelSelector) ([]*corev1.Pod, error) {
	objects, err := osq.objectStore.List(ctx, key)
	if err != nil {
		return nil, err
	}

	var list []*corev1.Pod

	for _, object := range objects {
		pod := &corev1.Pod{}
		if err := scheme.Scheme.Convert(object, pod, runtime.InternalGroupVersioner); err != nil {
			return nil, err
		}

		if err := copyObjectMeta(pod, object); err != nil {
			return nil, err
		}

		podSelector := &metav1.LabelSelector{
			MatchLabels: pod.GetLabels(),
		}

		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, err
		}

		if selector == nil || isEqualSelector(labelSelector, podSelector) || selector.Matches(labels.Set(pod.Labels)) {
			list = append(list, pod)
		}
	}

	return list, nil
}

func (osq *ObjectStoreQueryer) ServicesForIngress(ctx context.Context, ingress *v1beta1.Ingress) ([]*corev1.Service, error) {
	if ingress == nil {
		return nil, errors.New("ingress is nil")
	}

	backends := osq.listIngressBackends(*ingress)
	var services []*corev1.Service
	for _, backend := range backends {
		key := store.Key{
			Namespace:  ingress.Namespace,
			APIVersion: "v1",
			Kind:       "Service",
			Name:       backend.ServiceName,
		}
		u, err := osq.objectStore.Get(ctx, key)
		if err != nil {
			return nil, errors.Wrapf(err, "retrieving service backend: %v", backend)
		}

		if u == nil {
			continue
		}

		svc := &corev1.Service{}
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, svc)
		if err != nil {
			return nil, errors.Wrap(err, "converting unstructured service")
		}
		if err := copyObjectMeta(svc, u); err != nil {
			return nil, errors.Wrap(err, "copying object metadata")
		}
		services = append(services, svc)
	}
	return services, nil
}

func (osq *ObjectStoreQueryer) ServicesForPod(ctx context.Context, pod *corev1.Pod) ([]*corev1.Service, error) {
	var results []*corev1.Service
	if pod == nil {
		return nil, errors.New("nil pod")
	}

	key := store.Key{
		Namespace:  pod.Namespace,
		APIVersion: "v1",
		Kind:       "Service",
	}
	ul, err := osq.objectStore.List(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving services")
	}
	for _, u := range ul {
		svc := &corev1.Service{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, svc)
		if err != nil {
			return nil, errors.Wrap(err, "converting unstructured service")
		}
		if err = copyObjectMeta(svc, u); err != nil {
			return nil, errors.Wrap(err, "copying object metadata")
		}
		labelSelector, err := osq.getSelector(svc)
		if err != nil {
			return nil, errors.Wrapf(err, "creating pod selector for service: %v", svc.Name)
		}
		selector, err := metav1.LabelSelectorAsSelector(labelSelector)
		if err != nil {
			return nil, errors.Wrap(err, "invalid selector")
		}

		if selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
			continue
		}
		results = append(results, svc)
	}
	return results, nil
}

func (osq *ObjectStoreQueryer) ServiceAccountForPod(ctx context.Context, pod *corev1.Pod) (*corev1.ServiceAccount, error) {
	if pod == nil {
		return nil, errors.New("pod is nil")
	}

	if pod.Spec.ServiceAccountName == "" {
		return nil, nil
	}

	key := store.Key{
		Namespace:  pod.Namespace,
		APIVersion: "v1",
		Kind:       "ServiceAccount",
		Name:       pod.Spec.ServiceAccountName,
	}

	u, err := osq.objectStore.Get(ctx, key)
	if err != nil {
		return nil, errors.WithMessagef(err, "retrieve service account %q from namespace %q",
			key.Namespace, key.Namespace)
	}

	serviceAccount := &corev1.ServiceAccount{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, serviceAccount); err != nil {
		return nil, errors.WithMessage(err, "converting unstructured object to service account")
	}

	if err = copyObjectMeta(serviceAccount, u); err != nil {
		return nil, errors.Wrap(err, "copying object metadata")
	}

	return serviceAccount, nil

}

func (osq *ObjectStoreQueryer) getSelector(object runtime.Object) (*metav1.LabelSelector, error) {
	switch t := object.(type) {
	case *appsv1.DaemonSet:
		return t.Spec.Selector, nil
	case *appsv1.StatefulSet:
		return t.Spec.Selector, nil
	case *batchv1beta1.CronJob:
		return nil, nil
	case *corev1.ReplicationController:
		selector := &metav1.LabelSelector{
			MatchLabels: t.Spec.Selector,
		}
		return selector, nil
	case *v1beta1.ReplicaSet:
		return t.Spec.Selector, nil
	case *appsv1.ReplicaSet:
		return t.Spec.Selector, nil
	case *appsv1.Deployment:
		return t.Spec.Selector, nil
	case *corev1.Service:
		selector := &metav1.LabelSelector{
			MatchLabels: t.Spec.Selector,
		}
		return selector, nil
	case *apps.DaemonSet:
		return t.Spec.Selector, nil
	case *apps.StatefulSet:
		return t.Spec.Selector, nil
	case *batch.CronJob:
		return nil, nil
	case *core.ReplicationController:
		selector := &metav1.LabelSelector{
			MatchLabels: t.Spec.Selector,
		}
		return selector, nil
	case *apps.ReplicaSet:
		return t.Spec.Selector, nil
	case *apps.Deployment:
		return t.Spec.Selector, nil
	case *core.Service:
		selector := &metav1.LabelSelector{
			MatchLabels: t.Spec.Selector,
		}
		return selector, nil
	default:
		return nil, errors.Errorf("unable to retrieve selector for type %T", object)
	}
}

func copyObjectMeta(to interface{}, from *unstructured.Unstructured) error {
	object, ok := to.(metav1.Object)
	if !ok {
		return errors.Errorf("%T is not an object", to)
	}

	t, err := meta.TypeAccessor(object)
	if err != nil {
		return errors.Wrapf(err, "accessing type meta")
	}
	t.SetAPIVersion(from.GetAPIVersion())
	t.SetKind(from.GetObjectKind().GroupVersionKind().Kind)

	object.SetNamespace(from.GetNamespace())
	object.SetName(from.GetName())
	object.SetGenerateName(from.GetGenerateName())
	object.SetUID(from.GetUID())
	object.SetResourceVersion(from.GetResourceVersion())
	object.SetGeneration(from.GetGeneration())
	object.SetSelfLink(from.GetSelfLink())
	object.SetCreationTimestamp(from.GetCreationTimestamp())
	object.SetDeletionTimestamp(from.GetDeletionTimestamp())
	object.SetDeletionGracePeriodSeconds(from.GetDeletionGracePeriodSeconds())
	object.SetLabels(from.GetLabels())
	object.SetAnnotations(from.GetAnnotations())
	object.SetInitializers(from.GetInitializers())
	object.SetOwnerReferences(from.GetOwnerReferences())
	object.SetClusterName(from.GetClusterName())
	object.SetFinalizers(from.GetFinalizers())

	return nil
}

// extraKeys are keys that should be ignored in labels. These keys are added
// by tools or by Kubernetes itself.
var extraKeys = []string{
	"statefulset.kubernetes.io/pod-name",
	appsv1.DefaultDeploymentUniqueLabelKey,
	"controller-revision-hash",
	"pod-template-generation",
}

func isEqualSelector(s1, s2 *metav1.LabelSelector) bool {
	s1Copy := s1.DeepCopy()
	s2Copy := s2.DeepCopy()

	for _, key := range extraKeys {
		delete(s1Copy.MatchLabels, key)
		delete(s2Copy.MatchLabels, key)
	}

	return equality.Semantic.DeepEqual(s1Copy, s2Copy)
}

func containsBackend(lst []v1beta1.IngressBackend, s string) bool {
	for _, item := range lst {
		if item.ServiceName == s {
			return true
		}
	}
	return false
}
