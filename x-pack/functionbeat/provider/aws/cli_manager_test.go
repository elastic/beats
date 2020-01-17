// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestChecksum(t *testing.T) {
	t.Run("same bytes content return the same key", func(t *testing.T) {
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, checksum(content), checksum(content))
	})

	t.Run("different bytes return a different key", func(t *testing.T) {
		content, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		other, err := common.RandomBytes(100)
		if !assert.NoError(t, err) {
			return
		}

		assert.NotEqual(t, checksum(content), checksum(other))
	})
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		title     string
		candidate string
		chars     string
		expected  string
	}{
		{
			title:     "when the string contains invalid chars",
			candidate: "/var/log-alpha/tmp:ok",
			expected:  "varlogalphatmpok",
		},
		{
			title:     "when we have an empty string",
			candidate: "",
			expected:  "",
		},
		{
			title:     "when we don't have any invalid chars",
			candidate: "hello",
			expected:  "hello",
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			assert.Equal(t, test.expected, normalizeResourceName(test.candidate))
		})
	}
}
