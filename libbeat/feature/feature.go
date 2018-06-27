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

import (
	"fmt"
)

// Registry is the global plugin registry, this variable is meant to be temporary to move all the
// internal factory to receive a context that include the current beat registry.
var Registry = newRegistry()

// Featurable implements the description of a feature.
type Featurable interface {
	bundleable

	// Namespace is the kind of plugin or functionality we want to expose as a feature.
	// Examples: Autodiscover's provider, processors, outputs.
	Namespace() string

	// Name is the name of the feature, the name must unique by namespace and be a description of the
	// actual functionality, it is usually the name of the package.
	// Examples: dissect, elasticsearch, redis
	Name() string

	// Factory returns the function used to create an instance of the Feature, the signature
	// of the method is type checked by the 'FindFactory' of each namespace.
	Factory() interface{}

	// Stability is the stability of the Feature, this allow the user to filter embedded functionality
	// by their maturity at runtime.
	// Example: Beta, Experimental, Stable or Undefined.
	Stability() Stability

	String() string
}

// Feature contains the information for a specific feature
type Feature struct {
	namespace string
	name      string
	factory   interface{}
	stability Stability
}

// Namespace return the namespace of the feature.
func (f *Feature) Namespace() string {
	return f.namespace
}

// Name returns the name of the feature.
func (f *Feature) Name() string {
	return f.name
}

// Factory returns the factory for the feature.
func (f *Feature) Factory() interface{} {
	return f.factory
}

// Stability returns the stability level of the feature, current: stable, beta, experimental.
func (f *Feature) Stability() Stability {
	return f.stability
}

// Features return the current feature as a slice to be compatible with Bundle merging and filtering.
func (f *Feature) Features() []Featurable {
	return []Featurable{f}
}

// String return the debug information
func (f *Feature) String() string {
	return fmt.Sprintf("%s/%s (stability: %s)", f.namespace, f.name, f.stability)
}

// New returns a new Feature.
func New(namespace, name string, factory interface{}, stability Stability) *Feature {
	return &Feature{
		namespace: namespace,
		name:      name,
		factory:   factory,
		stability: stability,
	}
}

// RegisterBundle registers a bundle of features.
func RegisterBundle(bundle *Bundle) error {
	for _, f := range bundle.Features() {
		Registry.Register(f)
	}
	return nil
}

// Register register a new feature on the global registry.
func Register(feature Featurable) error {
	return Registry.Register(feature)
}

// MustRegister register a new Feature on the global registry and panic on error.
func MustRegister(feature Featurable) {
	err := Register(feature)
	if err != nil {
		panic(err)
	}
}
