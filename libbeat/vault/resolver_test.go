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

package vault

import (
	"encoding/base64"
	"errors"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-ucfg"
)

// TestGetResolverCachesClient verifies the Vault client is built once per
// connection and shared across monitors (never rebuilt per monitor run).
func TestGetResolverCachesClient(t *testing.T) {
	c := Config{Enabled: true, Address: "http://127.0.0.1:8200", AuthMethod: "token", Token: "t", KVMount: "secret"}

	r1, err := GetResolver(c, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r2, err := GetResolver(c, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1 != r2 {
		t.Fatal("expected the same cached resolver for the same connection")
	}

	other := c
	other.Address = "http://127.0.0.1:8201"
	r3, err := GetResolver(other, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r3 == r1 {
		t.Fatal("expected a distinct resolver for a different connection")
	}
}

// TestResolveConfigStripsVault verifies the per-monitor `vault` field is
// consumed (removed) and non-vault fields are preserved. A config with no
// ${vault/..} refs never triggers a Vault read.
func TestResolveConfigStripsVault(t *testing.T) {
	blob := base64.StdEncoding.EncodeToString(
		[]byte(`{"address":"http://127.0.0.1:8200","auth_method":"token","token":"t","kv_mount":"secret"}`))
	raw, err := config.NewConfigFrom(map[string]interface{}{
		"type":  "http",
		"name":  "m1",
		"urls":  []string{"http://example.com"},
		"vault": blob,
	})
	if err != nil {
		t.Fatalf("build config: %v", err)
	}

	out, err := ResolveConfig(raw, nil)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if out.HasField("vault") {
		t.Fatal("expected the vault field to be stripped")
	}
	var got struct {
		Name string   `config:"name"`
		URLs []string `config:"urls"`
	}
	if err := out.Unpack(&got); err != nil {
		t.Fatalf("unpack resolved: %v", err)
	}
	if got.Name != "m1" || len(got.URLs) != 1 || got.URLs[0] != "http://example.com" {
		t.Fatalf("non-vault fields not preserved: %+v", got)
	}
}

// TestResolveConfigNoVault returns configs without a vault field unchanged.
func TestResolveConfigNoVault(t *testing.T) {
	raw, _ := config.NewConfigFrom(map[string]interface{}{"type": "http", "name": "m2"})
	out, err := ResolveConfig(raw, nil)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if out != raw {
		t.Fatal("expected the same config instance when no vault field is present")
	}
}

// TestDecodeBase64Config verifies the Fleet delivery form: the whole connection
// arrives as one base64-encoded JSON string (a single Fleet secret).
func TestDecodeBase64Config(t *testing.T) {
	jsonCfg := `{"address":"https://vault.internal:8200","auth_method":"approle","role_id":"r1","secret_id":"s1","kv_mount":"secret"}`
	b64 := base64.StdEncoding.EncodeToString([]byte(jsonCfg))

	c, err := decodeBase64Config(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Enabled {
		t.Fatal("expected Enabled=true for a delivered connection")
	}
	if c.Address != "https://vault.internal:8200" || c.AuthMethod != "approle" ||
		c.RoleID != "r1" || c.SecretID != "s1" || c.KVMount != "secret" {
		t.Fatalf("decoded config mismatch: %+v", c)
	}

	if _, err := decodeBase64Config("not!base64!!"); err == nil {
		t.Fatal("expected error for invalid base64")
	}
	if _, err := decodeBase64Config(base64.StdEncoding.EncodeToString([]byte("not json"))); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestResolveRouting verifies the parsing/routing done before any network call:
// non-vault keys fall through with ErrMissing, and malformed references error
// out. These paths never touch the Vault client, so a zero-value resolver is
// sufficient.
func TestResolveRouting(t *testing.T) {
	r := &resolver{cache: map[string]cacheEntry{}}

	tests := []struct {
		name      string
		key       string
		wantMiss  bool // expect ucfg.ErrMissing (fall through to next resolver)
		wantError bool // expect a (non-ErrMissing) error
	}{
		{name: "non-vault key falls through", key: "output.elasticsearch.password", wantMiss: true},
		{name: "empty key falls through", key: "", wantMiss: true},
		{name: "missing hash separator", key: "vault/myapp/creds", wantError: true},
		{name: "empty path", key: "vault/#password", wantError: true},
		{name: "empty field", key: "vault/myapp/creds#", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := r.resolve(tc.key)
			switch {
			case tc.wantMiss:
				if !errors.Is(err, ucfg.ErrMissing) {
					t.Fatalf("key %q: expected ucfg.ErrMissing, got %v", tc.key, err)
				}
			case tc.wantError:
				if err == nil || errors.Is(err, ucfg.ErrMissing) {
					t.Fatalf("key %q: expected a parse error, got %v", tc.key, err)
				}
			}
		})
	}
}

// TestResolveUsesCache verifies a non-expired cached value is returned without a
// client call.
func TestResolveUsesCache(t *testing.T) {
	r := &resolver{cache: map[string]cacheEntry{
		"myapp/creds#password": {value: "cached-secret", expiresAt: time.Now().Add(time.Hour)},
	}}

	v, _, err := r.resolve("vault/myapp/creds#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "cached-secret" {
		t.Fatalf("expected cached-secret, got %q", v)
	}
}

// TestRefreshInterval verifies parsing + defaulting of secret_refresh_interval.
func TestRefreshInterval(t *testing.T) {
	cases := map[string]time.Duration{
		"":        defaultRefreshInterval,
		"30s":     30 * time.Second,
		"10m":     10 * time.Minute,
		"garbage": defaultRefreshInterval,
		"0s":      defaultRefreshInterval,
		"-5m":     defaultRefreshInterval,
	}
	for in, want := range cases {
		if got := (Config{SecretRefreshInterval: in}).refreshInterval(); got != want {
			t.Fatalf("refreshInterval(%q) = %s, want %s", in, got, want)
		}
	}
}

// TestExpiredCacheIsRefetched verifies an expired cache entry is not served; the
// read path re-fetches (and here fails, since no client is configured), proving
// the TTL gate is applied rather than returning the stale value.
func TestExpiredCacheIsRefetched(t *testing.T) {
	apiCfg := vaultapi.DefaultConfig()
	apiCfg.Address = "http://127.0.0.1:1" // unreachable -> fetch errors instead of returning stale
	client, err := vaultapi.NewClient(apiCfg)
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	client.SetToken("t")

	r := &resolver{
		client:          client,
		cfg:             Config{Address: apiCfg.Address, AuthMethod: "token", Token: "t"},
		mount:           "secret",
		refreshInterval: time.Minute,
		log:             logp.NewLogger("test"),
		cache: map[string]cacheEntry{
			"myapp/creds#password": {value: "stale", expiresAt: time.Now().Add(-time.Minute)},
		},
	}
	// Expired entry -> read() must not return "stale"; it re-fetches and errors.
	if v, err := r.read("myapp/creds", "password"); err == nil {
		t.Fatalf("expected a re-fetch error for an expired entry, got value %q", v)
	}
}
