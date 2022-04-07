// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/libbeat/common"
)

func TestNameMustBeUnique(t *testing.T) {
	tests := []struct {
		name string
		v    map[string]interface{}
		err  bool
	}{
		{
			name: "not unique names",
			err:  true,
			v: map[string]interface{}{
				"functions": []map[string]interface{}{
					map[string]interface{}{
						"enabled": true,
						"type":    "cloudwatchlogs",
						"name":    "ok",
					},
					map[string]interface{}{
						"enabled": true,
						"type":    "cloudwatchlogs",
						"name":    "ok",
					},
				},
			},
		},
		{
			name: "not unique names but duplicate is disabled",
			err:  false,
			v: map[string]interface{}{
				"functions": []map[string]interface{}{
					map[string]interface{}{
						"enabled": true,
						"type":    "cloudwatchlogs",
						"name":    "ok",
					},
					map[string]interface{}{
						"enabled": false,
						"type":    "cloudwatchlogs",
						"name":    "ok",
					},
				},
			},
		},
		{
			name: "name are uniques",
			err:  false,
			v: map[string]interface{}{
				"functions": []map[string]interface{}{
					map[string]interface{}{
						"enabled": true,
						"type":    "cloudwatchlogs",
						"name":    "ok",
					},
					map[string]interface{}{
						"enabled": true,
						"type":    "cloudwatchlogs",
						"name":    "another",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(test.v)
			if !assert.NoError(t, err) {
				return
			}
			provider := ProviderConfig{}

			err = cfg.Unpack(&provider)
			if test.err == true {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestFunctionName(t *testing.T) {
	t.Run("valid function name", func(t *testing.T) {
		f := functionName("")
		err := f.Unpack("hello-world")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, functionName("hello-world"), f)
	})

	t.Run("valid function name: length of 30 chars", func(t *testing.T) {
		f := functionName("")
		err := f.Unpack("something-which-is--30--chars")
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, functionName("something-which-is--30--chars"), f)
	})

	t.Run("invalid function name", func(t *testing.T) {
		f := functionName("")
		err := f.Unpack("hello world")
		assert.Error(t, err)
	})

	t.Run("invalid function name: length", func(t *testing.T) {
		f := functionName("")
		err := f.Unpack("something-which-is-greater-than-thirty-characters")
		assert.Error(t, err)
	})
}
