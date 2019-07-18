// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBucket(t *testing.T) {
	t.Run("valid bucket name", func(t *testing.T) {
		b := bucket("")
		err := b.Unpack("hello-bucket")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, bucket("hello-bucket"), b)
	})

	t.Run("too long", func(t *testing.T) {
		b := bucket("")
		err := b.Unpack("hello-bucket-abc12345566789012345678901234567890123456789012345678901234567890")
		assert.Error(t, err)
	})

	t.Run("too short", func(t *testing.T) {
		b := bucket("")
		err := b.Unpack("he")
		assert.Error(t, err)
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
		{
			title:     "when the string contains underscore",
			candidate: "/var/log-alpha/tmp:ok_moreok",
			expected:  "varlogalphatmpokmoreok",
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			assert.Equal(t, test.expected, NormalizeResourceName(test.candidate))
		})
	}
}
