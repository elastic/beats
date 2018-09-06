// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/beatless/core"
)

type mockProvider struct {
	runners []core.Runner
	name    string
}

func (m *mockProvider) CreateFunctions(clientFactory clientFactory) ([]core.Runner, error) {
	return m.runners, nil
}
func (m *mockProvider) Name() string { return m.name }

func TestRegistry(t *testing.T) {
	t.Run("provider", testProviderLookup)
	t.Run("functions", testFunctionLookup)
}

type mockFunction struct {
	name string
}

func (mf *mockFunction) Run(ctx context.Context, client core.Client) error { return nil }
func (mf *mockFunction) Name() string                                      { return mf.name }

func testProviderLookup(t *testing.T) {
	name := "myprovider"
	myprovider := &mockProvider{}

	providerFn := func(log *logp.Logger, registry *Registry, config *common.Config) (Provider, error) {
		return myprovider, nil
	}

	f := Feature(
		name,
		providerFn,
		feature.NewDetails(name, "provider for testing", feature.Experimental),
	)

	t.Run("adding and retrieving a provider", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		err := global.Register(f)
		if !assert.NoError(t, err) {
			return
		}

		factory, err := wrapper.Lookup(name)
		if !assert.NoError(t, err) {
			return
		}

		// Compare func pointers instead of comparing the function value.
		assert.Equal(t, reflect.ValueOf(providerFn).Pointer(), reflect.ValueOf(factory).Pointer())
	}))

	t.Run("retrieving a non existing provider", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		_, err := wrapper.Lookup("unknown")
		assert.Error(t, err)
	}))

	t.Run("invalid provider name when doing lookup", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		_, err := wrapper.Lookup("")
		assert.Error(t, err)
	}))
}

func testFunctionLookup(t *testing.T) {
	name := "myprovider"
	myprovider := &mockProvider{}

	providerFn := func(log *logp.Logger, registry *Registry, config *common.Config) (Provider, error) {
		return myprovider, nil
	}

	f := Feature(
		name,
		providerFn,
		feature.NewDetails(name, "provider for testing", feature.Experimental),
	)

	fnName := "myfunc"
	myfunction := &mockFunction{name}
	functionFn := func(provider Provider, config *common.Config) (Function, error) {
		return myfunction, nil
	}

	fnFeature := FunctionFeature(name, fnName, functionFn, feature.NewDetails(
		name,
		"provider for testing",
		feature.Experimental,
	))

	t.Run("adding and retrieving a function", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		err := global.Register(f)
		if !assert.NoError(t, err) {
			return
		}

		err = global.Register(fnFeature)
		if !assert.NoError(t, err) {
			return
		}

		factory, err := wrapper.LookupFunction(name, fnName)
		if !assert.NoError(t, err) {
			return
		}

		// Compare func pointers instead of comparing the function value.
		assert.Equal(t, reflect.ValueOf(functionFn).Pointer(), reflect.ValueOf(factory).Pointer())
	}))

	t.Run("return an error if the provider doesn't exist", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		err := global.Register(f)
		if !assert.NoError(t, err) {
			return
		}

		err = global.Register(fnFeature)
		if !assert.NoError(t, err) {
			return
		}

		_, err = wrapper.LookupFunction("I do not exist", fnName)
		assert.Equal(t, errInvalidProvider, err)
	}))

	t.Run("return an error if the function doesn't exist", withRegistry(func(
		t *testing.T,
		global *feature.FeatureRegistry,
		wrapper *Registry,
	) {
		err := global.Register(f)
		if !assert.NoError(t, err) {
			return
		}

		err = global.Register(fnFeature)
		if !assert.NoError(t, err) {
			return
		}

		_, err = wrapper.LookupFunction(name, "I do not exist")
		assert.Error(t, err)
	}))
}

func withRegistry(fn func(t *testing.T, global *feature.FeatureRegistry, registry *Registry)) func(t *testing.T) {
	return func(t *testing.T) {
		global := feature.NewRegistry()
		wrapped := NewRegistry(global)
		fn(t, global, wrapped)
	}
}
