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

package lifecycle

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

type mockESClient struct {
	serverless  bool
	hasPolicy   bool
	foundPolicy interface{}
}

func (client *mockESClient) GetVersion() version.V {
	return *version.MustNew("8.10.1")
}

func (client *mockESClient) IsServerless() bool {
	return client.serverless
}

func (client *mockESClient) Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error) {
	if method == "PUT" {
		client.foundPolicy = body
	}

	if method == "GET" {
		if client.hasPolicy || client.foundPolicy != nil {
			return http.StatusOK, []byte{}, nil
		} else {
			return http.StatusNotFound, []byte{}, nil
		}
	}

	return http.StatusCreated, []byte{}, nil
}

func TestESSetup(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}

	defaultILMCfg := RawConfig{
		ILM: config.MustNewConfigFrom(mapstr.M{"enabled": true, "policy_name": "test", "check_exists": true}),
		DSL: config.MustNewConfigFrom(mapstr.M{"enabled": false, "data_stream_pattern": "%{[beat.name]}-%{[beat.version]}", "check_exists": true}),
	}

	defaultDSLCfg := RawConfig{
		ILM: config.MustNewConfigFrom(mapstr.M{"enabled": false, "policy_name": "test", "check_exists": true}),
		DSL: config.MustNewConfigFrom(mapstr.M{"enabled": true, "data_stream_pattern": "%{[beat.name]}-%{[beat.version]}", "check_exists": true}),
	}

	bothDisabledConfig := RawConfig{
		ILM: config.MustNewConfigFrom(mapstr.M{"enabled": false, "policy_name": "test", "check_exists": true}),
		DSL: config.MustNewConfigFrom(mapstr.M{"enabled": false, "data_stream_pattern": "%{[beat.name]}-%{[beat.version]}", "check_exists": true}),
	}

	bothEnabledConfig := RawConfig{
		ILM: config.MustNewConfigFrom(mapstr.M{"enabled": true, "policy_name": "test", "check_exists": true}),
		DSL: config.MustNewConfigFrom(mapstr.M{"enabled": true, "data_stream_pattern": "%{[beat.name]}-%{[beat.version]}", "check_exists": true}),
	}
	withDSLBlank := RawConfig{
		ILM: config.MustNewConfigFrom(mapstr.M{"enabled": false, "policy_name": "test", "check_exists": true}),
		DSL: nil,
	}
	withILMBlank := RawConfig{
		ILM: nil,
		DSL: config.MustNewConfigFrom(mapstr.M{"enabled": false, "data_stream_pattern": "%{[beat.name]}-%{[beat.version]}", "check_exists": true}),
	}

	cases := map[string]struct {
		serverless      bool
		serverHasPolicy bool
		cfg             RawConfig
		err             bool
		expectedPUTPath string
		expectedName    string
		expectedPolicy  interface{}
		existingPolicy  interface{}
	}{
		"serverless-with-correct-defaults": {
			serverless:      true,
			cfg:             defaultDSLCfg,
			err:             false,
			expectedPUTPath: "/_data_stream/test-9.9.9/_lifecycle",
			expectedName:    "test-9.9.9",
			expectedPolicy:  DefaultDSLPolicy,
		},
		"stateful-with-correct-default": {
			serverless:      false,
			cfg:             defaultILMCfg,
			err:             false,
			expectedPUTPath: "/_ilm/policy/test",
			expectedName:    "test",
			expectedPolicy:  DefaultILMPolicy,
		},
		"serverless-with-wrong-defaults": {
			serverless: true,
			cfg:        defaultILMCfg,
			err:        true,
		},
		"stateful-with-wrong-defaults": {
			serverless: false,
			cfg:        defaultDSLCfg,
			err:        true,
		},
		"serverless-with-both-enabled": {
			serverless: true,
			cfg:        bothEnabledConfig,
			err:        true,
		},
		"stateful-with-both-enabled": {
			serverless: false,
			cfg:        bothEnabledConfig,
			err:        true,
		},
		"custom-policy-name": {
			serverless: false,
			cfg: RawConfig{
				ILM: config.MustNewConfigFrom(mapstr.M{"enabled": false, "policy_name": "test-%{[beat.version]}", "check_exists": true}),
			},
			err:          false,
			expectedName: "test-9.9.9",
		},
		"custom-policy-file": {
			serverless: false,
			cfg: RawConfig{
				ILM: config.MustNewConfigFrom(mapstr.M{"enabled": true,
					"policy_name":  "test",
					"policy_file":  "./testfiles/custom.json",
					"check_exists": true}),
			},
			expectedPolicy: mapstr.M{"hello": "world"},
			err:            false,
		},
		"do-not-overwrite": {
			serverless:     true,
			cfg:            defaultDSLCfg,
			err:            false,
			existingPolicy: mapstr.M{"existing": "policy"},
			expectedPolicy: mapstr.M{"existing": "policy"},
		},
		"do-overwrite": {
			serverless: true,
			cfg: RawConfig{
				DSL: config.MustNewConfigFrom(mapstr.M{"enabled": true, "overwrite": true,
					"check_exists": true, "data_stream_pattern": "test"}),
			},
			err:            false,
			existingPolicy: mapstr.M{"existing": "policy"},
			expectedPolicy: DefaultDSLPolicy,
		},
		"all-disabled-no-fail": {
			serverless: false,
			cfg:        bothDisabledConfig,
			err:        false,
		},
		"all-disabled-no-fail-serverless": {
			serverless: true,
			cfg:        bothDisabledConfig,
			err:        false,
		},
		"serverless-with-bare-config": {
			serverless: true,
			cfg:        withDSLBlank,
			err:        false,
		},
		"stateful-with-bare-config": {
			serverless: false,
			cfg:        withILMBlank,
			err:        false,
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			client := &mockESClient{serverless: test.serverless, foundPolicy: test.existingPolicy}
			gotClient, err := NewESClientHandler(client, info, test.cfg)
			if test.err {
				require.Error(t, err, "expected an error")
			} else {
				require.NoError(t, err, "no error expected")
			}
			if test.expectedPUTPath != "" {
				require.Equal(t, test.expectedPUTPath, gotClient.putPath, "URLs are not the same")
			}
			if test.expectedName != "" {
				require.Equal(t, test.expectedName, gotClient.name, "policy names are not equal")
			}
			if test.expectedPolicy != nil {
				err := gotClient.CreatePolicyFromConfig()
				require.NoError(t, err)
				require.Equal(t, test.expectedPolicy, client.foundPolicy, "found policies are not equal")
			}
		})
	}
}
