// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/libbeat/common"
)

func EnsureBlacklistItems(t *testing.T) {
	// NOTE: We do not permit to configure the console or the file output with CM for security reason.
	c := defaultConfig()
	v, _ := c.Blacklist.Patterns["output"]
	assert.Equal(t, "console|file", v)
}

func TestMetadata(t *testing.T) {
	t.Run("test unpack from config", testUnpackMetadata)
	t.Run("test metadata yaml serialize", testMetadataYAMLSerialize)
}

func testUnpackMetadata(t *testing.T) {
	tests := map[string]struct {
		kv       map[string]interface{}
		expected metadata
	}{
		"support uint": {
			kv:       map[string]interface{}{"mykey": 42},
			expected: metadata{"mykey": uint64(42)},
		},
		"support int": {
			kv:       map[string]interface{}{"mykey": -42},
			expected: metadata{"mykey": int64(-42)},
		},
		"support float": {
			kv:       map[string]interface{}{"mykey": 0.5},
			expected: metadata{"mykey": 0.5},
		},
		"support signed float": {
			kv:       map[string]interface{}{"mykey": -0.5},
			expected: metadata{"mykey": -0.5},
		},
		"support string": {
			kv:       map[string]interface{}{"mykey": "myvalue"},
			expected: metadata{"mykey": "myvalue"},
		},
		"support []uint": {
			kv:       map[string]interface{}{"mykey": []interface{}{1, 2, 3}},
			expected: metadata{"mykey": []interface{}{uint64(1), uint64(2), uint64(3)}},
		},
		"support []int": {
			kv:       map[string]interface{}{"mykey": []interface{}{-1, -2, -3}},
			expected: metadata{"mykey": []interface{}{int64(-1), int64(-2), int64(-3)}},
		},
		"support []float": {
			kv:       map[string]interface{}{"mykey": []interface{}{0.1, 0.2, 0.3}},
			expected: metadata{"mykey": []interface{}{0.1, 0.2, 0.3}},
		},
		"support []string": {
			kv:       map[string]interface{}{"mykey": []interface{}{"hello", "world"}},
			expected: metadata{"mykey": []interface{}{"hello", "world"}},
		},
		"support []T where T is can be a int, uint, string or a float": {
			kv:       map[string]interface{}{"mykey": []interface{}{1, 0.1, "hello world", -10}},
			expected: metadata{"mykey": []interface{}{uint64(1), 0.1, "hello world", int64(-10)}},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			config, _ := common.NewConfigFrom(map[string]interface{}{"metadata": test.kv})
			c := struct {
				Metadata metadata `config:"metadata"`
			}{}

			err := config.Unpack(&c)
			if !assert.NoError(t, err) {
				return
			}

			assert.Equal(t, test.expected, c.Metadata)
		})
	}
}

func testMetadataYAMLSerialize(t *testing.T) {
	config := Config{
		Metadata: metadata{
			"tag":     []string{"hello", "world"},
			"ratio":   0.1,
			"version": 42,
		},
	}

	b, err := yaml.Marshal(&config)
	if !assert.NoError(t, err) {
		return
	}

	expected := map[string]interface{}{
		"tag":     []interface{}{"hello", "world"},
		"ratio":   0.1,
		"version": 42,
	}

	var unpack map[string]interface{}
	err = yaml.Unmarshal(b, &unpack)
	if !assert.NoError(t, err) {
		return
	}

	fmt.Println(string(b))

	meta, ok := unpack["metadata"]
	if !assert.True(t, ok, "no metadata key found in the map") {
		return
	}

	assert.True(t, reflect.DeepEqual(expected, meta))
}
