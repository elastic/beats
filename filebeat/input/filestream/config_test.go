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

package filestream

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Run("paths cannot be empty", func(t *testing.T) {
		c := config{Paths: []string{}}
		err := c.Validate()
		require.Error(t, err)
	})
}
<<<<<<< HEAD
=======

func TestNormalizeConfig(t *testing.T) {
	tcs := []struct {
		name        string
		cfg         map[string]interface{}
		wantEnabled bool
	}{
		{
			name: "path identity disables prospector.scanner.fingerprint by default",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{"path": nil},
			},
			wantEnabled: false,
		},
		{
			name: "native identity disables scanner fingerprint by default",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{"native": nil},
			},
			wantEnabled: false,
		},
		{
			name: "explicit scanner fingerprint true is preserved",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{"path": nil},
				"prospector": map[string]interface{}{
					"scanner": map[string]interface{}{
						"fingerprint": map[string]interface{}{"enabled": true},
					},
				},
			},
			wantEnabled: true,
		},
		{
			name: "explicit scanner fingerprint false is preserved",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{"fingerprint": nil},
				"prospector": map[string]interface{}{
					"scanner": map[string]interface{}{
						"fingerprint": map[string]interface{}{"enabled": false},
					},
				},
			},
			wantEnabled: false,
		},
		{
			name: "fingerprint identity keeps default scanner fingerprint",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{"fingerprint": nil},
			},
			wantEnabled: true,
		},
		{
			name: "non-fingerprint inode_marker disables scanner fingerprint by default",
			cfg: map[string]interface{}{
				"file_identity": map[string]interface{}{
					"inode_marker": map[string]interface{}{"path": "/logs/.filebeat-marker"},
				},
			},
			wantEnabled: false,
		},
		{
			name:        "no file_identity keeps default scanner fingerprint",
			cfg:         map[string]interface{}{},
			wantEnabled: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			c := defaultConfig()
			cfg := map[string]interface{}{
				"paths": []string{"/tmp/logs/*.log"},
			}
			for key, value := range tc.cfg {
				cfg[key] = value
			}
			raw := conf.MustNewConfigFrom(cfg)
			require.NoError(t, raw.Unpack(&c))

			err := normalizeConfig(raw, &c)
			require.NoError(t, err)

			assert.Equal(t, tc.wantEnabled, c.FileWatcher.Scanner.Fingerprint.Enabled)
		})
	}
}

func TestValidateInputIDs(t *testing.T) {
	tcs := []struct {
		name       string
		cfg        []string
		assertErr  func(t *testing.T, err error)
		assertLogs func(t *testing.T, buff *observer.ObservedLogs)
	}{
		{
			name: "empty config",
			cfg:  []string{""},
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err, "empty config should not return an error")
			},
		},
		{
			name: "one empty ID is allowed",
			cfg: []string{`
type: filestream
`, `
type: filestream
id: some-id-1
`, `
type: filestream
id: some-id-2
`,
			},
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err, "one empty id is allowed")
			},
		},
		{
			name: "duplicated empty ID",
			cfg: []string{`
type: filestream
paths:
  - "/tmp/empty-1"
`, `
type: filestream
paths:
  - "/tmp/empty-2"
`, `
type: filestream
id: unique-id-1
`, `
type: filestream
id: unique-id-2
`, `
type: filestream
id: unique-ID
`,
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, `filestream inputs with duplicated IDs: ""`)

			},
			assertLogs: func(t *testing.T, obs *observer.ObservedLogs) {
				want := `[{"paths":["/tmp/empty-1"],"type":"filestream"},{"paths":["/tmp/empty-2"],"type":"filestream"}]`

				logs := obs.TakeAll()
				require.Len(t, logs, 1, "there should be only one log entry")

				got, err := json.Marshal(logs[0].ContextMap()["inputs"])
				require.NoError(t, err, "could not marshal duplicated IDs inputs")
				assert.Equal(t, want, string(got))
			},
		}, {
			name: "duplicated IDs",
			cfg: []string{`
type: filestream
id: duplicated-id-1
`, `
type: filestream
id: duplicated-id-1
`, `
type: filestream
id: duplicated-id-2
`, `
type: filestream
id: duplicated-id-2
`, `
type: filestream
id: duplicated-id-2
`, `
type: filestream
id: unique-ID
`,
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "filestream inputs with duplicated IDs")
				assert.ErrorContains(t, err, "duplicated-id-1")
				assert.ErrorContains(t, err, "duplicated-id-2")
				assert.Equal(t, 1, strings.Count(err.Error(), "duplicated-id-1"), "each IDs should appear only once")
				assert.Equal(t, 1, strings.Count(err.Error(), "duplicated-id-2"), "each IDs should appear only once")

			},
			assertLogs: func(t *testing.T, obs *observer.ObservedLogs) {
				want := `[{"id":"duplicated-id-1","type":"filestream"},{"id":"duplicated-id-1","type":"filestream"},{"id":"duplicated-id-2","type":"filestream"},{"id":"duplicated-id-2","type":"filestream"},{"id":"duplicated-id-2","type":"filestream"}]`

				logs := obs.TakeAll()
				require.Len(t, logs, 1, "there should be only one log entry")

				got, err := json.Marshal(logs[0].ContextMap()["inputs"])
				require.NoError(t, err, "could not marshal duplicated IDs inputs")
				assert.Equal(t, want, string(got))
			},
		},
		{
			name: "duplicated IDs and empty ID",
			cfg: []string{`
type: filestream
`, `
type: filestream
`, `
type: filestream
id: duplicated-id-1
`, `
type: filestream
id: duplicated-id-1
`, `
type: filestream
id: duplicated-id-2
`, `
type: filestream
id: duplicated-id-2
`, `
type: filestream
id: unique-ID
`,
			},
			assertErr: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "filestream inputs with duplicated IDs")
			},
			assertLogs: func(t *testing.T, obs *observer.ObservedLogs) {
				want := `[{"type":"filestream"},{"type":"filestream"},{"id":"duplicated-id-1","type":"filestream"},{"id":"duplicated-id-1","type":"filestream"},{"id":"duplicated-id-2","type":"filestream"},{"id":"duplicated-id-2","type":"filestream"}]`

				logs := obs.TakeAll()
				require.Len(t, logs, 1, "there should be only one log entry")

				got, err := json.Marshal(logs[0].ContextMap()["inputs"])
				require.NoError(t, err, "could not marshal duplicated IDs inputs")
				assert.Equal(t, want, string(got))

			},
		},
		{
			name: "only unique IDs",
			cfg: []string{`
type: filestream
id: unique-id-1
`, `
type: filestream
id: unique-id-2
`, `
type: filestream
id: unique-id-3
`,
			},
			assertErr: func(t *testing.T, err error) {
				assert.NoError(t, err, "only unique IDs should not return an error")
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var inputs []*conf.C
			for _, c := range tc.cfg {
				cfg, err := conf.NewConfigFrom(c)
				require.NoError(t, err, "could not create input configuration")
				inputs = append(inputs, cfg)
			}

			logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "")
			err := ValidateInputIDs(inputs, logger)
			tc.assertErr(t, err)
			if tc.assertLogs != nil {
				tc.assertLogs(t, observedLogs)
			}
		})
	}
}
>>>>>>> 134036433 (golangci: Enable testifylint for data-plane owned code (#49008))
