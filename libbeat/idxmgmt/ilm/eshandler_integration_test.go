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

//+build integration

package ilm_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

const (
	// ElasticsearchDefaultHost is the default host for elasticsearch.
	ElasticsearchDefaultHost = "localhost"
	// ElasticsearchDefaultPort is the default port for elasticsearch.
	ElasticsearchDefaultPort = "9200"
)

func TestESClientHandler_ILMEnabled(t *testing.T) {
	t.Run("no ilm if disabled", func(t *testing.T) {
		h := newESClientHandler(t)
		b, err := h.ILMEnabled(ilm.ModeDisabled)
		assert.NoError(t, err)
		assert.False(t, b)
	})

	t.Run("with ilm if auto", func(t *testing.T) {
		h := newESClientHandler(t)
		b, err := h.ILMEnabled(ilm.ModeAuto)
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("with ilm if enabled", func(t *testing.T) {
		h := newESClientHandler(t)
		b, err := h.ILMEnabled(ilm.ModeEnabled)
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
		name := makeName("esch-policy-no")
		h := newESClientHandler(t)
		err := h.CreateILMPolicy(name, ilm.DefaultPolicy)
		require.NoError(t, err)

		b, err := h.HasILMPolicy(name)
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("overwrite", func(t *testing.T) {
		name := makeName("esch-policy-no")
		h := newESClientHandler(t)

		err := h.CreateILMPolicy(name, ilm.DefaultPolicy)
		require.NoError(t, err)

		// check second 'create' does not throw (assuming race with other beat)
		err = h.CreateILMPolicy(name, ilm.DefaultPolicy)
		require.NoError(t, err)

		b, err := h.HasILMPolicy(name)
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func TestESClientHandler_Alias(t *testing.T) {
	t.Run("does not exist", func(t *testing.T) {
		name := makeName("esch-alias-no")
		h := newESClientHandler(t)
		b, err := h.HasAlias(name)
		assert.NoError(t, err)
		assert.False(t, b)
	})

	t.Run("create new", func(t *testing.T) {
		name := makeName("esch-alias-create")
		first := fmt.Sprintf("%v-000001", name)
		h := newESClientHandler(t)
		err := h.CreateAlias(name, first)
		assert.NoError(t, err)

		b, err := h.HasAlias(name)
		assert.NoError(t, err)
		assert.True(t, b)
	})

	t.Run("second create", func(t *testing.T) {
		name := makeName("esch-alias-2create")
		first := fmt.Sprintf("%v-000001", name)
		h := newESClientHandler(t)

		err := h.CreateAlias(name, first)
		assert.NoError(t, err)

		err = h.CreateAlias(name, first)
		require.Error(t, err)
		assert.Equal(t, ilm.ErrAliasAlreadyExists, ilm.ErrReason(err))

		b, err := h.HasAlias(name)
		assert.NoError(t, err)
		assert.True(t, b)
	})
}

func newESClientHandler(t *testing.T) ilm.APIHandler {
	client, err := elasticsearch.NewClient(elasticsearch.ClientSettings{
		URL:              getURL(),
		Index:            outil.MakeSelector(),
		Username:         getUser(),
		Password:         getUser(),
		Timeout:          60 * time.Second,
		CompressionLevel: 3,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect to Test Elasticsearch instance: %v", err)
	}

	return ilm.ESClientHandler(client)
}

func makeName(base string) string {
	id := uuid.NewV4()
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
