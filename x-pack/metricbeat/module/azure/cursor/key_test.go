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
	t.Run("key has expected prefix", func(t *testing.T) {
		key := GenerateStateKey("compute_vm", "sub-123")
		assert.Contains(t, key, "azure-cursor::")
		assert.Regexp(t, `^azure-cursor::[0-9a-f]+$`, key)
	})

	t.Run("same inputs produce same key", func(t *testing.T) {
		key1 := GenerateStateKey("compute_vm", "sub-123")
		key2 := GenerateStateKey("compute_vm", "sub-123")
		assert.Equal(t, key1, key2)
	})

	t.Run("different metricset produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("compute_vm", "sub-123")
		key2 := GenerateStateKey("monitor", "sub-123")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different subscription produces different key", func(t *testing.T) {
		key1 := GenerateStateKey("compute_vm", "sub-111")
		key2 := GenerateStateKey("compute_vm", "sub-222")
		assert.NotEqual(t, key1, key2)
	})

	t.Run("length prefix prevents ambiguity", func(t *testing.T) {
		// "ab" + "c" vs "a" + "bc" — must produce different keys
		key1 := GenerateStateKey("ab", "c")
		key2 := GenerateStateKey("a", "bc")
		assert.NotEqual(t, key1, key2)
	})
}
