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

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
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
				p, err := newProspector(c, logptest.NewTestingLogger(t, ""), mustSourceIdentifier("foo-id"))
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

				_, err = newProspector(cfg, logger, mustSourceIdentifier("foo-id"))
				require.NoError(t, err)
			})
		}
	})

	t.Run("derives scanner fingerprinting from the identity when normalization is bypassed", func(t *testing.T) {
		// The fingerprint identity without scanner fingerprints would
		// silently collapse all files into one empty-fingerprint registry
		// entry, so newProspector cannot trust the unpacked value.
		cases := []struct {
			cfgStr      string
			wantEnabled bool
		}{
			{cfgStr: `
paths: ['some']
prospector.scanner.fingerprint.enabled: false
`, wantEnabled: true},
			{cfgStr: `
paths: ['some']
file_identity.fingerprint: ~
prospector.scanner.fingerprint.enabled: false
`, wantEnabled: true},
			{cfgStr: `
paths: ['some']
file_identity.native: ~
prospector.scanner.fingerprint.enabled: true
`, wantEnabled: false},
		}
		for _, tc := range cases {
			c, err := conf.NewConfigWithYAML([]byte(tc.cfgStr), tc.cfgStr)
			require.NoError(t, err)

			cfg := defaultConfig()
			require.NoError(t, c.Unpack(&cfg))

			p, err := newProspector(cfg, logptest.NewTestingLogger(t, ""), mustSourceIdentifier("foo-id"))
			require.NoError(t, err)
			fp, ok := p.(*fileProspector)
			require.True(t, ok, "expected the standard file prospector, got %T", p)
			watcher, ok := fp.filewatcher.(*fileWatcher)
			require.True(t, ok, "expected the filestream file watcher, got %T", fp.filewatcher)
			assert.Equal(t, tc.wantEnabled, watcher.cfg.Scanner.Fingerprint.Enabled, tc.cfgStr)
		}
	})

	t.Run("copytruncate rotation and fingerprint file identity", func(t *testing.T) {
		cases := []struct {
			name             string
			fileIdentity     string
			wantCopyTruncate bool
		}{
			{
				name:             "Enhanced Fingerprint ignores copytruncate and uses the standard prospector",
				fileIdentity:     "file_identity.fingerprint: ~",
				wantCopyTruncate: false,
			},
			{
				name:             "opting out of Enhanced Fingerprint keeps the copytruncate prospector",
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

				p, err := newProspector(cfg, logger, mustSourceIdentifier("foo-id"))
				require.NoError(t, err, "creating the prospector must succeed")

				if tc.wantCopyTruncate {
					assert.IsType(t, &copyTruncateFileProspector{}, p)
					return
				}

				fp, ok := p.(*fileProspector)
				require.True(t, ok, "expected the standard file prospector, got %T", p)
				assert.True(t, fp.growingFingerprint,
					"Enhanced Fingerprint must stay enabled when copytruncate is ignored")
			})
		}
	})
}
