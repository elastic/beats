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

package beater

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inputEntry struct {
	ID string `config:"id"`
}

type takeoverTestStateStore struct {
	registry *statestore.Registry
}

func (s *takeoverTestStateStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("")
}

func (s *takeoverTestStateStore) CleanupInterval() time.Duration {
	return time.Second
}

func TestFetchInputConfiguration(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "config1.yml"), []byte(`
- type: filestream
  id: external-1
  paths:
    - "/some"
- type: filestream
  id: external-2
  paths:
    - "/another"
`), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "config2.yml.disabled"), []byte(`
- type: filestream
  id: disabled
  paths:
    - "/some"
`), 0644)
	require.NoError(t, err)

	cases := []struct {
		name       string
		configFile string
		expected   []inputEntry
	}{
		{
			name: "loads mixed configuration",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.yml
filebeat.inputs:
  - type: filestream
    id: internal
    paths:
      - "/another"
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "internal",
				},
				{
					ID: "external-1",
				},
				{
					ID: "external-2",
				},
			},
		},
		{
			name: "loads only internal configuration",
			configFile: `
filebeat.inputs:
  - type: filestream
    id: internal
    paths:
      - "/another"
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "internal",
				},
			},
		},
		{
			name: "loads only external configuration",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.yml
output.console:
  enabled: true
`,
			expected: []inputEntry{
				{
					ID: "external-1",
				},
				{
					ID: "external-2",
				},
			},
		},
		{
			name: "loads nothing",
			configFile: `
filebeat.config.inputs:
  enabled: true
  path: ` + dir + `/*.nothing
output.console:
  enabled: true
`,
			expected: []inputEntry{},
		},
	}

	logger := logptest.NewTestingLogger(t, "")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rawConfig, err := conf.NewConfigFrom(tc.configFile)
			require.NoError(t, err)

			cfg := struct {
				Filebeat config.Config `config:"filebeat"`
			}{
				Filebeat: config.DefaultConfig,
			}
			err = rawConfig.Unpack(&cfg)
			require.NoError(t, err)

			inputs, err := fetchInputConfiguration(&cfg.Filebeat, logger, &paths.Path{})
			require.NoError(t, err)

			actual := []inputEntry{}

			for _, i := range inputs {
				var entry inputEntry
				err := i.Unpack(&entry)
				require.NoError(t, err)
				actual = append(actual, entry)
			}

			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestFetchInputConfigurationResolvesRelativeExternalPath(t *testing.T) {
	filestreamExternalCfg := `
- type: filestream
  id: external
  paths:
    - /var/log/external.log
`

	rawCfgYml := `
filebeat.config.inputs:
  path: inputs.d/*.yml
`

	configDir := t.TempDir()
	inputsDir := filepath.Join(configDir, "inputs.d")
	require.NoError(t, os.Mkdir(inputsDir, 0755), "external input configuration directory must be created")
	require.NoError(
		t,
		os.WriteFile(filepath.Join(inputsDir, "filestream.yml"),
			[]byte(filestreamExternalCfg), 0644),
		"external input configuration must be written",
	)

	rawConfig, err := conf.NewConfigFrom(rawCfgYml)
	require.NoError(t, err, "Filebeat configuration must parse")

	cfg := struct {
		Filebeat config.Config `config:"filebeat"`
	}{
		Filebeat: config.DefaultConfig,
	}
	require.NoError(t, rawConfig.Unpack(&cfg), "Filebeat configuration must unpack")

	inputs, err := fetchInputConfiguration(
		&cfg.Filebeat,
		logptest.NewTestingLogger(t, ""),
		&paths.Path{Config: configDir},
	)
	require.NoError(t, err, "relative external input configuration path must load from path.config")
	require.Len(t, inputs, 1, "exactly one external input must be loaded")

	var input inputEntry
	require.NoError(t, inputs[0].Unpack(&input), "external input configuration must unpack")
	require.Equal(t, "external", input.ID, "external input loaded from path.config must be returned")
}

func TestFetchInputConfigurationSkipsDisabledDynamicInputs(t *testing.T) {
	filestreamExternalCfg := `
- type: filestream
  id: external
  paths:
    - /var/log/external.log
`

	rawCfgYml := `
filebeat.config.inputs:
  enabled: false
  path: inputs.d/*.yml
`

	configDir := t.TempDir()
	inputsDir := filepath.Join(configDir, "inputs.d")
	require.NoError(t, os.Mkdir(inputsDir, 0755), "external input configuration directory must be created")
	require.NoError(
		t,
		os.WriteFile(filepath.Join(inputsDir, "filestream.yml"),
			[]byte(filestreamExternalCfg), 0644),
		"external input configuration must be written",
	)

	rawConfig, err := conf.NewConfigFrom(rawCfgYml)
	require.NoError(t, err, "Filebeat configuration must parse")

	cfg := struct {
		Filebeat config.Config `config:"filebeat"`
	}{
		Filebeat: config.DefaultConfig,
	}
	require.NoError(t, rawConfig.Unpack(&cfg), "Filebeat configuration must unpack")

	inputs, err := fetchInputConfiguration(&cfg.Filebeat, logptest.NewTestingLogger(t, ""), &paths.Path{Config: configDir})
	require.NoError(t, err, "disabled external input configuration must not be loaded")
	require.Empty(t, inputs, "disabled external input configuration must not provide takeover inputs")
}

func TestProcessLogInputTakeOverSkipsDisabledStaticInput(t *testing.T) {
	filestreamCfg := `
type: filestream
id: disabled-filestream
enabled: false
take_over: true
paths:
  - /var/log/legacy.log
`
	input, err := conf.NewConfigFrom(filestreamCfg)
	require.NoError(t, err, "disabled filestream configuration must parse")

	cfg := config.DefaultConfig
	cfg.Inputs = []*conf.C{input}

	backend := storetest.NewMemoryStoreBackend()
	stateStore := &takeoverTestStateStore{registry: statestore.NewRegistry(backend)}
	t.Cleanup(func() {
		require.NoError(t, stateStore.registry.Close(), "state store registry must close")
	})

	store, err := stateStore.StoreFor("")
	require.NoError(t, err, "state store must be available")
	require.NoError(
		t,
		store.Set(
			"filebeat::logs::native::1-1",
			map[string]any{
				"source": "/var/log/legacy.log",
				"offset": 1,
			}),
		"legacy log input state must be stored",
	)
	require.NoError(t, store.Close(), "state store must close before takeover")

	err = processLogInputTakeOver(
		logptest.NewTestingLogger(t, ""),
		stateStore,
		&cfg,
		&paths.Path{Data: t.TempDir()},
	)
	require.NoError(t, err, "disabled filestream must not trigger state takeover")

	states := backend.Stores[""].Table
	assert.Contains(
		t,
		states,
		"filebeat::logs::native::1-1",
		"disabled filestream must leave legacy state unchanged",
	)
	assert.NotContains(
		t,
		states,
		"filestream::disabled-filestream::native::1-1",
		"disabled filestream must not create filestream state",
	)
}
