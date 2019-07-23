/*
Copyright (c) 2019 VMware, Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package queryer

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//go:generate mockgen -source=queryer.go -destination=./fake/mock_queryer.go -package=fake github.com/vmware/octant/internal/queryer Queryer

type Queryer interface {
	Children(ctx context.Context, object metav1.Object) ([]runtime.Object, error)
	Events(ctx context.Context, object metav1.Object) ([]*corev1.Event, error)
	IngressesForService(ctx context.Context, service *corev1.Service) ([]*extv1beta1.Ingress, error)
	OwnerReference(ctx context.Context, namespace string, ownerReference metav1.OwnerReference) (runtime.Object, error)
	PodsForService(ctx context.Context, service *corev1.Service) ([]*corev1.Pod, error)
	ServicesForIngress(ctx context.Context, ingress *extv1beta1.Ingress) ([]*corev1.Service, error)
	ServicesForPod(ctx context.Context, pod *corev1.Pod) ([]*corev1.Service, error)
	ServiceAccountForPod(ctx context.Context, pod *corev1.Pod) (*corev1.ServiceAccount, error)
}
