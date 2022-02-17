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

//go:build integration
// +build integration

package ilm_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/v7/libbeat/version"
)

const (
	// ElasticsearchDefaultHost is the default host for elasticsearch.
	ElasticsearchDefaultHost = "localhost"
	// ElasticsearchDefaultPort is the default port for elasticsearch.
	ElasticsearchDefaultPort = "9200"
)

func TestESClientHandler_CheckILMEnabled(t *testing.T) {
	t.Run("no ilm if disabled", func(t *testing.T) {
		h := newESClientHandler(t)
		b, err := h.CheckILMEnabled(false)
		assert.NoError(t, err)
		assert.False(t, b)
	})

	t.Run("with ilm if enabled", func(t *testing.T) {
		h := newESClientHandler(t)
		b, err := h.CheckILMEnabled(true)
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func TestESClientHandler_ILMPolicy(t *testing.T) {
	t.Run("does not exist", func(t *testing.T) {
		name := makeName("esch-policy-no")
		h := newESClientHandler(t)
		b, err := h.HasILMPolicy(name)
		assert.NoError(t, err)
		assert.False(t, b)
	})

	t.Run("create new", func(t *testing.T) {
		policy := ilm.Policy{
			Name: makeName("esch-policy-create"),
			Body: ilm.DefaultPolicy,
		}
		h := newESClientHandler(t)
		err := h.CreateILMPolicy(policy)
		require.NoError(t, err)

		b, err := h.HasILMPolicy(policy.Name)
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("overwrite", func(t *testing.T) {
		policy := ilm.Policy{
			Name: makeName("esch-policy-overwrite"),
			Body: ilm.DefaultPolicy,
		}
		h := newESClientHandler(t)

		err := h.CreateILMPolicy(policy)
		require.NoError(t, err)

		// check second 'create' does not throw (assuming race with other beat)
		err = h.CreateILMPolicy(policy)
		require.NoError(t, err)

		b, err := h.HasILMPolicy(policy.Name)
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func newESClientHandler(t *testing.T) ilm.ClientHandler {
	client := newRawESClient(t)
	return ilm.NewESClientHandler(client)
}

func newRawESClient(t *testing.T) ilm.ESClient {
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
	return fmt.Sprintf("http://%v:%v", getEsHost(), getEsPort())
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
	for name, test := range map[string]struct {
		enabled    bool
		version    string
		ilmEnabled bool
		err        bool
	}{
		"ilm enabled": {
			enabled:    true,
			ilmEnabled: true,
		},
		"ilm disabled": {
			enabled:    false,
			ilmEnabled: false,
		},
		"ilm enabled, version too old": {
			enabled: true,
			version: "5.0.0",
			err:     true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			h := ilm.NewFileClientHandler(newMockClient(test.version))
			b, err := h.CheckILMEnabled(test.enabled)
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
	c := newMockClient("")
	h := ilm.NewFileClientHandler(c)
	name := "test-policy"
	body := common.MapStr{"foo": "bar"}
	h.CreateILMPolicy(ilm.Policy{Name: name, Body: body})

	assert.Equal(t, name, c.name)
	assert.Equal(t, "policy", c.component)
	var out common.MapStr
	json.Unmarshal([]byte(c.body), &out)
	assert.Equal(t, body, out)
}

type mockClient struct {
	v                     common.Version
	component, name, body string
}

func newMockClient(v string) *mockClient {
	if v == "" {
		v = version.GetDefaultVersion()
	}
	return &mockClient{v: *common.MustNewVersion(v)}
}

func (c *mockClient) GetVersion() common.Version {
	return c.v
}

func (c *mockClient) Write(component string, name string, body string) error {
	c.component, c.name, c.body = component, name, body
	return nil
}
