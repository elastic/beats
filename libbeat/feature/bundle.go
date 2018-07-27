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

package feature

// FilterFunc is the function use to filter elements from a bundle.
type FilterFunc = func(Featurable) bool

// Bundleable merges featurable and bundle interface together.
type bundleable interface {
	Features() []Featurable
}

// Bundle defines a list of features available in the current beat.
type Bundle struct {
	features []Featurable
}

// NewBundle creates a new Bundle of feature to be registered.
func NewBundle(features ...Featurable) *Bundle {
	return &Bundle{features: features}
}

// FilterWith takes a predicate and return a list of filtered bundle matching the predicate.
func (b *Bundle) FilterWith(pred FilterFunc) *Bundle {
	var filtered []Featurable

	for _, feature := range b.features {
		if pred(feature) {
			filtered = append(filtered, feature)
		}
	}
	return NewBundle(filtered...)
}

// Filter creates a new bundle with only the feature matching the requested stability.
func (b *Bundle) Filter(stabilities ...Stability) *Bundle {
	return b.FilterWith(HasStabilityPred(stabilities...))
}

// Features returns the interface features slice so
func (b *Bundle) Features() []Featurable {
	return b.features
}

// MustBundle takes existing bundle or features and create a new Bundle with all the merged Features.
func MustBundle(bundle ...bundleable) *Bundle {
	var merged []Featurable
	for _, feature := range bundle {
		merged = append(merged, feature.Features()...)
	}
	return NewBundle(merged...)
}

// HasStabilityPred returns true if the feature match any of the provided stabilities.
func HasStabilityPred(stabilities ...Stability) FilterFunc {
	return func(f Featurable) bool {
		for _, s := range stabilities {
			if s == f.Description().Stability() {
				return true
			}
		}
		return false
	}
}
