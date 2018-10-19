// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestCodeKey(t *testing.T) {
	t.Run("same bytes content return the same key", func(t *testing.T) {
		name := "hello"
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, codeKey(name, content), codeKey(name, content))
	})

	t.Run("different bytes return a different key", func(t *testing.T) {
		name := "hello"
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		other, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.NotEqual(t, codeKey(name, content), codeKey(name, other))
	})
}
