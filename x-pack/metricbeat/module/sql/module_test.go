// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sql

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/paths"
)

// TestModuleBuilderSharesState verifies that all module instances created by the
// same ModuleBuilder share the same sharedRegistryState pointer.
func TestModuleBuilderSharesState(t *testing.T) {
	factory := ModuleBuilder()

	mod1, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	mod2, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	m1, ok := mod1.(*module)
	require.True(t, ok, "mod1 should be *module")
	m2, ok := mod2.(*module)
	require.True(t, ok, "mod2 should be *module")

	assert.Same(t, m1.shared, m2.shared,
		"All module instances from the same ModuleBuilder must share the same sharedRegistryState")
}

// TestGetCursorRegistryReturnsSamePointer verifies that repeated calls to
// GetCursorRegistry return the exact same *statestore.Registry pointer when
// the data path has not changed.
func TestGetCursorRegistryReturnsSamePointer(t *testing.T) {
	tmpPaths := paths.New()
	tmpPaths.Data = t.TempDir()

	factory := ModuleBuilder()

	mod1, err := factory(mb.BaseModule{Logger: logp.NewNopLogger(), Paths: tmpPaths})
	require.NoError(t, err)

	mod2, err := factory(mb.BaseModule{Logger: logp.NewNopLogger(), Paths: tmpPaths})
	require.NoError(t, err)

	sqlMod1, ok := mod1.(Module)
	require.True(t, ok, "mod1 should implement sql.Module")
	reg1, err := sqlMod1.GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg1)

	sqlMod2, ok := mod2.(Module)
	require.True(t, ok, "mod2 should implement sql.Module")
	reg2, err := sqlMod2.GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg2)

	assert.Same(t, reg1, reg2,
		"GetCursorRegistry must return the same pointer when data path is unchanged")

	// Call again on the first module - still the same pointer (cached).
	reg1again, err := sqlMod1.GetCursorRegistry()
	require.NoError(t, err)
	assert.Same(t, reg1, reg1again,
		"Repeated calls on the same module must return cached registry")
}

// TestGetCursorRegistryPathChange verifies that using a module with a different
// per-instance data path causes GetCursorRegistry to create a new registry at
// the new location.
func TestGetCursorRegistryPathChange(t *testing.T) {
	tmpPaths1 := paths.New()
	tmpPaths1.Data = t.TempDir()
	tmpPaths2 := paths.New()
	tmpPaths2.Data = t.TempDir()

	factory := ModuleBuilder()
	mod1, err := factory(mb.BaseModule{Logger: logp.NewNopLogger(), Paths: tmpPaths1})
	require.NoError(t, err)

	mod2, err := factory(mb.BaseModule{Logger: logp.NewNopLogger(), Paths: tmpPaths2})
	require.NoError(t, err)

	sqlMod1, ok := mod1.(Module)
	require.True(t, ok, "mod1 should implement sql.Module")

	sqlMod2, ok := mod2.(Module)
	require.True(t, ok, "mod2 should implement sql.Module")

	// First path
	reg1, err := sqlMod1.GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg1)

	// Same path - cached
	reg1again, err := sqlMod1.GetCursorRegistry()
	require.NoError(t, err)
	assert.Same(t, reg1, reg1again)

	// Second path - new registry expected
	reg2, err := sqlMod2.GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg2)
	assert.NotSame(t, reg1, reg2,
		"Changing the data path must produce a new registry")
}

// TestGetCursorRegistryConcurrent verifies that concurrent calls to
// GetCursorRegistry from multiple goroutines are safe and all return
// the same pointer.
func TestGetCursorRegistryConcurrent(t *testing.T) {
	tmpPaths := paths.New()
	tmpPaths.Data = t.TempDir()

	factory := ModuleBuilder()

	const n = 20
	modules := make([]mb.Module, n)
	for i := range modules {
		mod, err := factory(mb.BaseModule{Logger: logp.NewNopLogger(), Paths: tmpPaths})
		require.NoError(t, err)
		modules[i] = mod
	}

	type result struct {
		reg interface{}
		err error
	}

	// Call GetCursorRegistry concurrently from all modules.
	var wg sync.WaitGroup
	resultsCh := make(chan result, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(mod mb.Module) {
			defer wg.Done()
			sqlMod, ok := mod.(Module)
			if !ok {
				resultsCh <- result{err: fmt.Errorf("module does not implement sql.Module")}
				return
			}
			reg, err := sqlMod.GetCursorRegistry()
			if err != nil {
				resultsCh <- result{err: err}
				return
			}
			resultsCh <- result{reg: reg}
		}(modules[i])
	}

	wg.Wait()
	close(resultsCh)

	var first interface{}
	for r := range resultsCh {
		require.NoError(t, r.err)
		require.NotNil(t, r.reg)
		if first == nil {
			first = r.reg
		} else {
			assert.Same(t, first, r.reg,
				"All concurrent calls must return the same registry pointer")
		}
	}
}

// TestModuleImplementsInterface verifies the module type implements the Module interface.
func TestModuleImplementsInterface(t *testing.T) {
	factory := ModuleBuilder()
	mod, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	_, ok := mod.(Module)
	assert.True(t, ok, "module must implement the sql.Module interface")

	// mod is already mb.Module (the return type of factory), so no assertion needed.
	assert.NotNil(t, mod, "module must not be nil")
}
