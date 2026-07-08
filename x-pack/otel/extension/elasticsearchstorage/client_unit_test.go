// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// This file was contributed to by generative AI

package elasticsearchstorage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckKey(t *testing.T) {
	assert.ErrorIs(t, checkKey(""), errEmptyKey)
	assert.NoError(t, checkKey("cursor"))
	assert.NoError(t, checkKey("a/b c+d"))
}
