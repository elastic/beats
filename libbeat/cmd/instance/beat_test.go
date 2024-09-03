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

//go:build !integration

package instance

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/queue/memqueue"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-ucfg/yaml"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInstance(t *testing.T) {
	b, err := NewBeat("testbeat", "testidx", "0.9", false, nil)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testidx", b.Info.IndexPrefix)
	assert.Equal(t, "0.9", b.Info.Version)

	// UUID4 should be 36 chars long
	assert.Equal(t, 16, len(b.Info.ID))
	assert.Equal(t, 36, len(b.Info.ID.String()))

	// indexPrefix set to name if empty
	b, err = NewBeat("testbeat", "", "0.9", false, nil)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "testbeat", b.Info.Beat)
	assert.Equal(t, "testbeat", b.Info.IndexPrefix)

}

func TestNewInstanceUUID(t *testing.T) {
	b, err := NewBeat("testbeat", "", "0.9", false, nil)
	if err != nil {
		panic(err)
	}

	// Make sure the ID's are different
	differentUUID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating ID: %v", err)
	}
	assert.NotEqual(t, b.Info.ID, differentUUID)
}

func TestInitKibanaConfig(t *testing.T) {
	b, err := NewBeat("filebeat", "testidx", "0.9", false, nil)
	if err != nil {
		panic(err)
	}

	assert.Equal(t, "filebeat", b.Info.Beat)
	assert.Equal(t, "testidx", b.Info.IndexPrefix)
	assert.Equal(t, "0.9", b.Info.Version)

	const configPath = "../test/filebeat_test.yml"

	// Ensure that the config has owner-exclusive write permissions.
	// This is necessary on some systems which have a default umask
	// of 0o002, meaning that files are checked out by git with mode
	// 0o664. This would cause cfgfile.Load to fail.
	err = os.Chmod(configPath, 0o644)
	assert.NoError(t, err)

	cfg, err := cfgfile.Load(configPath, nil)
	assert.NoError(t, err)
	err = cfg.Unpack(&b.Config)
	assert.NoError(t, err)

	kibanaConfig := InitKibanaConfig(b.Config)
	username, err := kibanaConfig.String("username", -1)
	assert.NoError(t, err)
	password, err := kibanaConfig.String("password", -1)
	assert.NoError(t, err)
	api_key, err := kibanaConfig.String("api_key", -1)
	assert.NoError(t, err)
	protocol, err := kibanaConfig.String("protocol", -1)
	assert.NoError(t, err)
	host, err := kibanaConfig.String("host", -1)
	assert.NoError(t, err)

	assert.Equal(t, "elastic-test-username", username)
	assert.Equal(t, "elastic-test-password", password)
	assert.Equal(t, "elastic-test-api-key", api_key)
	assert.Equal(t, "https", protocol)
	assert.Equal(t, "127.0.0.1:5601", host)
}

func TestEmptyMetaJson(t *testing.T) {
	b, err := NewBeat("filebeat", "testidx", "0.9", false, nil)
	if err != nil {
		panic(err)
	}

	// prepare empty meta file
	metaFile, err := ioutil.TempFile("../test", "meta.json")
	assert.Equal(t, nil, err, "Unable to create temporary meta file")

	metaPath := metaFile.Name()
	metaFile.Close()
	defer os.Remove(metaPath)

	// load metadata
	err = b.loadMeta(metaPath)

	assert.Equal(t, nil, err, "Unable to load meta file properly")
	assert.NotEqual(t, uuid.Nil, b.Info.ID, "Beats UUID is not set")
}

func TestMetaJsonWithTimestamp(t *testing.T) {
	firstBeat, err := NewBeat("filebeat", "testidx", "0.9", false, nil)
	if err != nil {
		panic(err)
	}
	firstStart := firstBeat.Info.FirstStart

	metaFile, err := ioutil.TempFile("../test", "meta.json")
	assert.Equal(t, nil, err, "Unable to create temporary meta file")

	metaPath := metaFile.Name()
	metaFile.Close()
	defer os.Remove(metaPath)

	err = firstBeat.loadMeta(metaPath)
	assert.Equal(t, nil, err, "Unable to load meta file properly")

	secondBeat, err := NewBeat("filebeat", "testidx", "0.9", false, nil)
	if err != nil {
		panic(err)
	}
	assert.False(t, firstStart.Equal(secondBeat.Info.FirstStart), "Before meta.json is loaded, first start must be different")
	err = secondBeat.loadMeta(metaPath)
	require.NoError(t, err)

	assert.Equal(t, nil, err, "Unable to load meta file properly")
	assert.True(t, firstStart.Equal(secondBeat.Info.FirstStart), "Cannot load first start")
}

func TestSanitizeIPs(t *testing.T) {
	cases := []struct {
		name        string
		ips         []string
		expectedIPs []string
	}{
		{
			name: "does not change valid IPs",
			ips: []string{
				"127.0.0.1",
				"::1",
				"fe80::1",
				"fe80::6ca6:cdff:fe6a:4f59",
				"192.168.1.101",
			},
			expectedIPs: []string{
				"127.0.0.1",
				"::1",
				"fe80::1",
				"fe80::6ca6:cdff:fe6a:4f59",
				"192.168.1.101",
			},
		},
		{
			name: "cuts the masks",
			ips: []string{
				"127.0.0.1/8",
				"::1/128",
				"fe80::1/64",
				"fe80::6ca6:cdff:fe6a:4f59/64",
				"192.168.1.101/24",
			},
			expectedIPs: []string{
				"127.0.0.1",
				"::1",
				"fe80::1",
				"fe80::6ca6:cdff:fe6a:4f59",
				"192.168.1.101",
			},
		},
		{
			name: "excludes invalid IPs",
			ips: []string{
				"",
				"fe80::6ca6:cdff:fe6a:4f59",
				"invalid",
				"192.168.1.101",
			},
			expectedIPs: []string{
				"fe80::6ca6:cdff:fe6a:4f59",
				"192.168.1.101",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expectedIPs, sanitizeIPs(tc.ips))
		})
	}
}

func TestReloader(t *testing.T) {
	t.Run("updates the output configuration on the beat", func(t *testing.T) {
		b, err := NewBeat("testbeat", "testidx", "0.9", false, nil)
		require.NoError(t, err)

		cfg := `
elasticsearch:
  hosts: ["https://127.0.0.1:9200"]
  username: "elastic"
  allow_older_versions: false
`
		c, err := config.NewConfigWithYAML([]byte(cfg), cfg)
		require.NoError(t, err)
		outCfg, err := c.Child("elasticsearch", -1)
		require.NoError(t, err)

		update := &reload.ConfigWithMeta{Config: c}
		m := &outputReloaderMock{}
		reloader := b.makeOutputReloader(m)

		require.False(t, b.Config.Output.IsSet(), "the output should not be set yet")
		require.True(t, b.isConnectionToOlderVersionAllowed(), "allow_older_versions flag should be true from 8.11")
		err = reloader.Reload(update)
		require.NoError(t, err)
		require.True(t, b.Config.Output.IsSet(), "now the output should be set")
		require.Equal(t, outCfg, b.Config.Output.Config())
		require.Same(t, c, m.cfg.Config)
		require.False(t, b.isConnectionToOlderVersionAllowed(), "allow_older_versions flag should now be set to false")
	})
}

type outputReloaderMock struct {
	cfg *reload.ConfigWithMeta
}

func (r *outputReloaderMock) Reload(
	cfg *reload.ConfigWithMeta,
	factory func(o outputs.Observer, cfg config.Namespace) (outputs.Group, error),
) error {
	r.cfg = cfg
	return nil
}

func TestPromoteOutputQueueSettings(t *testing.T) {
	tests := map[string]struct {
		input     []byte
		memEvents int
	}{
		"blank": {
			input:     []byte(""),
			memEvents: 3200,
		},
		"defaults": {
			input: []byte(`
name: mockbeat
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
`),
			memEvents: 3200,
		},
		"topLevelQueue": {
			input: []byte(`
name: mockbeat
queue:
  mem:
    events: 8096
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
`),
			memEvents: 8096,
		},
		"outputLevelQueue": {
			input: []byte(`
name: mockbeat
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
    queue:
      mem:
        events: 8096
`),
			memEvents: 8096,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := yaml.NewConfig(tc.input)
			require.NoError(t, err)

			config := beatConfig{}
			err = cfg.Unpack(&config)
			require.NoError(t, err)

			err = promoteOutputQueueSettings(&config)
			require.NoError(t, err)

			ms, err := memqueue.SettingsForUserConfig(config.Pipeline.Queue.Config())
			require.NoError(t, err)
			require.Equalf(t, tc.memEvents, ms.Events, "config was: %v", config.Pipeline.Queue.Config())
		})
	}
}

func TestValidateBeatConfig(t *testing.T) {
	tests := map[string]struct {
		input                 []byte
		expectValidationError string
	}{
		"blank": {
			input:                 []byte(""),
			expectValidationError: "",
		},
		"defaults": {
			input: []byte(`
name: mockbeat
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
`),
			expectValidationError: "",
		},
		"topAndOutputLevelQueue": {
			input: []byte(`
name: mockbeat
queue:
  mem:
    events: 2048
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
    queue:
      mem:
        events: 8096
`),
			expectValidationError: "top level queue and output level queue settings defined, only one is allowed accessing config",
		},
		"managementTopLevelDiskQueue": {
			input: []byte(`
name: mockbeat
management:
  enabled: true
queue:
  disk:
    max_size: 1G
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
`),
			expectValidationError: "disk queue is not supported when management is enabled accessing config",
		},
		"managementOutputLevelDiskQueue": {
			input: []byte(`
name: mockbeat
management:
  enabled: true
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
    queue:
      disk:
        max_size: 1G
`),
			expectValidationError: "disk queue is not supported when management is enabled accessing config",
		},
		"managementFalseOutputLevelDiskQueue": {
			input: []byte(`
name: mockbeat
management:
  enabled: false
output:
  elasticsearch:
    hosts:
      - "localhost:9200"
    queue:
      disk:
        max_size: 1G
`),
			expectValidationError: "",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := yaml.NewConfig(tc.input)
			require.NoError(t, err)
			config := beatConfig{}
			err = cfg.Unpack(&config)
			if tc.expectValidationError != "" {
				require.Error(t, err)
				require.Equal(t, tc.expectValidationError, err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLogSystemInfo(t *testing.T) {
	tcs := []struct {
		name     string
		managed  bool
		assertFn func(*testing.T, *bytes.Buffer)
	}{
		{
			name: "managed mode", managed: true,
			assertFn: func(t *testing.T, b *bytes.Buffer) {
				assert.Empty(t, b, "logSystemInfo should not have produced any log")
			},
		},
		{
			name: "stand alone", managed: false,
			assertFn: func(t *testing.T, b *bytes.Buffer) {
				logs := b.String()
				assert.Contains(t, logs, "Beat info")
				assert.Contains(t, logs, "Build info")
				assert.Contains(t, logs, "Go runtime info")
			},
		},
	}
	log, buff := logp.NewInMemory("beat", logp.ConsoleEncoderConfig())
	log.WithOptions()

	b, err := NewBeat("testingbeat", "test-idx", "42", false, nil)
	require.NoError(t, err, "could not create beat")

	for _, tc := range tcs {
		buff.Reset()

		b.Manager = mockManager{enabled: tc.managed}
		b.logSystemInfo(log)

		tc.assertFn(t, buff)
	}
}

type mockManager struct {
	enabled bool
}

func (m mockManager) AgentInfo() client.AgentInfo         { return client.AgentInfo{} }
func (m mockManager) CheckRawConfig(cfg *config.C) error  { return nil }
func (m mockManager) Enabled() bool                       { return m.enabled }
func (m mockManager) RegisterAction(action client.Action) {}
func (m mockManager) RegisterDiagnosticHook(name, description, filename, contentType string, hook client.DiagnosticHook) {
}
func (m mockManager) SetPayload(payload map[string]interface{})     {}
func (m mockManager) SetStopCallback(f func())                      {}
func (m mockManager) Start() error                                  { return nil }
func (m mockManager) Status() status.Status                         { return status.Status(-42) }
func (m mockManager) Stop()                                         {}
func (m mockManager) UnregisterAction(action client.Action)         {}
func (m mockManager) UpdateStatus(status status.Status, msg string) {}
