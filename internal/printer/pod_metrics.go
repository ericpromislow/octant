/*
 * Copyright (c) 2019 VMware, Inc. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package printer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/vmware/octant/pkg/store"
	"github.com/vmware/octant/pkg/view/component"
)

var (
	podMetricsCols = component.NewTableCols("Container", "Memory", "CPU")
)

func printPodMetrics(ctx context.Context, pod *corev1.Pod, options Options) (component.Component, error) {
	if pod == nil {
		return nil, fmt.Errorf("can't load metrics for nil pod")
	}

	key := store.Key{
		Namespace:  pod.Namespace,
		APIVersion: "metrics.k8s.io/v1beta1",
		Kind:       "PodMetrics",
		Name:       pod.Name,
	}

	objectStore := options.DashConfig.ObjectStore()

	object, found, err := objectStore.Get(ctx, key, store.Direct)
	if err != nil {
		return nil, fmt.Errorf("load pod metrics for %s: %w", pod.Name, err)
	}

	table := component.NewTable("Metrics", "No metrics were found", podMetricsCols)

	if !found {
		return table, nil
	}

	metric, err := loadPodMetric(object)
	if err != nil {
		return nil, fmt.Errorf("load pod metric: %w", err)
	}

	var containerNames []string
	for containerName := range metric.containers {
		containerNames = append(containerNames, containerName)
	}
	sort.Strings(containerNames)

	for _, containerName := range containerNames {
		cm := metric.containers[containerName]
		row := component.TableRow{
			"Container": component.NewText(containerName),
			"Memory":    component.NewText(cm.memory),
			"CPU":       component.NewText(cm.cpu),
		}
		table.Add(row)
	}

	return table, nil
}

type podMetric struct {
	name       string
	containers map[string]containerMetric
}

type containerMetric struct {
	cpu    string
	memory string
}

func loadPodMetric(object *unstructured.Unstructured) (podMetric, error) {
	if object == nil {
		return podMetric{}, fmt.Errorf("pod metric is nil")
	}

	pm := podMetric{
		name:       object.GetName(),
		containers: make(map[string]containerMetric),
	}

	rawList, found, err := unstructured.NestedSlice(object.Object, "containers")
	if err != nil {
		return podMetric{}, fmt.Errorf("load containers: %w", err)
	}

	if !found {
		return podMetric{}, fmt.Errorf("unknown object structure")
	}

	for _, raw := range rawList {
		container, ok := raw.(map[string]interface{})
		if !ok {
			return podMetric{}, fmt.Errorf("container was not an object")
		}

		name, metric, err := loadContainerMetric(container)
		if err != nil {
			return podMetric{}, fmt.Errorf("load container metric: %w", err)
		}

		pm.containers[name] = metric
	}

	return pm, nil
}

func loadContainerMetric(object map[string]interface{}) (string, containerMetric, error) {
	if object == nil {
		return "", containerMetric{}, fmt.Errorf("container object is nil")
	}

	name, err := unstructuredString(object, "name")
	if err != nil {
		return "", containerMetric{}, err
	}

	usage, found, err := unstructured.NestedMap(object, "usage")
	if err != nil {
		return "", containerMetric{}, fmt.Errorf("unable to get container usage object: %w", err)
	}
	if !found {
		return "", containerMetric{}, fmt.Errorf("container usage object was not found")
	}

	cpu, err := unstructuredString(usage, "cpu")
	if err != nil {
		return "", containerMetric{}, err
	}

	memory, err := unstructuredString(usage, "memory")
	if err != nil {
		return "", containerMetric{}, err
	}

	return name, containerMetric{
		cpu:    cpu,
		memory: memory,
	}, nil
}

func unstructuredString(object map[string]interface{}, fields ...string) (string, error) {
	if object == nil {
		return "", fmt.Errorf("object is nil")
	}

	s, found, err := unstructured.NestedString(object, fields...)
	if err != nil {
		return "", fmt.Errorf("unable to get string %s: %w", strings.Join(fields, "."), err)
	}

	if !found {
		return "", fmt.Errorf("%s was not found", strings.Join(fields, "."))
	}

	return s, nil
}
