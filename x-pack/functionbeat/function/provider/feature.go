// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"github.com/elastic/beats/v7/libbeat/feature"
)

// getNamespace return the namespace for functions of a specific provider. The registry have a flat view
// representation of the plugin world this mean we don't really have a tree, instead what we do is
// to create a unique keys per providers that will only keep the functions of the provider.
func getNamespace(provider string) string {
	return namespace + "." + provider + ".functions"
}

// newFeature creates a new Provider feature to be added to the global registry.
// The namespace will be 'functionbeat.provider' in the registry.
func newFeature(name string, factory Factory) *feature.Feature {
	return feature.New(namespace, name, factory)
}

// newFunctionFeature Feature creates a new function feature to be added to the global registry
// The namespace will be 'functionbeat.provider.local' in the registry.
func newFunctionFeature(
	provider, name string,
	factory FunctionFactory,
) *feature.Feature {
	return feature.New(getNamespace(provider), name, factory)
}

// builder is used to have a fluent interface to build a set of function for a specific provider, it
// provides a fluent interface to the developper of provider and functions, it wraps the Feature
// functions to make sure the namespace are correctly configured.
type builder struct {
	name     string
	features []feature.Featurable
}

// Builder creates a new provider builder, it is used to define a provider and the function
// it supports.
func Builder(name string, factory Factory, details feature.Details) *builder {
	return &builder{
		name: name,
		features: []feature.Featurable{
			newFeature(name, factory),
		},
	}
}

func (b *builder) Features() []feature.Featurable {
	return b.features
}

// AddFunction adds a new function type to the provider and return the builder.
func (b *builder) AddFunction(
	name string,
	factory FunctionFactory,
	details feature.Details,
) *builder {
	b.features = append(b.features, newFunctionFeature(b.name, name, factory))
	return b
}
