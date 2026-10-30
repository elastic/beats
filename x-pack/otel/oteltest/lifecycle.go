// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oteltest

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// lifecycleTimeout bounds each Start and Shutdown call so that a hanging
// component fails the test instead of blocking until the test binary times out.
const lifecycleTimeout = 2 * time.Minute

// CheckReceiverLifecycle runs CheckComponentLifecycle against a logs receiver.
func CheckReceiverLifecycle(t *testing.T, factory receiver.Factory, cfg component.Config) {
	t.Helper()
	CheckComponentLifecycle(t, factory, func(t *testing.T) (component.Component, error) {
		return factory.CreateLogs(t.Context(), receivertest.NewNopSettings(factory.Type()), cfg, consumertest.NewNop())
	})
}

// CheckExporterLifecycle runs CheckComponentLifecycle against a logs exporter.
func CheckExporterLifecycle(t *testing.T, factory exporter.Factory, cfg component.Config) {
	t.Helper()
	CheckComponentLifecycle(t, factory, func(t *testing.T) (component.Component, error) {
		return factory.CreateLogs(t.Context(), exportertest.NewNopSettings(factory.Type()), cfg)
	})
}

// CheckProcessorLifecycle runs CheckComponentLifecycle against a logs processor.
func CheckProcessorLifecycle(t *testing.T, factory processor.Factory, cfg component.Config) {
	t.Helper()
	CheckComponentLifecycle(t, factory, func(t *testing.T) (component.Component, error) {
		return factory.CreateLogs(t.Context(), processortest.NewNopSettings(factory.Type()), cfg, consumertest.NewNop())
	})
}

// CheckExtensionLifecycle runs CheckComponentLifecycle against an extension.
func CheckExtensionLifecycle(t *testing.T, factory extension.Factory, cfg component.Config) {
	t.Helper()
	CheckComponentLifecycle(t, factory, func(t *testing.T) (component.Component, error) {
		return factory.Create(t.Context(), extensiontest.NewNopSettings(factory.Type()), cfg)
	})
}

// CheckComponentLifecycle runs the same lifecycle checks that mdatagen generates
// for every opentelemetry-collector-contrib component (generated_component_test.go):
//   - the default config passes componenttest.CheckConfigStruct
//   - Shutdown without Start returns without error
//   - two components created by create can be started and shut down one after the other
//
// create must build the component from a valid config. Shutdown is called with
// a context without deadline, as the collector may do.
func CheckComponentLifecycle(t *testing.T, factory component.Factory, create func(t *testing.T) (component.Component, error)) {
	t.Helper()

	t.Run("config-struct", func(t *testing.T) {
		cfg := factory.CreateDefaultConfig()
		if reflect.ValueOf(cfg).Kind() == reflect.Map {
			t.Skip("CheckConfigStruct only supports struct configs")
		}
		require.NoError(t, componenttest.CheckConfigStruct(cfg), "default config struct is invalid")
	})

	t.Run("shutdown-without-start", func(t *testing.T) {
		comp, err := create(t)
		require.NoError(t, err, "creating the component should succeed")
		callWithin(t, "Shutdown", func() error { return comp.Shutdown(context.Background()) })
	})

	t.Run("start-shutdown", func(t *testing.T) {
		host := componenttest.NewNopHost()
		for i := range 2 {
			comp, err := create(t)
			require.NoError(t, err, "creating the component should succeed")
			callWithin(t, "Start", func() error { return comp.Start(context.Background(), host) })
			callWithin(t, "Shutdown", func() error { return comp.Shutdown(context.Background()) })
			t.Logf("component %d started and shut down", i+1)
		}
	})
}

func callWithin(t *testing.T, name string, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- fn() }()
	select {
	case err := <-done:
		require.NoError(t, err, "%s should succeed", name)
	case <-time.After(lifecycleTimeout):
		t.Fatalf("%s did not return within %s", name, lifecycleTimeout)
	}
}
