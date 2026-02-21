// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sql

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/metricbeat/mb"
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

	m1 := mod1.(*module)
	m2 := mod2.(*module)

	assert.Same(t, m1.shared, m2.shared,
		"All module instances from the same ModuleBuilder must share the same sharedRegistryState")
}

// TestGetCursorRegistryReturnsSamePointer verifies that repeated calls to
// GetCursorRegistry return the exact same *statestore.Registry pointer when
// the data path has not changed.
func TestGetCursorRegistryReturnsSamePointer(t *testing.T) {
	tmpDir := t.TempDir()
	origData := paths.Paths.Data
	paths.Paths.Data = tmpDir
	t.Cleanup(func() { paths.Paths.Data = origData })

	factory := ModuleBuilder()

	mod1, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	mod2, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	reg1, err := mod1.(Module).GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg1)

	reg2, err := mod2.(Module).GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg2)

	assert.Same(t, reg1, reg2,
		"GetCursorRegistry must return the same pointer when data path is unchanged")

	// Call again on the first module - still the same pointer (cached).
	reg1again, err := mod1.(Module).GetCursorRegistry()
	require.NoError(t, err)
	assert.Same(t, reg1, reg1again,
		"Repeated calls on the same module must return cached registry")
}

// TestGetCursorRegistryPathChange verifies that changing paths.Paths.Data
// causes GetCursorRegistry to create a new registry at the new location.
func TestGetCursorRegistryPathChange(t *testing.T) {
	tmpDir1 := t.TempDir()
	tmpDir2 := t.TempDir()

	origData := paths.Paths.Data
	t.Cleanup(func() { paths.Paths.Data = origData })

	factory := ModuleBuilder()
	mod, err := factory(mb.BaseModule{})
	require.NoError(t, err)

	// First path
	paths.Paths.Data = tmpDir1
	reg1, err := mod.(Module).GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg1)

	// Same path - cached
	reg1again, err := mod.(Module).GetCursorRegistry()
	require.NoError(t, err)
	assert.Same(t, reg1, reg1again)

	// Change path - new registry expected
	paths.Paths.Data = tmpDir2
	reg2, err := mod.(Module).GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg2)
	assert.NotSame(t, reg1, reg2,
		"Changing the data path must produce a new registry")

	// Revert to first path - yet another new registry (previous one is not cached)
	paths.Paths.Data = tmpDir1
	reg3, err := mod.(Module).GetCursorRegistry()
	require.NoError(t, err)
	require.NotNil(t, reg3)
	assert.NotSame(t, reg1, reg3,
		"Reverting to a previous path creates a new registry (old cache was replaced)")
	assert.NotSame(t, reg2, reg3)
}

// TestGetCursorRegistryConcurrent verifies that concurrent calls to
// GetCursorRegistry from multiple goroutines are safe and all return
// the same pointer.
func TestGetCursorRegistryConcurrent(t *testing.T) {
	tmpDir := t.TempDir()
	origData := paths.Paths.Data
	paths.Paths.Data = tmpDir
	t.Cleanup(func() { paths.Paths.Data = origData })

	factory := ModuleBuilder()

	const n = 20
	modules := make([]mb.Module, n)
	for i := range modules {
		mod, err := factory(mb.BaseModule{})
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
			reg, err := mod.(Module).GetCursorRegistry()
			resultsCh <- result{reg: reg, err: err}
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

	_, ok = mod.(mb.Module)
	assert.True(t, ok, "module must implement mb.Module")
}
