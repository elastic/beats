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
	base := func(overrides ...string) [5]string {
		args := [5]string{"monitor", "sub-123", "5m0s", "2m0s", "type=Microsoft.Compute/virtualMachines ids= groups= query= svc="}
		for i, v := range overrides {
			if i < 5 {
				args[i] = v
			}
		}
		return args
	}
	call := func(args [5]string) string {
		return GenerateStateKey(args[0], args[1], args[2], args[3], args[4])
	}

	t.Run("key has expected prefix", func(t *testing.T) {
		key := call(base())
		assert.Contains(t, key, "azure-cursor::")
		assert.Regexp(t, `^azure-cursor::[0-9a-f]+$`, key)
	})

	t.Run("same inputs produce same key", func(t *testing.T) {
		assert.Equal(t, call(base()), call(base()))
	})

	t.Run("different metricset produces different key", func(t *testing.T) {
		assert.NotEqual(t, call(base("compute_vm")), call(base("monitor")))
	})

	t.Run("different subscription produces different key", func(t *testing.T) {
		assert.NotEqual(t, call(base("monitor", "sub-111")), call(base("monitor", "sub-222")))
	})

	t.Run("different period produces different key", func(t *testing.T) {
		assert.NotEqual(t, call(base("monitor", "sub-123", "5m0s")), call(base("monitor", "sub-123", "1m0s")))
	})

	t.Run("different latency produces different key", func(t *testing.T) {
		assert.NotEqual(t,
			call(base("monitor", "sub-123", "5m0s", "2m0s")),
			call(base("monitor", "sub-123", "5m0s", "0s")))
	})

	t.Run("different resourcesKey produces different key", func(t *testing.T) {
		a := base()
		b := base()
		b[4] = "type=Microsoft.Storage/storageAccounts ids= groups= query= svc="
		assert.NotEqual(t, call(a), call(b))
	})

	t.Run("length prefix prevents ambiguity", func(t *testing.T) {
		// "ab"+"c" vs "a"+"bc" in any position must produce different keys
		key1 := GenerateStateKey("ab", "c", "x", "y", "z")
		key2 := GenerateStateKey("a", "bc", "x", "y", "z")
		assert.NotEqual(t, key1, key2)
	})
}
