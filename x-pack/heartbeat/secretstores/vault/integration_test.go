// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"
	"github.com/elastic/beats/v7/heartbeat/monitors/stdfields"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"

	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
)

const (
	itRootToken = "root" // VAULT_DEV_ROOT_TOKEN_ID in docker-compose.yml
	itPath      = "heartbeat-it/creds"
	itRole      = "heartbeat-it"
)

// itAddress returns the address of the Vault dev server from docker-compose.yml.
func itAddress() string {
	if addr := os.Getenv("VAULT_INTEGRATION_ADDR"); addr != "" {
		return addr
	}
	return "http://localhost:8200"
}

type itFixture struct {
	token    string
	roleID   string
	secretID string
}

// setupVault writes a secret and creates an AppRole allowed to read it.
func setupVault(t *testing.T) itFixture {
	t.Helper()
	ctx := context.Background()

	cfg := vaultapi.DefaultConfig()
	cfg.Address = itAddress()
	admin, err := vaultapi.NewClient(cfg)
	require.NoError(t, err, "admin client must be created")
	admin.SetToken(itRootToken)

	token := uuid.Must(uuid.NewV4()).String()
	_, err = admin.KVv2("secret").Put(ctx, itPath, map[string]any{"token": token})
	require.NoError(t, err, "secret must be written")

	err = admin.Sys().EnableAuthWithOptionsWithContext(ctx, "approle", &vaultapi.EnableAuthOptions{Type: "approle"})
	if err != nil && !strings.Contains(err.Error(), "path is already in use") {
		require.NoError(t, err, "approle auth must be enabled")
	}
	require.NoError(t, admin.Sys().PutPolicyWithContext(ctx, itRole, `path "secret/data/heartbeat-it/*" { capabilities = ["read"] }`), "policy must be written")
	_, err = admin.Logical().WriteWithContext(ctx, "auth/approle/role/"+itRole, map[string]any{"token_policies": itRole, "token_ttl": "1m"})
	require.NoError(t, err, "role must be written")

	roleID, err := admin.Logical().ReadWithContext(ctx, "auth/approle/role/"+itRole+"/role-id")
	require.NoError(t, err, "role id must be read")
	secretID, err := admin.Logical().WriteWithContext(ctx, "auth/approle/role/"+itRole+"/secret-id", nil)
	require.NoError(t, err, "secret id must be created")

	fx := itFixture{token: token}
	var ok bool
	fx.roleID, ok = roleID.Data["role_id"].(string)
	require.True(t, ok, "role id must be a string")
	fx.secretID, ok = secretID.Data["secret_id"].(string)
	require.True(t, ok, "secret id must be a string")
	return fx
}

func TestIntegrationResolve(t *testing.T) {
	fx := setupVault(t)

	entries := map[string]map[string]any{
		"token": {"address": itAddress(), "auth": map[string]any{"token": map[string]any{"value": itRootToken}}},
		"approle": {"address": itAddress(), "auth": map[string]any{"approle": map[string]any{
			"role_id": fx.roleID, "secret_id": fx.secretID,
		}}},
	}
	for name, entry := range entries {
		t.Run(name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(entry)
			require.NoError(t, err, "entry must be valid")
			s, err := newStore(cfg, logptest.NewTestingLogger(t, ""))
			require.NoError(t, err, "store must be created")

			v, err := s.Resolve(context.Background(), itPath, "token")
			require.NoError(t, err, "secret must be read from Vault")
			assert.Equal(t, fx.token, v, "unexpected secret value")

			_, err = s.Resolve(context.Background(), "heartbeat-it/missing", "token")
			assert.ErrorContains(t, err, "no secret at secret/data/heartbeat-it/missing", "a missing secret must be reported")
		})
	}
}

func TestIntegrationHTTPMonitor(t *testing.T) {
	fx := setupVault(t)

	// The endpoint only answers 200 when the secret from Vault is sent.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+fx.token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg, err := conf.NewConfigFrom(map[string]any{
		"type":                  "http",
		"id":                    "vault-it",
		"name":                  "vault-it",
		"schedule":              "@every 1m",
		"urls":                  []string{srv.URL},
		"check.request.headers": map[string]any{"Authorization": "Bearer $vault{" + itPath + "#token}"},
		"check.response.status": []int{200},
		secretstores.ConfigKey: []any{map[string]any{
			"type":    storeType,
			"address": itAddress(),
			"auth":    map[string]any{"approle": map[string]any{"role_id": fx.roleID, "secret_id": fx.secretID}},
		}},
	})
	require.NoError(t, err, "monitor config must be valid")

	logger := logptest.NewTestingLogger(t, "")
	resolved, err := secretstores.NewResolver(logger).Resolve(cfg)
	require.NoError(t, err, "references must resolve")

	factory, ok := plugin.GlobalPluginsReg.Get("http")
	require.True(t, ok, "the http plugin must be registered")
	p, err := factory.Make("http", resolved, beat.Info{Logger: logger})
	require.NoError(t, err, "the http plugin must be created")
	defer p.Close()

	sf, err := stdfields.ConfigToStdMonitorFields(cfg)
	require.NoError(t, err, "std fields must be read")

	var statuses []any
	for event := range p.RunWrapped(sf) {
		if s, err := event.Fields.GetValue("monitor.status"); err == nil {
			statuses = append(statuses, s)
		}
	}
	require.NotEmpty(t, statuses, "the monitor must publish events")
	for _, s := range statuses {
		assert.Equal(t, "up", s, "the monitor must authenticate with the secret from Vault")
	}
}
