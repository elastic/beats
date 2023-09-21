package lifecycle

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
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
			return 200, []byte{}, nil
		} else {
			return 404, []byte{}, nil
		}
	}

	return 201, []byte{}, nil
}

func TestESSetup(t *testing.T) {
	info := beat.Info{Beat: "test", Version: "9.9.9"}
	bothEnabledConfig := DefaultDSLConfig(info)
	bothEnabledConfig.ILM.Enabled = true
	cases := map[string]struct {
		serverless      bool
		serverHasPolicy bool
		cfg             LifecycleConfig
		err             bool
		expectedPUTPath string
		expectedName    string
		expectedPolicy  interface{}
		existingPolicy  interface{}
	}{
		"serverless-with-correct-defaults": {
			serverless:      true,
			cfg:             DefaultDSLConfig(info),
			err:             false,
			expectedPUTPath: "/_data_stream/test-9.9.9/_lifecycle",
			expectedName:    "test-9.9.9",
			expectedPolicy:  DefaultDSLPolicy,
		},
		"stateful-with-correct-default": {
			serverless:      false,
			cfg:             DefaultILMConfig(info),
			err:             false,
			expectedPUTPath: "/_ilm/policy/test",
			expectedName:    "test",
			expectedPolicy:  DefaultILMPolicy,
		},
		"serverless-with-wrong-defaults": {
			serverless:      true,
			cfg:             DefaultILMConfig(info),
			err:             false,
			expectedPUTPath: "/_data_stream/test-9.9.9/_lifecycle",
			expectedName:    "test-9.9.9",
			expectedPolicy:  DefaultDSLPolicy,
		},
		"stateful-with-wrong-defaults": {
			serverless:      false,
			cfg:             DefaultDSLConfig(info),
			err:             false,
			expectedPUTPath: "/_ilm/policy/test",
			expectedName:    "test",
		},
		"serverless-with-both-enabled": {
			serverless:      true,
			cfg:             bothEnabledConfig,
			err:             false,
			expectedPUTPath: "/_data_stream/test-9.9.9/_lifecycle",
			expectedName:    "test-9.9.9",
			expectedPolicy:  DefaultDSLPolicy,
		},
		"stateful-with-both-enabled": {
			serverless:      false,
			cfg:             bothEnabledConfig,
			err:             false,
			expectedPUTPath: "/_ilm/policy/test",
			expectedName:    "test",
			expectedPolicy:  DefaultILMPolicy,
		},
		"serverless-with-bare-PolicyName": {
			serverless: true,
			cfg:        LifecycleConfig{DSL: Config{Enabled: true, CheckExists: true, PolicyName: *fmtstr.MustCompileEvent("")}},
			err:        true,
		},
		"everything-disabled": {
			serverless: true,
			cfg:        LifecycleConfig{DSL: Config{Enabled: false}, ILM: Config{Enabled: false}},
			err:        true,
		},
		"custom-policy-name": {
			serverless: false,
			cfg: LifecycleConfig{ILM: Config{Enabled: true, CheckExists: true,
				PolicyName: *fmtstr.MustCompileEvent("test-%{[beat.version]}")}},
			err:          false,
			expectedName: "test-9.9.9",
		},
		"custom-policy-file": {
			serverless: false,
			cfg: LifecycleConfig{ILM: Config{Enabled: true, CheckExists: true,
				PolicyFile: "./testfiles/custom.json", PolicyName: *fmtstr.MustCompileEvent("test")}},
			expectedPolicy: mapstr.M{"hello": "world"},
			err:            false,
		},
		"do-not-overwrite": {
			serverless:     true,
			cfg:            DefaultDSLConfig(info),
			err:            false,
			existingPolicy: mapstr.M{"existing": "policy"},
			expectedPolicy: mapstr.M{"existing": "policy"},
		},
		"do-overwrite": {
			serverless: true,
			cfg: LifecycleConfig{DSL: Config{Enabled: true, Overwrite: true, CheckExists: true,
				PolicyName: *fmtstr.MustCompileEvent("test")}},
			err:            false,
			existingPolicy: mapstr.M{"existing": "policy"},
			expectedPolicy: DefaultDSLPolicy,
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
