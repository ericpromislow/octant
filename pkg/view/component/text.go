/*
Copyright (c) 2019 the Octant contributors. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"encoding/json"
)

// Text is a component for text
type Text struct {
	base
	Config TextConfig `json:"config"`
}

// TextConfig is the contents of Text
type TextConfig struct {
	Text       string `json:"value"`
	IsMarkdown bool   `json:"isMarkdown,omitempty"`
}

// NewText creates a text component
func NewText(s string) *Text {
	return &Text{
		base: newBase(typeText, nil),
		Config: TextConfig{
			Text: s,
		},
	}
}

// NewMarkdownText creates a text component styled with markdown.
func NewMarkdownText(s string) *Text {
	t := NewText(s)
	t.Config.IsMarkdown = true

	return t
}

// IsMarkdown returns if this component is markdown.
func (t *Text) IsMarkdown() bool {
	return t.Config.IsMarkdown
}

// EnableMarkdown enables markdown for this text component.
func (t *Text) EnableMarkdown() {
	t.Config.IsMarkdown = true
}

// DisableMarkdown disables markdown for this text component.
func (t *Text) DisableMarkdown() {
	t.Config.IsMarkdown = false
}

// SupportsTitle denotes this is a TextComponent.
func (t *Text) SupportsTitle() {}

type textMarshal Text

// MarshalJSON implements json.Marshaler
func (t *Text) MarshalJSON() ([]byte, error) {
	m := textMarshal(*t)
	m.Metadata.Type = typeText

	checksum, err := t.base.GenerateChecksum(t.Config)
	if err != nil {
		return nil, err
	}

	m.Metadata.Checksum = checksum

	return json.Marshal(&m)
}

// String returns the text content of the component.
func (t *Text) String() string {
	return t.Config.Text
}

// LessThan returns true if this component's value is less than the argument supplied.
func (t *Text) LessThan(i interface{}) bool {
	v, ok := i.(*Text)
	if !ok {
		return false
	}

	return t.Config.Text < v.Config.Text

}
