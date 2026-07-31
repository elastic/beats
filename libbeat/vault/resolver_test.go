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

	"github.com/elastic/go-ucfg"
)

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
	r := &resolver{cache: map[string]string{}}

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

// TestResolveUsesCache verifies a cached value is returned without a client call.
func TestResolveUsesCache(t *testing.T) {
	r := &resolver{cache: map[string]string{"myapp/creds#password": "cached-secret"}}

	v, _, err := r.resolve("vault/myapp/creds#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "cached-secret" {
		t.Fatalf("expected cached-secret, got %q", v)
	}
}
