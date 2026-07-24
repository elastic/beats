// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package cursor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateStateKey(t *testing.T) {
	t.Run("key has expected prefix and format", func(t *testing.T) {
		key := GenerateStateKey("monitor", "sub-123", "Microsoft.Compute/virtualMachines")
		assert.Contains(t, key, "azure-cursor::")
		assert.Regexp(t, `^azure-cursor::[0-9a-f]+$`, key)
	})

	t.Run("same inputs produce same key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-123", "Microsoft.Compute/virtualMachines")
		key2 := GenerateStateKey("monitor", "sub-123", "Microsoft.Compute/virtualMachines")
		assert.Equal(t, key1, key2)
	})

	t.Run("different metricset produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-123", "Microsoft.Compute/virtualMachines")
		key2 := GenerateStateKey("compute_vm", "sub-123", "Microsoft.Compute/virtualMachines")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different subscription produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-111", "Microsoft.Compute/virtualMachines")
		key2 := GenerateStateKey("monitor", "sub-222", "Microsoft.Compute/virtualMachines")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different namespaces produce different key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Compute/virtualMachines|ids=|groups=|types=|queries=")
		key2 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Storage/storageAccounts|ids=|groups=|types=|queries=")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different resource group produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Compute/virtualMachines|ids=|groups=prod|types=|queries=")
		key2 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Compute/virtualMachines|ids=|groups=staging|types=|queries=")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different resource query produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Compute/virtualMachines|ids=|groups=|types=|queries=resourceType eq 'Microsoft.Compute/virtualMachines'")
		key2 := GenerateStateKey("monitor", "sub-123", "ns=Microsoft.Compute/virtualMachines|ids=|groups=|types=|queries=resourceType eq 'Microsoft.Storage/storageAccounts'")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("length prefix prevents ambiguity", func(t *testing.T) {
		// "ab"+"c" vs "a"+"bc" in any position must produce different keys
		key1 := GenerateStateKey("ab", "c", "ns")
		key2 := GenerateStateKey("a", "bc", "ns")
		assert.NotEqual(t, key1, key2)
	})
}
