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

package logv2

import (
	_ "embed"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/statestore"
	"github.com/elastic/beats/v7/libbeat/statestore/storetest"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

//go:embed testdata/log-input-all.yaml
var logInputAllYaml string

//go:embed testdata/filestream-all.json
var filestreamAllJson string

func TestTranslateCfgAllLogInputConfigs(t *testing.T) {
	cfg := config.MustNewConfigFrom(logInputAllYaml)
	newCfg, err := convertConfig(cfg)
	if err != nil {
		t.Fatalf("could not convert Log config into Filestream: %s", err)
	}

	validateConfig(t, newCfg, filestreamAllJson)

	store := openTestStatestore()
	p := PluginV2(logp.NewNopLogger(), store)
	m := p.Manager.(manager)
	if _, err := m.next.Create(newCfg); err != nil {
		t.Fatalf("Filestream input cannot be created from config: %s", err)
	}
}

func TestConvertHandlesFileIdentityCorrectly(t *testing.T) {
	testCases := map[string]struct {
		logYamlCfg      string
		expectedJsonCfg string
	}{
		"file_identity is not set": {
			logYamlCfg: `
id: foo
paths:
  - /tmp/foo
`,
			expectedJsonCfg: `
		{
		  "file_identity": {
		    "native": null
		  },
		  "id": "foo",
		  "paths": [
		    "/tmp/foo"
		  ],
		  "prospector": {
		    "scanner": {
		      "fingerprint": {
		        "enabled": false
		      }
		    }
		  },
		  "take_over": {
		    "enabled": true
		  },
		  "type": "filestream"
		}
		`,
		},
		"file_identiy is non default": {
			logYamlCfg: `
id: foo
paths:
   - /tmp/foo
file_identity.path: ~
`,
			expectedJsonCfg: `
		{
		  "file_identity": {
		    "path": null
		  },
		  "id": "foo",
		  "paths": [
		    "/tmp/foo"
		  ],
		  "prospector": {
		    "scanner": {
		      "fingerprint": {
		        "enabled": false
		      }
		    }
		  },
		  "take_over": {
		    "enabled": true
		  },
		  "type": "filestream"
		}
		`,
		},
		"file_identiy is fingerprint": {
			logYamlCfg: `
id: foo
paths:
 - /tmp/foo
file_identity.fingerprint: ~
`,
			expectedJsonCfg: `
{
  "file_identity": {
    "fingerprint": null
  },
  "id": "foo",
  "paths": [
    "/tmp/foo"
  ],
  "take_over": {
    "enabled": true
  },
  "type": "filestream"
}
`,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg, err := convertConfig(config.MustNewConfigFrom(tc.logYamlCfg))
			if err != nil {
				t.Fatalf("cannot convert Log input config to Filestream: %s", err)
			}

			validateConfig(t, cfg, tc.expectedJsonCfg)
		})
	}
}

func validateConfig(t *testing.T, cfg *config.C, expected string) {
	t.Helper()

	gotJson := config.DebugString(cfg, false)
	defer func() {
		if t.Failed() {
			t.Log("Final config as JSON:")
			t.Log(gotJson)
		}
	}()
	require.JSONEq(
		t,
		expected,
		gotJson,
		"configuration was not correctly converted from Log to Filestream",
	)
}

var _ statestore.States = (*testInputStore)(nil)

type testInputStore struct {
	registry *statestore.Registry
}

func openTestStatestore() *testInputStore {
	return &testInputStore{
		registry: statestore.NewRegistry(storetest.NewMemoryStoreBackend()),
	}
}

func (s *testInputStore) Close() {
	s.registry.Close()
}

func (s *testInputStore) StoreFor(string) (*statestore.Store, error) {
	return s.registry.Get("filebeat")
}

func (s *testInputStore) CleanupInterval() time.Duration {
	return 24 * time.Hour
}
