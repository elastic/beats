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
	"maps"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	ucfg "github.com/elastic/go-ucfg"
)

// fakeType is a secret store registered for the tests of this package.
const fakeType = "fake"

// otherType is a second fake store type, to test stores run in order.
const otherType = "fake_other"

// fakePrefix starts a fake reference: a top-level string "fake:<name>".
const fakePrefix = "fake:"

var errFakeUnavailable = errors.New("fake: store unavailable")

// fakeStore replaces top-level "fake:<name>" values by "<id>-<name>".
type fakeStore struct {
	id   string
	fail bool
}

func (s *fakeStore) Validate(cfg *conf.C, scope Scope) error {
	var m map[string]any
	if err := (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep(""), ucfg.ResolveNOOP); err != nil {
		return err
	}
	_, err := s.replace(m, scope, func(name string) (string, error) { return name, nil })
	return err
}

func (s *fakeStore) Replace(_ context.Context, cfg *conf.C, scope Scope) (*conf.C, error) {
	// Resolve keystore and environment references now: the result is built
	// without variable expansion, so secret values stay literal.
	var m map[string]any
	if err := cfg.Unpack(&m); err != nil {
		return nil, err
	}
	return s.replace(m, scope, func(name string) (string, error) {
		if s.fail {
			return "", errFakeUnavailable
		}
		return s.id + "-" + name, nil
	})
}

func (s *fakeStore) replace(m map[string]any, scope Scope, resolve func(name string) (string, error)) (*conf.C, error) {
	for k, v := range m {
		str, ok := v.(string)
		content, found := strings.CutPrefix(str, fakePrefix)
		if !ok || !found {
			continue
		}
		name, mine, err := scope.Claim(content)
		if err != nil {
			return nil, err
		}
		if !mine {
			continue
		}
		if name == "" {
			return nil, errors.New("fake: empty reference")
		}
		value, err := resolve(name)
		if err != nil {
			return nil, err
		}
		m[k] = value
	}
	c, err := ucfg.NewFrom(m, ucfg.PathSep(""))
	return (*conf.C)(c), err
}

func init() {
	mustRegister(fakeType, func(cfg *conf.C, _ *logp.Logger) (Store, error) {
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
		return &fakeStore{id: c.ID, fail: c.Fail}, nil
	})
	mustRegister(otherType, func(*conf.C, *logp.Logger) (Store, error) {
		return &fakeStore{id: "other"}, nil
	})
}

func mustRegister(storeType string, factory Factory) {
	if err := Register(storeType, factory); err != nil {
		panic(err)
	}
}

// newTestConfig builds a config the way the process-wide options do.
func newTestConfig(t *testing.T, m map[string]any) *conf.C {
	t.Helper()
	c, err := ucfg.NewFrom(m, ucfg.PathSep("."), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	return (*conf.C)(c)
}

func fakeStores(extra ...map[string]any) []any {
	entry := map[string]any{"type": fakeType, "id": "secret"}
	for _, e := range extra {
		maps.Copy(entry, e)
	}
	return []any{entry}
}

func resetSharedStores(t *testing.T) {
	t.Helper()
	sharedStores.mu.Lock()
	sharedStores.stores = map[string]Store{}
	sharedStores.mu.Unlock()
}

func sharedStoreCount() int {
	sharedStores.mu.Lock()
	defer sharedStores.mu.Unlock()
	return len(sharedStores.stores)
}

func TestRegisterRejectsInvalidAndDuplicateTypes(t *testing.T) {
	factory := func(*conf.C, *logp.Logger) (Store, error) { return nil, nil }
	assert.ErrorContains(t, Register(fakeType, factory), "already registered", "registering a type twice must fail")
	assert.ErrorContains(t, Register("Bad-Type", factory), "invalid secret store type", "an invalid type must fail")
	assert.ErrorContains(t, Register("nofactory", nil), "needs a factory", "a missing factory must fail")
	for _, storeType := range []string{"Bad-Type", "nofactory"} {
		_, ok := lookupFactory(storeType)
		assert.False(t, ok, "a rejected type %q must not be registered", storeType)
	}
}

func TestParseEntries(t *testing.T) {
	toBase64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	tests := []struct {
		name    string
		raw     any
		want    int
		wantErr string
	}{
		{name: "nil", raw: nil, want: 0},
		{name: "list", raw: fakeStores(), want: 1},
		{name: "base64 json", raw: toBase64(`[{"type":"fake","id":"x"}]`), want: 1},
		{name: "base64 yaml", raw: toBase64("- type: fake\n  id: x\n"), want: 1},
		{name: "empty string", raw: "  ", want: 0},
		{name: "invalid base64", raw: "not base64!", wantErr: "invalid base64"},
		{name: "base64 not a list", raw: toBase64(`{"type":"fake"}`), wantErr: "not a YAML/JSON list"},
		{name: "wrong type", raw: 42, wantErr: "expected a list or a base64 string"},
		{name: "item not an object", raw: []any{"fake"}, wantErr: "expected an object"},
		{name: "missing type", raw: []any{map[string]any{"id": "x"}}, wantErr: "'type' is required"},
		{name: "unknown type", raw: []any{map[string]any{"type": "nope"}}, wantErr: `unknown secret store type "nope"`},
		{
			name:    "duplicate unnamed entries",
			raw:     []any{map[string]any{"type": fakeType}, map[string]any{"type": fakeType, "id": "2"}},
			wantErr: `more than one "fake" entry without a 'name'`,
		},
		{
			name:    "duplicate names",
			raw:     []any{map[string]any{"type": fakeType, "name": "a"}, map[string]any{"type": fakeType, "name": "a", "id": "2"}},
			wantErr: `duplicate "fake" entry name "a"`,
		},
		{
			name:    "name is not a string",
			raw:     []any{map[string]any{"type": fakeType, "name": 3}},
			wantErr: "'name' must be a string",
		},
		{
			name: "same name for different types",
			raw:  []any{map[string]any{"type": fakeType, "name": "a"}, map[string]any{"type": otherType, "name": "a"}},
			want: 2,
		},
		{
			name: "named and unnamed entries",
			raw:  []any{map[string]any{"type": fakeType}, map[string]any{"type": fakeType, "name": "a", "id": "2"}},
			want: 2,
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
	a1, err := entry{settings: map[string]any{"type": fakeType, "a": "1", "b": "2"}}.identity()
	require.NoError(t, err, "identity must be computed")
	a2, err := entry{settings: map[string]any{"b": "2", "type": fakeType, "a": "1"}}.identity()
	require.NoError(t, err, "identity must be computed")
	different, err := entry{settings: map[string]any{"type": fakeType, "a": "1", "b": "3"}}.identity()
	require.NoError(t, err, "identity must be computed")

	assert.Equal(t, a1, a2, "equal entries must have the same identity")
	assert.NotEqual(t, a1, different, "different entries must have different identities")

	named, err := entry{settings: map[string]any{"type": fakeType, "a": "1", "b": "2", "name": "prod"}}.identity()
	require.NoError(t, err, "identity must be computed")
	assert.Equal(t, a1, named, "the name must not change the identity, so equal connections share a store")
}

func TestResolveWithoutSecretStores(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	cfg := newTestConfig(t, map[string]any{"type": "http", "target": "fake:token", "x": "${unknown.ref}"})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "a config without secret stores must not fail")
	assert.Same(t, cfg, got, "a config without secret stores must be returned as is")
	assert.NoError(t, r.Validate(cfg), "references are not checked without secret stores")
}

func TestResolveReplacesOnlyInCopy(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{
		"type":     "http",
		"username": "fake:user",
		ConfigKey:  fakeStores(),
	})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")

	var resolved struct {
		Username string `config:"username"`
		Stores   any    `config:"secret_stores"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "secret-user", resolved.Username, "the store must replace its references")
	assert.Nil(t, resolved.Stores, "secret_stores must not be given to the plugin")

	var original struct {
		Username string `config:"username"`
	}
	require.NoError(t, cfg.Unpack(&original), "original config must unpack")
	assert.Equal(t, "fake:user", original.Username, "the original config must keep the reference")
	assert.True(t, cfg.HasField(ConfigKey), "the original config must keep secret_stores")
}

func TestResolveRunsStoresInOrder(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	// The fake store replaces "fake:x" by "<id>-x". With id "fake:", it
	// returns "fake:-x", a new reference only a later store can replace.
	cfg := newTestConfig(t, map[string]any{
		"type":    "http",
		"chained": "fake:x",
		ConfigKey: []any{map[string]any{"type": fakeType, "id": "fake:"}, map[string]any{"type": otherType}},
	})
	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")
	var resolved struct {
		Chained string `config:"chained"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "other--x", resolved.Chained, "stores must run in the secret_stores order")
}

func TestResolveSharesStoresAcrossMonitors(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	newCfg := func() *conf.C {
		return newTestConfig(t, map[string]any{"type": "http", "target": "fake:token", ConfigKey: fakeStores()})
	}

	_, err := r.Resolve(newCfg())
	require.NoError(t, err, "first monitor must resolve")
	_, err = r.Resolve(newCfg())
	require.NoError(t, err, "second monitor must resolve")

	assert.Equal(t, 1, sharedStoreCount(), "monitors with the same connection must share one store")
}

func TestResolveErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{
			name:    "store error",
			cfg:     map[string]any{"type": "http", "target": "fake:token", ConfigKey: fakeStores(map[string]any{"fail": true})},
			wantErr: "could not resolve fake secret references: " + errFakeUnavailable.Error(),
		},
		{
			name:    "invalid store config",
			cfg:     map[string]any{"type": "http", ConfigKey: fakeStores(map[string]any{"invalid": true})},
			wantErr: `secret_stores: invalid "fake" entry: invalid fake config`,
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

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{
			name: "valid reference",
			cfg:  map[string]any{"type": "http", "target": "fake:token", ConfigKey: fakeStores()},
		},
		{
			name:    "invalid reference",
			cfg:     map[string]any{"type": "http", "target": "fake:", ConfigKey: fakeStores()},
			wantErr: "invalid fake secret references: fake: empty reference",
		},
		{
			name:    "invalid store config",
			cfg:     map[string]any{"type": "http", ConfigKey: fakeStores(map[string]any{"invalid": true})},
			wantErr: `secret_stores: invalid "fake" entry: invalid fake config`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetSharedStores(t)
			r := NewResolver(logptest.NewTestingLogger(t, ""))
			err := r.Validate(newTestConfig(t, tc.cfg))
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "validation must fail")
			} else {
				assert.NoError(t, err, "validation must pass")
			}
			assert.Zero(t, sharedStoreCount(), "validation must not share stores with running monitors")
		})
	}
}

func TestResolveNamedEntries(t *testing.T) {
	stores := []any{
		map[string]any{"type": fakeType, "name": "prod", "id": "prod"},
		map[string]any{"type": fakeType, "name": "stg", "id": "stg"},
	}
	tests := []struct {
		name    string
		stores  []any
		cfg     map[string]any
		want    map[string]any
		wantErr string
	}{
		{
			name:   "each reference uses its own entry",
			stores: stores,
			cfg:    map[string]any{"a": "fake:prod@x", "b": "fake:stg@y"},
			want:   map[string]any{"a": "prod-x", "b": "stg-y"},
		},
		{
			name:    "unnamed reference is ambiguous",
			stores:  stores,
			cfg:     map[string]any{"a": "fake:x"},
			wantErr: `no connection name, but there are several entries: use <scheme>://<name>@... with one of "prod", "stg"`,
		},
		{
			name:    "unknown name",
			stores:  stores,
			cfg:     map[string]any{"a": "fake:dev@x"},
			wantErr: `no entry has the name "dev", available names: "prod", "stg"`,
		},
		{
			name:   "unnamed reference uses the unnamed entry",
			stores: append([]any{map[string]any{"type": fakeType, "id": "def"}}, stores...),
			cfg:    map[string]any{"a": "fake:x", "b": "fake:stg@y"},
			want:   map[string]any{"a": "def-x", "b": "stg-y"},
		},
		{
			name:   "unnamed reference uses the only entry, named or not",
			stores: stores[:1],
			cfg:    map[string]any{"a": "fake:x", "b": "fake:prod@y"},
			want:   map[string]any{"a": "prod-x", "b": "prod-y"},
		},
		{
			name:   "name is percent-encoded",
			stores: []any{map[string]any{"type": fakeType, "name": "my conn@1", "id": "c"}},
			cfg:    map[string]any{"a": "fake:my%20conn%401@x"},
			want:   map[string]any{"a": "c-x"},
		},
		{
			name:    "empty name",
			stores:  stores,
			cfg:     map[string]any{"a": "fake:@x"},
			wantErr: "empty connection name",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetSharedStores(t)
			r := NewResolver(logptest.NewTestingLogger(t, ""))
			cfgMap := map[string]any{"type": "http", ConfigKey: tc.stores}
			for k, v := range tc.cfg {
				cfgMap[k] = v
			}
			cfg := newTestConfig(t, cfgMap)

			// Validation must find the same problems without contacting the stores.
			verr := r.Validate(cfg)
			got, err := r.Resolve(cfg)
			if tc.wantErr != "" {
				assert.ErrorContains(t, verr, tc.wantErr, "validation must fail")
				assert.ErrorContains(t, err, tc.wantErr, "resolution must fail")
				return
			}
			require.NoError(t, verr, "validation must pass")
			require.NoError(t, err, "references must resolve")
			var resolved map[string]any
			require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
			delete(resolved, "type")
			assert.Equal(t, tc.want, resolved, "unexpected resolved config")
		})
	}
}
