// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package elasticsearchstorage

import (
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

func TestConfigValidate(t *testing.T) {
	require.ErrorIs(t, (&Config{}).Validate(), errMissingAuthenticator, "storage configuration without an authenticator must fail")

	authenticatorID, _ := newTestElasticsearchAuthenticator(t, "http://localhost:9200", "", "")
	require.NoError(t, newTestStorageConfig(authenticatorID).Validate(), "storage configuration with an authenticator must validate")
}

func TestConfigDecode(t *testing.T) {
	config := createDefaultConfig().(*Config)
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
		"auth": map[string]any{
			"authenticator": "elasticsearchauth/state",
		},
	}).Unmarshal(config), "storage authenticator configuration must decode")
	require.NoError(t, config.Validate(), "decoded storage configuration must validate")
	require.True(t, config.Auth.HasValue(), "decoded configuration must contain an authenticator")
	require.Equal(t, component.MustNewIDWithName("elasticsearchauth", "state"), config.Auth.Get().AuthenticatorID, "decoded configuration must preserve the authenticator ID")
}

func TestElasticStorageStartUsesElasticsearchAuthenticator(t *testing.T) {
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"version":{"number":"8.10.0","build_flavor":"default"},"name":"fake"}`)
	}))
	t.Cleanup(server.Close)

	authenticatorID, authenticator := newTestElasticsearchAuthenticator(t, server.URL, "elastic", "password")
	storage := &elasticStorage{
		cfg:    newTestStorageConfig(authenticatorID),
		logger: logptest.NewTestingLogger(t, ""),
	}

	require.Equal(t, []component.ID{authenticatorID}, storage.Dependencies(), "storage must depend on its configured authenticator")
	require.NoError(t, storage.Start(t.Context(), storageTestHost{authenticatorID: authenticator}), "storage must connect through the configured authenticator")
	t.Cleanup(func() {
		require.NoError(t, storage.Shutdown(t.Context()), "storage must shut down")
	})

	require.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("elastic:password")), authorization, "storage connection ping must carry authenticator credentials")
}
