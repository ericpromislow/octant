/*
 * Copyright (c) 2019 VMware, Inc. All Rights Reserved.
 * SPDX-License-Identifier: Apache-2.0
 */

package component

import "encoding/json"

type GraphLink struct {
	ID            string            `json:"id,omitempty"`
	Source        string            `json:"source"`
	Target        string            `json:"target"`
	Label         string            `json:"label,omitempty"`
	Data          GraphData         `json:"data,omitempty"`
	Line          string            `json:"line,omitempty"`
	TextTransform string            `json:"textTransform,omitempty"`
	TextAngle     float64           `json:"textAngle,omitempty"`
	TextPath      string            `json:"textPath,omitempty"`
	MidPoint      GraphNodePosition `json:"midPoint,omitempty"`
}

type GraphNodePosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type GraphNodeDimension struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type GraphData struct {
	Palette ColorPalette `json:"palette,omitempty"`
}

type GraphMeta interface{}

type GraphNodeOption func(gn *GraphNode)

type ColorPalette struct {
	LightFg string `json:"lightFg"`
	LightBg string `json:"lightBg"`
	DarkFg  string `json:"darkFg"`
	DarkBg  string `json:"darkBg"`
}

func GraphNodeOptionPalette(palette ColorPalette) GraphNodeOption {
	return func(gn *GraphNode) {
		gn.Data.Palette = palette
	}
}

type GraphNode struct {
	ID    string    `json:"id"`
	Label string    `json:"label,omitempty"`
	Data  GraphData `json:"data,omitempty"`
	Meta  GraphMeta `json:"meta,omitempty"`
}

func CreateGraphNode(id, label string, options ...GraphNodeOption) GraphNode {
	gn := GraphNode{
		ID:    id,
		Label: label,
	}

	for _, option := range options {
		option(&gn)
	}

	return gn
}

type GraphConfig struct {
	Links []GraphLink `json:"links"`
	Nodes []GraphNode `json:"nodes"`
}

type Graph struct {
	base
	Config GraphConfig `json:"config"`
}

var _ Component = (*Graph)(nil)

func NewGraph() *Graph {
	g := &Graph{
		base: newBase(typeGraph, nil),
	}

	return g
}

func (g *Graph) MarshalJSON() ([]byte, error) {
	type t Graph
	m := t(*g)
	m.Metadata.Type = typeGraph
	return json.Marshal(&m)
}
