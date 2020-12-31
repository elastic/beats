// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToSlice(t *testing.T) {
	assert.Equal(t, []string{"hello", "world", "bye"}, StringToSlice("hello, world,bye"))
	assert.Equal(t, []string{"hello"}, StringToSlice("hello"))
	assert.Equal(t, []string{}, StringToSlice(""))
}
