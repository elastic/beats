// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"github.com/menderesk/beats/v7/libbeat/feature"
)

// getNamespace return the namespace for functions of a specific provider. The registry have a flat view
// representation of the plugin world this mean we don't really have a tree, instead what we do is
// to create a unique keys per providers that will only keep the functions of the provider.
func getNamespace(provider string) string {
	return namespace + "." + provider + ".functions"
}

// Feature creates a new Provider feature to be added to the global registry.
// The namespace will be 'functionbeat.provider' in the registry.
func Feature(name string, factory Factory, details feature.Details) *feature.Feature {
	return feature.New(namespace, name, factory, details)
}

// FunctionFeature Feature creates a new function feature to be added to the global registry
// The namespace will be 'functionbeat.provider.local' in the registry.
func FunctionFeature(
	provider, name string,
	factory FunctionFactory,
	details feature.Details,
) *feature.Feature {
	return feature.New(getNamespace(provider), name, factory, details)
}

// Builder is used to have a fluent interface to build a set of function for a specific provider, it
// provides a fluent interface to the developper of provider and functions, it wraps the Feature
// functions to make sure the namespace are correctly configured.
type Builder struct {
	name   string
	bundle *feature.Bundle
}

// MustCreate creates a new provider builder, it is used to define a provider and the function
// it supports.
func MustCreate(name string, factory Factory, details feature.Details) *Builder {
	return &Builder{name: name, bundle: feature.NewBundle(Feature(name, factory, details))}
}

// Bundle transforms the provider and the functions into a bundle feature.
func (b *Builder) Bundle() *feature.Bundle {
	return b.bundle
}

// MustAddFunction adds a new function type to the provider and return the builder.
func (b *Builder) MustAddFunction(
	name string,
	factory FunctionFactory,
	details feature.Details,
) *Builder {
	b.bundle = feature.MustBundle(b.bundle, FunctionFeature(b.name, name, factory, details))
	return b
}
