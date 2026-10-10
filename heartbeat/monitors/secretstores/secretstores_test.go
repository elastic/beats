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

package secretstores

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	ucfg "github.com/elastic/go-ucfg"
)

// fakeType is a secret store registered for the tests of this package.
const fakeType = "fake"

// fakeSecrets holds the secrets served by every fake store, by path.
var fakeSecrets = map[string]map[string]string{
	"app/creds": {"token": "s3cr3t", "user": "bob", "tricky": "a${b}c"},
}

var errFakeUnavailable = errors.New("fake: store unavailable")

type fakeStore struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (s *fakeStore) Resolve(_ context.Context, path, field string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.fail {
		return "", errFakeUnavailable
	}
	v, ok := fakeSecrets[path][field]
	if !ok {
		return "", fmt.Errorf("fake: field %q not found at %s", field, path)
	}
	return v, nil
}

func (s *fakeStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func init() {
	Register(fakeType, func(cfg *conf.C, _ *logp.Logger) (Store, error) {
		var c struct {
			Fail    bool   `config:"fail"`
			Invalid bool   `config:"invalid"`
			ID      string `config:"id"`
		}
		if err := cfg.Unpack(&c); err != nil {
			return nil, err
		}
		if c.Invalid {
			return nil, errors.New("invalid fake config")
		}
		return &fakeStore{fail: c.Fail}, nil
	})
}

// newTestConfig builds a config the way the process-wide options do.
func newTestConfig(t *testing.T, m map[string]any) *conf.C {
	t.Helper()
	c, err := ucfg.NewFrom(m, ucfg.PathSep("."), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	return (*conf.C)(c)
}

// unpackAsText unpacks cfg keeping every reference as text.
func unpackAsText(t *testing.T, cfg *conf.C) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP), "config must unpack")
	return m
}

func fakeStores(extra ...map[string]any) []any {
	entry := map[string]any{"type": fakeType}
	for _, e := range extra {
		maps.Copy(entry, e)
	}
	return []any{entry}
}

func resetSharedStores(t *testing.T) {
	t.Helper()
	storesMu.Lock()
	stores = map[string]Store{}
	storesMu.Unlock()
}

func TestParseReference(t *testing.T) {
	tests := []struct {
		content string
		want    Reference
		wantErr string
	}{
		{content: "app/creds#token", want: Reference{Type: fakeType, Path: "app/creds", Field: "token"}},
		{content: "/app/creds/#token", want: Reference{Type: fakeType, Path: "app/creds", Field: "token"}},
		{content: "a#b#c", want: Reference{Type: fakeType, Path: "a#b", Field: "c"}},
		{content: "app:creds#token", want: Reference{Type: fakeType, Path: "app:creds", Field: "token"}},
		{content: "app/creds", wantErr: "must have the form $fake{<path>#<field>}"},
		{content: "#token", wantErr: "empty path or field"},
		{content: "app/creds#", wantErr: "empty path or field"},
		{content: "prod@app/creds#token", wantErr: "named connections are not supported yet"},
	}
	for _, tc := range tests {
		t.Run(tc.content, func(t *testing.T) {
			got, err := ParseReference(fakeType, tc.content)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "reference %q must be rejected", tc.content)
				return
			}
			require.NoError(t, err, "reference %q must be valid", tc.content)
			assert.Equal(t, tc.want, got, "unexpected parsed reference")
			assert.Equal(t, "$fake{"+got.Path+"#"+got.Field+"}", got.String(), "unexpected reference text")
		})
	}
}

func TestScanReferences(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{in: "$fake{app#token}", want: []string{"app#token"}},
		{in: "Bearer $fake{a#b} and $fake{c#d}", want: []string{"a#b", "c#d"}},
		{in: "${HOME} $fake{a#b}", want: []string{"a#b"}},
		{in: "$unknown{a#b}", want: nil},
		{in: "${fake/a#b}", want: nil},
		{in: "$co.elastic.secret{id}", want: nil},
		{in: "$fake{a{b}#c}", want: nil},
		{in: "price $5 {x}", want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			var got []string
			scanReferences(tc.in, func(storeType, content string) {
				assert.Equal(t, fakeType, storeType, "only registered types are references")
				got = append(got, content)
			})
			assert.Equal(t, tc.want, got, "unexpected references in %q", tc.in)
		})
	}
}

func TestRegisterRejectsInvalidAndDuplicateTypes(t *testing.T) {
	factory := func(*conf.C, *logp.Logger) (Store, error) { return nil, nil }
	assert.Panics(t, func() { Register(fakeType, factory) }, "registering a type twice must panic")
	assert.Panics(t, func() { Register("Bad-Type", factory) }, "an invalid type must panic")
}

func TestParseEntries(t *testing.T) {
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	tests := []struct {
		name    string
		raw     any
		want    int
		wantErr string
	}{
		{name: "nil", raw: nil, want: 0},
		{name: "list", raw: fakeStores(), want: 1},
		{name: "base64 json", raw: b64(`[{"type":"fake","id":"x"}]`), want: 1},
		{name: "base64 yaml", raw: b64("- type: fake\n  id: x\n"), want: 1},
		{name: "empty string", raw: "  ", want: 0},
		{name: "invalid base64", raw: "not base64!", wantErr: "invalid base64"},
		{name: "base64 not a list", raw: b64(`{"type":"fake"}`), wantErr: "not a YAML/JSON list"},
		{name: "wrong type", raw: 42, wantErr: "expected a list or a base64 string"},
		{name: "item not an object", raw: []any{"fake"}, wantErr: "expected an object"},
		{name: "missing type", raw: []any{map[string]any{"id": "x"}}, wantErr: "'type' is required"},
		{name: "unknown type", raw: []any{map[string]any{"type": "nope"}}, wantErr: `unknown secret store type "nope"`},
		{
			name:    "duplicate type",
			raw:     []any{map[string]any{"type": fakeType}, map[string]any{"type": fakeType, "id": "2"}},
			wantErr: `only one "fake" entry is supported`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseEntries(tc.raw)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "entries must be rejected")
				return
			}
			require.NoError(t, err, "entries must be valid")
			assert.Len(t, got, tc.want, "unexpected number of entries")
		})
	}
}

func TestParseEntriesKeepsBase64ContentLiteral(t *testing.T) {
	raw := base64.StdEncoding.EncodeToString([]byte(`[{"type":"fake","id":"pa${ss}"}]`))
	entries, err := parseEntries(raw)
	require.NoError(t, err, "entries must be valid")
	require.Len(t, entries, 1, "one entry expected")

	var c struct {
		ID string `config:"id"`
	}
	require.NoError(t, entries[0].config.Unpack(&c), "entry must unpack")
	assert.Equal(t, "pa${ss}", c.ID, "base64 content must not be variable-expanded")
}

func TestEntryIdentity(t *testing.T) {
	a, err := entryIdentity(map[string]any{"type": fakeType, "a": "1", "b": "2"})
	require.NoError(t, err, "identity must be computed")
	b, err := entryIdentity(map[string]any{"b": "2", "type": fakeType, "a": "1"})
	require.NoError(t, err, "identity must be computed")
	c, err := entryIdentity(map[string]any{"type": fakeType, "a": "1", "b": "3"})
	require.NoError(t, err, "identity must be computed")

	assert.Equal(t, a, b, "equal entries must have the same identity")
	assert.NotEqual(t, a, c, "different entries must have different identities")
}

func TestResolveWithoutSecretStores(t *testing.T) {
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	cfg := newTestConfig(t, map[string]any{"type": "http", "urls": []string{"http://localhost"}, "x": "${unknown.ref}"})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "a config without secret stores must not fail")
	assert.Same(t, cfg, got, "a config without secret stores must be returned as is")
}

func TestResolveReplacesReferencesOnlyInCopy(t *testing.T) {
	resetSharedStores(t)
	t.Setenv("SECRETSTORES_TEST_ENV", "from-env")
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{
		"type":          "http",
		"check.request": map[string]any{"headers": map[string]any{"Authorization": "Bearer $fake{app/creds#token}"}},
		"username":      "$fake{app/creds#user}",
		"tricky":        "$fake{app/creds#tricky}",
		"env":           "${SECRETSTORES_TEST_ENV}",
		"mixed":         "${SECRETSTORES_TEST_ENV}:$fake{app/creds#user}",
		"escaped":       "$${not.a.ref}",
		ConfigKey:       fakeStores(),
	})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")

	var resolved struct {
		Check struct {
			Request struct {
				Headers map[string]string `config:"headers"`
			} `config:"request"`
		} `config:"check"`
		Username string `config:"username"`
		Tricky   string `config:"tricky"`
		Env      string `config:"env"`
		Mixed    string `config:"mixed"`
		Escaped  string `config:"escaped"`
		Stores   any    `config:"secret_stores"`
	}
	// The process-wide options use VarExp: resolved values must stay literal.
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack with the process-wide options")
	assert.Equal(t, "Bearer s3cr3t", resolved.Check.Request.Headers["Authorization"], "header reference must be resolved")
	assert.Equal(t, "bob", resolved.Username, "field reference must be resolved")
	assert.Equal(t, "a${b}c", resolved.Tricky, "a resolved value must not be expanded again")
	assert.Equal(t, "from-env", resolved.Env, "env references must still resolve")
	assert.Equal(t, "from-env:bob", resolved.Mixed, "env and secret references must resolve in the same value")
	assert.Equal(t, "${not.a.ref}", resolved.Escaped, "escaped expressions must stay escaped")
	assert.Nil(t, resolved.Stores, "secret_stores must not be given to the plugin")

	original := unpackAsText(t, cfg)
	assert.Equal(t, "$fake{app/creds#user}", original["username"], "the original config must keep the reference")
	assert.NotNil(t, original[ConfigKey], "the original config must keep secret_stores")
}

func TestResolveUsesKeystore(t *testing.T) {
	resetSharedStores(t)
	ks := mapKeystore{"KS_USER": "from-keystore"}
	// Libbeat sets these process-wide options when a keystore exists.
	conf.OverwriteConfigOpts([]ucfg.Option{ucfg.PathSep("."), ucfg.Resolve(keystore.ResolverWrap(ks)), ucfg.ResolveEnv, ucfg.VarExp})
	t.Cleanup(func() {
		conf.OverwriteConfigOpts([]ucfg.Option{ucfg.PathSep("."), ucfg.ResolveEnv, ucfg.VarExp})
	})
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{
		"type":     "http",
		"username": "${KS_USER}",
		"target":   "$fake{app/creds#token}",
		ConfigKey:  fakeStores(map[string]any{"id": "${KS_USER}"}),
	})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")
	var resolved struct {
		Username string `config:"username"`
		Target   string `config:"target"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "from-keystore", resolved.Username, "keystore references must still resolve")
	assert.Equal(t, "s3cr3t", resolved.Target, "secret reference must be resolved")
}

func TestResolveParams(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{"type": "browser", ConfigKey: fakeStores()})
	// Under Elastic Agent params are parsed without variable expansion and
	// with literal dotted keys.
	params, err := ucfg.NewFrom(map[string]any{
		"site.url": "$fake{app/creds#token}",
		"nested":   map[string]any{"list": []any{"x-$fake{app/creds#user}"}},
		"plain":    "${not.a.secret}",
	}, ucfg.PathSep(""))
	require.NoError(t, err, "params must be valid")
	require.NoError(t, cfg.SetChild("params", -1, (*conf.C)(params)), "params must be set")

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "params references must resolve")

	var resolved struct {
		Params map[string]any `config:"params"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "s3cr3t", resolved.Params["site.url"], "dotted params keys must stay literal and be resolved")
	assert.Equal(t, map[string]any{"list": []any{"x-bob"}}, resolved.Params["nested"], "nested params must be resolved")
	assert.Equal(t, "${not.a.secret}", resolved.Params["plain"], "other expressions in params must stay untouched")
}

func TestResolveSharesStoresAcrossMonitors(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	newCfg := func() *conf.C {
		return newTestConfig(t, map[string]any{"type": "http", "target": "$fake{app/creds#token}", ConfigKey: fakeStores()})
	}

	_, err := r.Resolve(newCfg())
	require.NoError(t, err, "first monitor must resolve")
	_, err = r.Resolve(newCfg())
	require.NoError(t, err, "second monitor must resolve")

	storesMu.Lock()
	defer storesMu.Unlock()
	assert.Len(t, stores, 1, "monitors with the same connection must share one store")
}

func TestResolveErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{
			name:    "reference without secret_stores",
			cfg:     map[string]any{"type": "http", "target": "$fake{app/creds#token}"},
			wantErr: `requires a "fake" entry in secret_stores`,
		},
		{
			name:    "store error",
			cfg:     map[string]any{"type": "http", "target": "$fake{app/creds#token}", ConfigKey: fakeStores(map[string]any{"fail": true})},
			wantErr: errFakeUnavailable.Error(),
		},
		{
			name:    "missing field",
			cfg:     map[string]any{"type": "http", "target": "$fake{app/creds#nope}", ConfigKey: fakeStores()},
			wantErr: `field "nope" not found`,
		},
		{
			name:    "named connection",
			cfg:     map[string]any{"type": "http", "target": "$fake{prod@app/creds#token}", ConfigKey: fakeStores()},
			wantErr: "named connections are not supported yet",
		},
		{
			name:    "reference in secret_stores",
			cfg:     map[string]any{"type": "http", "target": "$fake{app/creds#token}", ConfigKey: fakeStores(map[string]any{"id": "$fake{app/creds#user}"})},
			wantErr: "secret references are not supported in secret_stores",
		},
		{
			name:    "invalid reference",
			cfg:     map[string]any{"type": "http", "target": "$fake{app/creds}", ConfigKey: fakeStores()},
			wantErr: "must have the form",
		},
		{
			name:    "invalid store config",
			cfg:     map[string]any{"type": "http", ConfigKey: fakeStores(map[string]any{"invalid": true})},
			wantErr: "invalid fake config",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetSharedStores(t)
			r := NewResolver(logptest.NewTestingLogger(t, ""))
			_, err := r.Resolve(newTestConfig(t, tc.cfg))
			assert.ErrorContains(t, err, tc.wantErr, "resolution must fail")
		})
	}
}

func TestValidateDoesNotContactStores(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{"type": "http", "target": "$fake{app/creds#token}", ConfigKey: fakeStores()})
	require.NoError(t, r.Validate(cfg), "valid config must pass")

	storesMu.Lock()
	assert.Empty(t, stores, "validation must not create shared stores")
	storesMu.Unlock()

	bad := newTestConfig(t, map[string]any{"type": "http", "target": "$fake{app/creds#token}"})
	assert.ErrorContains(t, r.Validate(bad), "requires a", "a reference without store must be rejected")
}

func TestResolveDoesNotCallStoreWithoutReferences(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	cfg := newTestConfig(t, map[string]any{"type": "http", ConfigKey: fakeStores()})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "a config without references must not fail")
	assert.Same(t, cfg, got, "a config without references must be returned as is")

	storesMu.Lock()
	defer storesMu.Unlock()
	for _, s := range stores {
		fs, ok := s.(*fakeStore)
		require.True(t, ok, "only fake stores are expected")
		assert.Zero(t, fs.callCount(), "no store request is expected without references")
	}
}

// mapKeystore is a read-only in-memory keystore.
type mapKeystore map[string]string

func (k mapKeystore) Retrieve(key string) (*keystore.SecureString, error) {
	v, ok := k[key]
	if !ok {
		return nil, keystore.ErrKeyDoesntExists
	}
	return keystore.NewSecureString([]byte(v)), nil
}

func (k mapKeystore) GetConfig() (*conf.C, error) { return conf.NewConfig(), nil }
func (k mapKeystore) IsPersisted() bool           { return true }
