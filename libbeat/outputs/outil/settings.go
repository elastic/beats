// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package outil

import "strings"

// Settings configures how BuildSelectorFromConfig creates a Selector from
// a given configuration object.
type Settings struct {
	// single selector key and default option keyword
	Key string

	// multi-selector key in config
	MultiKey string

	// if enabled a selector `key` in config will be generated, if `key` is present
	EnableSingleOnly bool

	// Fail building selector if `key` and `multiKey` are missing
	FailEmpty bool

	// Case configures the case-sensitivity of generated strings.
	Case SelectorCase
}

// SelectorCase is used to configure a Selector output string casing.
// Use SelectorLowerCase or SelectorUpperCase to enforce the Selector to
// always generate lower case or upper case strings.
type SelectorCase uint8

const (
	// SelectorKeepCase instructs the Selector to not modify the string output.
	SelectorKeepCase SelectorCase = iota

	// SelectorLowerCase instructs the Selector to always transform the string output to lower case only.
	SelectorLowerCase

	// SelectorUpperCase instructs the Selector to always transform the string output to upper case only.
	SelectorUpperCase
)

// WithKey returns a new Settings struct with updated `Key` setting.
func (s Settings) WithKey(key string) Settings {
	s.Key = key
	return s
}

// WithMultiKey returns a new Settings struct with updated `MultiKey` setting.
func (s Settings) WithMultiKey(key string) Settings {
	s.MultiKey = key
	return s
}

// WithEnableSingleOnly returns a new Settings struct with updated `EnableSingleOnly` setting.
func (s Settings) WithEnableSingleOnly(b bool) Settings {
	s.EnableSingleOnly = b
	return s
}

// WithFailEmpty returns a new Settings struct with updated `FailEmpty` setting.
func (s Settings) WithFailEmpty(b bool) Settings {
	s.FailEmpty = b
	return s
}

// WithSelectorCase returns a new Settings struct with updated `Case` setting.
func (s Settings) WithSelectorCase(c SelectorCase) Settings {
	s.Case = c
	return s
}

func (selCase SelectorCase) apply(in string) string {
	switch selCase {
	case SelectorLowerCase:
		return strings.ToLower(in)
	case SelectorUpperCase:
		return strings.ToUpper(in)
	default:
		return in
	}
}
