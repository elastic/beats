// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

// captureCreateBody starts a fake ES that reports the given build flavor and
// records the body of the create-index request (PUT /<index>, not /_doc/),
// then drives a single Set to trigger lazy index creation.
func captureCreateBody(t *testing.T, buildFlavor string, idx IndexConfig) string {
	t.Helper()

	var (
		mu   sync.Mutex
		body string
		seen bool
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"`+buildFlavor+`"},"name":"fake"}`)
		case r.Method == http.MethodPut && r.URL.Path != "/" && !strings.Contains(r.URL.Path, "/_doc/"):
			b, _ := io.ReadAll(r.Body)
			mu.Lock()
			body = string(b)
			seen = true
			mu.Unlock()
			_, _ = io.WriteString(w, `{"acknowledged":true}`)
		default: // _doc write and anything else
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer srv.Close()

	cfg := &Config{
		ElasticsearchConfig: map[string]interface{}{
			"hosts":    []string{srv.URL},
			"username": "elastic",
			"password": "changeme",
		},
		Index: idx,
	}
	ext := &elasticStorage{cfg: cfg, logger: logptest.NewTestingLogger(t, t.Name())}
	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	c, err := ext.GetClient(context.Background(), component.KindReceiver, component.MustNewIDWithName("srv_test", "c"), "")
	require.NoError(t, err)
	require.NoError(t, c.Set(context.Background(), "k", []byte(`{"a":1}`)))

	mu.Lock()
	defer mu.Unlock()
	require.True(t, seen, "index-create request was never observed")
	return body
}

func TestEnsureIndex_Serverless_OmitsShardSettings(t *testing.T) {
	// Even with an explicit index config, serverless must not receive the
	// shard/replica settings.
	body := captureCreateBody(t, "serverless", IndexConfig{NumberOfShards: 3, NumberOfReplicas: 2})
	assert.NotContains(t, body, "number_of_shards", "serverless create must not send shard settings")
	assert.NotContains(t, body, "number_of_replicas")
	assert.NotContains(t, body, "settings")
	assert.Contains(t, body, "mappings", "mappings must still be sent on serverless")
	assert.Contains(t, body, "enabled", "the v enabled:false mapping must be present")
}

func TestEnsureIndex_Stateful_IncludesShardSettings(t *testing.T) {
	body := captureCreateBody(t, "default", IndexConfig{})
	assert.Contains(t, body, "number_of_shards")
	assert.Contains(t, body, "number_of_replicas")
	assert.Contains(t, body, "mappings")
}

func TestEnsureIndex_Stateful_UsesConfiguredShardSettings(t *testing.T) {
	// Configured shard/replica counts must reach the create request on a
	// stateful cluster.
	body := captureCreateBody(t, "default", IndexConfig{NumberOfShards: 3, NumberOfReplicas: 2})
	assert.Contains(t, body, `"number_of_shards":3`)
	assert.Contains(t, body, `"number_of_replicas":2`)
}
