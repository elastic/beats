// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

func TestStandaloneStore(t *testing.T) {

	var key = []byte("somekey")
	var value = []byte("somevalue")

	tempDir := t.TempDir()

	log := logptest.NewTestingLogger(t, "")
	store, err := newStore(log, tempDir, "store-cache")
	require.NoError(t, err)

	err = store.Set(key, value, 0)
	assert.NoError(t, err)

	result, err := store.Get(key)
	if assert.NoError(t, err) {
		assert.Equal(t, value, result)
	}

	err = store.Close()
	assert.NoError(t, err)
}
