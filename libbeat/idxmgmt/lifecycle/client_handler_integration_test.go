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

// //go:build integration

package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
	libversion "github.com/elastic/elastic-agent-libs/version"
)

const (
	// ElasticsearchDefaultHost is the default host for elasticsearch.
	ElasticsearchDefaultHost = "http://localhost"
	// ElasticsearchDefaultPort is the default port for elasticsearch.
	ElasticsearchDefaultPort = "9200"
)

func TestESClientHandler_CheckILMEnabled(t *testing.T) {
	t.Run("no ilm if disabled", func(t *testing.T) {
		cfg := DefaultILMConfig(beat.Info{Name: "test"})
		cfg.ILM.Enabled = false
		h, err := newESClientHandler(t, cfg)
		require.NoError(t, err)
		b, err := h.CheckEnabled()
		assert.NoError(t, err)
		assert.False(t, b)
	})

	t.Run("with ilm if enabled", func(t *testing.T) {
		h, err := newESClientHandler(t, DefaultILMConfig(beat.Info{Name: "test"}))
		require.NoError(t, err)
		b, err := h.CheckEnabled()
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func TestESClientHandler_RecoverBadConfg(t *testing.T) {
	info := beat.Info{Name: "test"}
	client := newRawESClient(t)
	cfg := DefaultILMConfig(info)
	if client.IsServerless() {
		cfg.DSL.Enabled = false
		cfg.ILM.Enabled = true
	} else {
		cfg.DSL.Enabled = true
		cfg.ILM.Enabled = false
	}

	h, err := newESClientHandler(t, cfg)
	require.NoError(t, err)
	enabled, err := h.CheckEnabled()
	require.NoError(t, err)
	require.True(t, enabled)

}

func TestESClientHandler_ILMPolicy(t *testing.T) {

	t.Run("create new", func(t *testing.T) {
		policy := Policy{
			Name: makeName("esch-policy-create"),
			Body: DefaultILMPolicy,
		}
		cfg := DefaultILMConfig(beat.Info{Name: "test"})
		cfg.ILM.policyRaw = &policy
		h, err := newESClientHandler(t, cfg)
		require.NoError(t, err)
		err = h.CreatePolicyFromConfig()
		require.NoError(t, err)

		b, err := h.HasPolicy()
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("overwrite", func(t *testing.T) {
		policy := Policy{
			Name: makeName("esch-policy-overwrite"),
			Body: DefaultILMPolicy,
		}
		cfg := DefaultILMConfig(beat.Info{Name: "test"})
		cfg.ILM.policyRaw = &policy
		h, err := newESClientHandler(t, cfg)
		require.NoError(t, err)

		err = h.CreatePolicyFromConfig()
		require.NoError(t, err)

		// check second 'create' does not throw (assuming race with other beat)
		err = h.CreatePolicyFromConfig()
		require.NoError(t, err)

		b, err := h.HasPolicy()
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func newESClientHandler(t *testing.T, cfg LifecycleConfig) (ClientHandler, error) {
	client := newRawESClient(t)
	return NewESClientHandler(client, beat.Info{Name: "testbeat"}, cfg)
}

func newRawESClient(t *testing.T) ESClient {
	transport := httpcommon.DefaultHTTPTransportSettings()
	transport.Timeout = 60 * time.Second
	client, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              getURL(),
		Username:         getUser(),
		Password:         getPass(),
		CompressionLevel: 3,
		Transport:        transport,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to Test Elasticsearch instance: %v", err)
	}

	return client
}

func makeName(base string) string {
	id, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%v-%v", base, id.String())
}

func getURL() string {
	return fmt.Sprintf("%v:%v", getEsHost(), getEsPort())
}

// GetEsHost returns the Elasticsearch testing host.
func getEsHost() string {
	return getEnv("ES_HOST", ElasticsearchDefaultHost)
}

// GetEsPort returns the Elasticsearch testing port.
func getEsPort() string {
	return getEnv("ES_PORT", ElasticsearchDefaultPort)
}

// GetUser returns the Elasticsearch testing user.
func getUser() string { return getEnv("ES_USER", "") }

// GetPass returns the Elasticsearch testing user's password.
func getPass() string { return getEnv("ES_PASS", "") }

func getEnv(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func TestFileClientHandler_CheckILMEnabled(t *testing.T) {
	defaultCfg := DefaultILMConfig(beat.Info{Name: "test"})
	defaultCfgDisabled := defaultCfg
	defaultCfgDisabled.ILM.Enabled = false
	for name, test := range map[string]struct {
		version    string
		ilmEnabled bool
		err        bool
		cfg        LifecycleConfig
	}{
		"ilm enabled": {
			cfg: defaultCfg,

			ilmEnabled: true,
		},
		"ilm disabled": {
			ilmEnabled: false,
			cfg:        defaultCfgDisabled,
		},
		"ilm enabled, version too old": {
			version: "5.0.0",
			err:     true,
			cfg:     defaultCfg,
		},
	} {
		t.Run(name, func(t *testing.T) {
			h, err := NewFileClientHandler(newMockClient(test.version), beat.Info{Name: "test"}, test.cfg)
			require.NoError(t, err)
			b, err := h.CheckEnabled()
			assert.Equal(t, test.ilmEnabled, b)
			if test.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFileClientHandler_CreateILMPolicy(t *testing.T) {
	info := beat.Info{Name: "test"}
	cfg := DefaultILMConfig(info)
	testPolicy := Policy{
		Name: "test-policy",
		Body: mapstr.M{"foo": "bar"},
	}
	cfg.ILM.policyRaw = &testPolicy
	testClient := newMockClient("")
	h, err := NewFileClientHandler(testClient, info, cfg)
	require.NoError(t, err)
	err = h.CreatePolicyFromConfig()
	require.NoError(t, err)

	assert.Equal(t, testPolicy.Name, testClient.name)
	assert.Equal(t, "policy", testClient.component)
	var out mapstr.M
	json.Unmarshal([]byte(testClient.body), &out)
	assert.Equal(t, testPolicy.Body, out)
}

type mockClient struct {
	v                     libversion.V
	component, name, body string
}

func newMockClient(v string) *mockClient {
	if v == "" {
		v = version.GetDefaultVersion()
	}
	return &mockClient{v: *libversion.MustNew(v)}
}

func (c *mockClient) GetVersion() libversion.V {
	return c.v
}

func (c *mockClient) Write(component string, name string, body string) error {
	c.component, c.name, c.body = component, name, body
	return nil
}
