// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/keystore"
)

func TestConfigsMapping(t *testing.T) {
	config, _ := common.NewConfigFrom(map[string]interface{}{
		"correct": "config",
	})

	tests := []struct {
		mapping  string
		event    bus.Event
		expected []*common.Config
	}{
		// No match
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - type: config1`,
			event: bus.Event{
				"foo": "no match",
			},
			expected: nil,
		},
		// Match config
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - correct: config`,
			event: bus.Event{
				"foo": 3,
			},
			expected: []*common.Config{config},
		},
		// No condition
		{
			mapping: `
- config:
    - correct: config`,
			event: bus.Event{
				"foo": 3,
			},
			expected: []*common.Config{config},
		},
	}

	for _, test := range tests {
		var mappings MapperSettings
		config, err := common.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings)
		if err != nil {
			t.Fatal(err)
		}

		res := mapper.GetConfig(test.event)
		assert.Equal(t, test.expected, res)
	}
}

func TestConfigsMappingKeystore(t *testing.T) {
	secret := "mapping_secret"
	//expected config
	config, _ := common.NewConfigFrom(map[string]interface{}{
		"correct":  "config",
		"password": secret,
	})

	path := getTemporaryKeystoreFile()
	defer os.Remove(path)
	// store the secret
	keystore := createAnExistingKeystore(path, secret)

	tests := []struct {
		mapping  string
		event    bus.Event
		expected []*common.Config
	}{
		// Match config
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - correct: config
    password: "${PASSWORD}"`,
			event: bus.Event{
				"foo":      3,
				"keystore": keystore,
			},
			expected: []*common.Config{config},
		},
	}

	for _, test := range tests {
		var mappings MapperSettings
		config, err := common.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings)
		if err != nil {
			t.Fatal(err)
		}

		res := mapper.GetConfig(test.event)
		assert.Equal(t, test.expected, res)
	}
}

func TestNilConditionConfig(t *testing.T) {
	var mappings MapperSettings
	data := `
- config:
    - type: config1`
	config, err := common.NewConfigWithYAML([]byte(data), "")
	if err != nil {
		t.Fatal(err)
	}

	if err := config.Unpack(&mappings); err != nil {
		t.Fatal(err)
	}

	_, err = NewConfigMapper(mappings)
	assert.NoError(t, err)
	assert.Nil(t, mappings[0].ConditionConfig)
}

// create a keystore with an existing key
/// `PASSWORD` with the value of `secret` variable.
func createAnExistingKeystore(path string, secret string) keystore.Keystore {
	keyStore, err := keystore.NewFileKeystore(path)
	// Fail fast in the test suite
	if err != nil {
		panic(err)
	}

	writableKeystore, err := keystore.AsWritableKeystore(keyStore)
	if err != nil {
		panic(err)
	}

	writableKeystore.Store("PASSWORD", []byte(secret))
	writableKeystore.Save()
	return keyStore
}

// create a temporary file on disk to save the keystore.
func getTemporaryKeystoreFile() string {
	path, err := ioutils.TempDir("", "testing")
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "keystore")
}
