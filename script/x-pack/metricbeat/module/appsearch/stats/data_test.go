// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package stats

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventMapping(t *testing.T) {
	files, err := filepath.Glob("./_meta/test/*.json")
	assert.NoError(t, err)

	for _, f := range files {
		input, err := ioutil.ReadFile(f)
		assert.NoError(t, err)

		_, err = eventMapping(input)
		assert.NoError(t, err, f)
	}
}
