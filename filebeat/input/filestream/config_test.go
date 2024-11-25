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
		require.Error(t, err)
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
