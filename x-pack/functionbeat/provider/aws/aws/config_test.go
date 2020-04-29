// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemSizeFactor64(t *testing.T) {
	t.Run("human format", func(t *testing.T) {
		t.Run("value is a factor of 64", func(t *testing.T) {
			v := MemSizeFactor64(0)
			err := v.Unpack("128MiB")
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, MemSizeFactor64(128*1024*1024), v)
		})
	})

	t.Run("raw value", func(t *testing.T) {
		t.Run("value is a factor of 64", func(t *testing.T) {
			v := MemSizeFactor64(0)
			err := v.Unpack(fmt.Sprintf("%d", 128*1024*1024))
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, MemSizeFactor64(128*1024*1024), v)
		})

		t.Run("value is not a factor of 64", func(t *testing.T) {
			v := MemSizeFactor64(0)
			err := v.Unpack("121")
			assert.Error(t, err)
		})
	})

	t.Run("returns the value in megabyte", func(t *testing.T) {
		v := MemSizeFactor64(128 * 1024 * 1024)
		assert.Equal(t, 128, v.Megabytes())
	})
}
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
