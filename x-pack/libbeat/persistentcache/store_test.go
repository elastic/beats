// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package persistentcache

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
)

func TestStandaloneStore(t *testing.T) {
	type valueType struct {
		Something string
	}

	var key = []byte("somekey")
	var value = []byte("somevalue")

	tempDir, err := ioutil.TempDir("", "beat-data-dir-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	store, err := newStore(logp.NewLogger("test"), tempDir, "store-cache")
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
