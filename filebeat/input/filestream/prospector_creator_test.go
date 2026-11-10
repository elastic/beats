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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

func TestCreateProspector(t *testing.T) {
	t.Run("SetIgnoreInactiveSince", func(t *testing.T) {
		testCases := map[string]struct {
			ignore_inactive_since string
		}{
			"ignore_inactive_since set to since_last_start": {
				ignore_inactive_since: "since_last_start",
			},
			"ignore_inactive_since set to since_first_start": {
				ignore_inactive_since: "since_first_start",
			},
			"ignore_inactive_since not set": {
				ignore_inactive_since: "",
			},
		}
		for name, test := range testCases {
			t.Run(name, func(t *testing.T) {
				c := defaultConfig()
				c.IgnoreInactive = ignoreInactiveSettings[test.ignore_inactive_since]
				p, err := newProspector(c, logptest.NewTestingLogger(t, ""), mustSourceIdentifier("foo-id"), nil, nil)
				require.NoError(t, err)
				fileProspector := p.(*fileProspector) //nolint:errcheck // we know the type
				assert.Equal(t, fileProspector.ignoreInactiveSince, ignoreInactiveSettings[test.ignore_inactive_since])
			})
		}
	})
	t.Run("accepts every scanner fingerprint and file identity combination", func(t *testing.T) {
		cases := []struct {
			name   string
			cfgStr string
		}{
			{
				name: "returns no error for a fully default config",
				cfgStr: `
paths: ['some']
`,
			},
			{
				name: "returns no error when fingerprint and identity is configured",
				cfgStr: `
paths: ['some']
file_identity.fingerprint: ~
prospector.scanner.fingerprint.enabled: true
`,
			},
			{
				name: "returns no error when deprecated scanner fingerprint contradicts the identity",
				cfgStr: `
paths: ['some']
file_identity.path: ~
prospector.scanner.fingerprint.enabled: true
`,
			},
			{
				name: "returns no error when the scanner fingerprint is disabled with fingerprint identity",
				cfgStr: `
paths: ['some']
file_identity.fingerprint: ~
prospector.scanner.fingerprint.enabled: false
`,
			},
			{
				name: "returns no error when the scanner fingerprint is disabled with omitted file identity",
				cfgStr: `
paths: ['some']
prospector.scanner.fingerprint.enabled: false
`,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				logger := logptest.NewTestingLogger(t, "")
				c, err := conf.NewConfigWithYAML([]byte(tc.cfgStr), tc.cfgStr)
				require.NoError(t, err)

				cfg := defaultConfig()
				err = c.Unpack(&cfg)
				require.NoError(t, err)
				require.NoError(t, normalizeConfig(c, &cfg, logger))

				_, err = newProspector(cfg, logger, mustSourceIdentifier("foo-id"), nil, nil)
				require.NoError(t, err)
			})
		}
	})

	t.Run("copytruncate rotation and fingerprint file identity", func(t *testing.T) {
		cases := []struct {
			name             string
			fileIdentity     string
			wantCopyTruncate bool
		}{
			{
				name:             "growing fingerprint ignores copytruncate and uses the standard prospector",
				fileIdentity:     "file_identity.fingerprint: ~",
				wantCopyTruncate: false,
			},
			{
				name:             "opting out of growing fingerprint keeps the copytruncate prospector",
				fileIdentity:     "file_identity.fingerprint.growing: false",
				wantCopyTruncate: true,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				logger := logptest.NewTestingLogger(t, "")
				cfgStr := fmt.Sprintf(`
paths: ['some']
%s
rotation.external.strategy.copytruncate:
  suffix_regex: '\.\d$'
`, tc.fileIdentity)

				c, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
				require.NoError(t, err, "test config must be valid YAML")

				cfg := defaultConfig()
				require.NoError(t, c.Unpack(&cfg), "test config must unpack into filestream config")
				require.NoError(t, normalizeConfig(c, &cfg, logger), "normalizeConfig must succeed")

				p, err := newProspector(cfg, logger, mustSourceIdentifier("foo-id"), nil, nil)
				require.NoError(t, err, "creating the prospector must succeed")

				if tc.wantCopyTruncate {
					assert.IsType(t, &copyTruncateFileProspector{}, p)
					return
				}

				fp, ok := p.(*fileProspector)
				require.True(t, ok, "expected the standard file prospector, got %T", p)
				assert.True(t, fp.growingFingerprint,
					"growing fingerprint must stay enabled when copytruncate is ignored")
			})
		}
	})

	t.Run("copytruncate rotation and preset", func(t *testing.T) {
		const copytruncate = `
rotation.external.strategy.copytruncate:
  suffix_regex: '\.\d$'
`
		const noGrowing = "file_identity.fingerprint.growing: false"
		const growing = "file_identity.fingerprint: ~"

		cases := []struct {
			name             string
			fileIdentity     string
			rotation         string
			preset           string
			wantCopyTruncate bool
			wantErr          []string
		}{
			{
				name:             "preset not set runs the copytruncate prospector forced to default",
				fileIdentity:     noGrowing,
				rotation:         copytruncate,
				wantCopyTruncate: true,
			},
			{
				name:             "preset.default runs the copytruncate prospector forced to default",
				fileIdentity:     noGrowing,
				rotation:         copytruncate,
				preset:           "preset.default: ~",
				wantCopyTruncate: true,
			},
			{
				name:         "preset.auto is rejected",
				fileIdentity: noGrowing,
				rotation:     copytruncate,
				preset:       "preset.auto: ~",
				wantErr:      []string{"rotation.external.strategy.copytruncate", "preset.auto", "preset.default"},
			},
			{
				name:         "preset.high_cardinality is rejected",
				fileIdentity: noGrowing,
				rotation:     copytruncate,
				preset:       "preset.high_cardinality: ~",
				wantErr:      []string{"rotation.external.strategy.copytruncate", "preset.high_cardinality", "preset.default"},
			},
			{
				name:         "growing fingerprint accepts preset.high_cardinality",
				fileIdentity: growing,
				rotation:     copytruncate,
				preset:       "preset.high_cardinality: ~",
			},
			{
				name:         "growing fingerprint accepts preset.auto",
				fileIdentity: growing,
				rotation:     copytruncate,
				preset:       "preset.auto: ~",
			},
			{
				name:         "no rotation accepts preset.high_cardinality",
				fileIdentity: growing,
				preset:       "preset.high_cardinality: ~",
			},
			{
				name:         "no rotation without preset",
				fileIdentity: growing,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				logger := logp.NewNopLogger()
				cfgStr := fmt.Sprintf("paths: ['some']\n%s\n%s\n%s\n", tc.fileIdentity, tc.rotation, tc.preset)

				c, err := conf.NewConfigWithYAML([]byte(cfgStr), cfgStr)
				require.NoError(t, err, "test config must be valid YAML")

				cfg := defaultConfig()
				require.NoError(t, c.Unpack(&cfg), "test config must unpack into filestream config")
				require.NoError(t, normalizeConfig(c, &cfg, logger), "normalizeConfig must succeed")

				p, err := newProspector(cfg, logger, mustSourceIdentifier("foo-id"), nil, nil)
				if len(tc.wantErr) > 0 {
					require.Error(t, err, "newProspector must reject the configuration")
					for _, want := range tc.wantErr {
						assert.ErrorContains(t, err, want, "the error must name the conflicting options")
					}
					return
				}
				require.NoError(t, err, "creating the prospector must succeed")

				if tc.wantCopyTruncate {
					ct, ok := p.(*copyTruncateFileProspector)
					require.True(t, ok, "expected the copytruncate prospector, got %T", p)
					assert.True(t, ct.forceDefaultPreset,
						"the copytruncate prospector must be forced to the default preset")
					return
				}

				fp, ok := p.(*fileProspector)
				require.True(t, ok, "expected the standard file prospector, got %T", p)
				assert.False(t, fp.forceDefaultPreset,
					"only the copytruncate prospector is forced to the default preset")
			})
		}
	})
}
