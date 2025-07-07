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
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest/observer"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConfigValidate(t *testing.T) {
	t.Run("paths cannot be empty", func(t *testing.T) {
		c := config{Paths: []string{}}
		err := c.Validate()
		assert.Error(t, err)
	})

	t.Run("take_over requires ID", func(t *testing.T) {
		c := config{
			Paths:    []string{"/foo/bar"},
			TakeOver: takeOverConfig{Enabled: true},
		}
		err := c.Validate()
		assert.Error(t, err, "take_over.enabled can only be true if ID is set")
	})

	t.Run("take_over works with ID set", func(t *testing.T) {
		c := config{
			Paths:    []string{"/foo/bar"},
			ID:       "some id",
			TakeOver: takeOverConfig{Enabled: true},
		}
		err := c.Validate()
		assert.NoError(t, err)
	})
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
				assert.Equal(t, strings.Count(err.Error(), "duplicated-id-1"), 1, "each IDs should appear only once")
				assert.Equal(t, strings.Count(err.Error(), "duplicated-id-2"), 1, "each IDs should appear only once")

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
			err := logp.DevelopmentSetup(logp.ToObserverOutput())
			require.NoError(t, err, "could not setup log for development")

			err = ValidateInputIDs(inputs, logp.L())
			tc.assertErr(t, err)
			if tc.assertLogs != nil {
				tc.assertLogs(t, logp.ObserverLogs())
			}
		})
	}
}

func TestTakeOverCfg(t *testing.T) {
	testCases := map[string]struct {
		cfgYAML     string
		takeOverCfg takeOverConfig
		expectErr   bool
	}{
		"legacy mode enabled": {
			cfgYAML: `
              take_over: true`,
			takeOverCfg: takeOverConfig{
				Enabled: true,
			},
		},
		"legacy mode disabled": {
			cfgYAML: `
              take_over: false`,
			takeOverCfg: takeOverConfig{
				Enabled: false,
			},
		},
		"new mode enabled": {
			cfgYAML: `
              take_over:
                enabled: true`,
			takeOverCfg: takeOverConfig{
				Enabled: true,
			},
		},
		"new mode disabled": {
			cfgYAML: `
              take_over:
                enabled: false`,
			takeOverCfg: takeOverConfig{
				Enabled: false,
			},
		},
		"new mode with IDs": {
			cfgYAML: `
              take_over:
                enabled: false
                from_ids: ["foo", "bar"]`,
			takeOverCfg: takeOverConfig{
				Enabled: false,
				FromIDs: []string{"foo", "bar"},
			},
		},
		"take_over not defined": {
			cfgYAML:   "",
			expectErr: false,
		},
		"invalid config": {
			cfgYAML:   "take_over.enabled: 42",
			expectErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// It is required to have 'paths' set, so set it here for all tests
			cfg := conf.MustNewConfigFrom(tc.cfgYAML)
			err := cfg.SetChild("paths", -1, conf.MustNewConfigFrom(`["foo"]`))
			if err != nil {
				t.Fatalf("cannot set 'paths' in config: %s", err)
			}

			_, inp, err := configure(cfg, logp.NewNopLogger())
			if tc.expectErr {
				require.Error(t, err, "expecting error when parsing config")
				require.Nil(t, inp, "returned filestream must be nil on error")
				return
			} else {
				require.NoError(t, err, "expecting the config to be successfully parsed")
			}

			f, ok := inp.(*filestream)
			if !ok {
				t.Fatalf("expecting type filestream, got %T", inp)
			}

			assert.Equal(t, tc.takeOverCfg, f.takeOver, "take over config does not match")
		})
	}
}
