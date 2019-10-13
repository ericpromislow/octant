/*
 * Copyright (c) 2019 VMware, Inc. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package printer

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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

func loadPodMetrics(list *unstructured.UnstructuredList) ([]podMetric, error) {
	if list == nil {
		return nil, fmt.Errorf("list is nil")
	}

	var metrics []podMetric

	for i := range list.Items {
		pm, err := loadPodMetric(&list.Items[i])
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, pm)
	}

	return metrics, nil
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
