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

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-autodiscover/bus"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestConfigsMapping(t *testing.T) {
	logp.TestingSetup()

	config, _ := conf.NewConfigFrom(map[string]interface{}{
		"correct": "config",
	})

	configPorts, _ := conf.NewConfigFrom(map[string]interface{}{
		"correct": "config",
		"hosts":   [1]string{"1.2.3.4:8080"},
	})

	const envValue = "valuefromenv"
	configFromEnv, _ := conf.NewConfigFrom(map[string]interface{}{
		"correct": envValue,
	})

	os.Setenv("CONFIGS_MAPPING_TESTENV", envValue)

	tests := []struct {
		mapping  string
		event    bus.Event
		expected []*conf.C
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
			expected: []*conf.C{config},
		},
		// No condition
		{
			mapping: `
- config:
    - correct: config`,
			event: bus.Event{
				"foo": 3,
			},
			expected: []*conf.C{config},
		},
		// No condition, value from environment
		{
			mapping: `
- config:
    - correct: ${CONFIGS_MAPPING_TESTENV}`,
			event: bus.Event{
				"foo": 3,
			},
			expected: []*conf.C{configFromEnv},
		},
		// Match config and replace data.host and data.ports.<name> properly
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - correct: config
    hosts: ["${data.host}:${data.ports.web}"]`,
			event: bus.Event{
				"foo":  3,
				"host": "1.2.3.4",
				"ports": mapstr.M{
					"web": 8080,
				},
			},
			expected: []*conf.C{configPorts},
		},
		// Match config and replace data.host and data.port properly
		{
			mapping: `
- condition.equals:
    foo: 3
  config:
  - correct: config
    hosts: ["${data.host}:${data.port}"]`,
			event: bus.Event{
				"foo":  3,
				"host": "1.2.3.4",
				"port": 8080,
			},
			expected: []*conf.C{configPorts},
		},
		// Missing variable, config is not generated
		{
			mapping: `
- config:
  - module: something
    hosts: ["${not.exists.host}"]`,
			event: bus.Event{
				"host": "1.2.3.4",
			},
			expected: nil,
		},
	}

	logger := logptest.NewTestingLogger(t, "")

	for _, test := range tests {
		var mappings MapperSettings
		config, err := conf.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings, nil, nil, logger)
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
	config, _ := conf.NewConfigFrom(map[string]interface{}{
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
		expected []*conf.C
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
				"foo": 3,
			},
			expected: []*conf.C{config},
		},
	}

	logger := logptest.NewTestingLogger(t, "")

	for _, test := range tests {
		var mappings MapperSettings
		config, err := conf.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings, keystore, nil, logger)
		if err != nil {
			t.Fatal(err)
		}

		res := mapper.GetConfig(test.event)
		assert.Equal(t, test.expected, res)
	}
}

func TestConfigsMappingKeystoreProvider(t *testing.T) {
	secret := "mapping_provider_secret"
	//expected config
	config, _ := conf.NewConfigFrom(map[string]interface{}{
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
		expected []*conf.C
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
				"foo": 3,
			},
			expected: []*conf.C{config},
		},
	}

	keystoreProvider := newMockKeystoreProvider(secret)
	logger := logptest.NewTestingLogger(t, "")

	for _, test := range tests {
		var mappings MapperSettings
		config, err := conf.NewConfigWithYAML([]byte(test.mapping), "")
		if err != nil {
			t.Fatal(err)
		}

		if err := config.Unpack(&mappings); err != nil {
			t.Fatal(err)
		}

		mapper, err := NewConfigMapper(mappings, keystore, keystoreProvider, logger)
		if err != nil {
			t.Fatal(err)
		}

		res := mapper.GetConfig(test.event)
		assert.Equal(t, test.expected, res)
	}
}

type mockKeystore struct {
	secret string
}

func newMockKeystoreProvider(secret string) bus.KeystoreProvider {
	return &mockKeystore{secret}
}

// GetKeystore return a KubernetesSecretsKeystore if it already exists for a given namespace or creates a new one.
func (kr *mockKeystore) GetKeystore(event bus.Event) keystore.Keystore {
	path := getTemporaryKeystoreFile()
	defer os.Remove(path)
	// store the secret
	keystore := createAnExistingKeystore(path, kr.secret)
	return keystore
}

func TestNilConditionConfig(t *testing.T) {
	var mappings MapperSettings
	data := `
- config:
    - type: config1`
	config, err := conf.NewConfigWithYAML([]byte(data), "")
	if err != nil {
		t.Fatal(err)
	}

	if err := config.Unpack(&mappings); err != nil {
		t.Fatal(err)
	}

	_, err = NewConfigMapper(mappings, nil, nil, logptest.NewTestingLogger(t, ""))
	assert.NoError(t, err)
	assert.Nil(t, mappings[0].ConditionConfig)
}

// create a keystore with an existing key
// `PASSWORD` with the value of `secret` variable.
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
	path, err := os.MkdirTemp("", "testing")
	if err != nil {
		panic(err)
	}
	return filepath.Join(path, "keystore")
}
