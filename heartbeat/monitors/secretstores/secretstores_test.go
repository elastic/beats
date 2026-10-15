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
	"os"
	"strings"
	"sync"
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

// fakePrefix starts a fake reference: a top-level string "fake:<name>".
const fakePrefix = "fake:"

var errFakeUnavailable = errors.New("fake: store unavailable")

// fakeStore replaces top-level "fake:<name>" values by "<id>-<name>".
type fakeStore struct {
	id   string
	fail bool

	mu    sync.Mutex
	calls int
}

func (s *fakeStore) Validate(cfg *conf.C) error {
	var m map[string]any
	if err := (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep(""), ucfg.ResolveNOOP); err != nil {
		return err
	}
	_, err := s.replace(m, func(name string) (string, error) { return name, nil })
	return err
}

func (s *fakeStore) Replace(_ context.Context, cfg *conf.C) (*conf.C, error) {
	// Resolve keystore and environment references now: the result is built
	// without variable expansion, so secret values stay literal.
	var m map[string]any
	if err := cfg.Unpack(&m); err != nil {
		return nil, err
	}
	return s.replace(m, func(name string) (string, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.calls++
		if s.fail {
			return "", errFakeUnavailable
		}
		return s.id + "-" + name, nil
	})
}

func (s *fakeStore) replace(m map[string]any, resolve func(name string) (string, error)) (*conf.C, error) {
	for k, v := range m {
		str, ok := v.(string)
		name, found := strings.CutPrefix(str, fakePrefix)
		if !ok || !found {
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

func (s *fakeStore) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func TestMain(m *testing.M) {
	if err := registerFakeStore(); err != nil {
		fmt.Fprintf(os.Stderr, "could not register the fake secret store: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func registerFakeStore() error {
	return Register(fakeType, func(cfg *conf.C, _ *logp.Logger) (Store, error) {
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

func sharedFakeStores(t *testing.T) []*fakeStore {
	t.Helper()
	sharedStores.mu.Lock()
	defer sharedStores.mu.Unlock()
	var out []*fakeStore
	for _, s := range sharedStores.stores {
		fs, ok := s.(*fakeStore)
		require.True(t, ok, "only fake stores are expected")
		out = append(out, fs)
	}
	return out
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
	a, err := entry{settings: map[string]any{"type": fakeType, "a": "1", "b": "2"}}.identity()
	require.NoError(t, err, "identity must be computed")
	b, err := entry{settings: map[string]any{"b": "2", "type": fakeType, "a": "1"}}.identity()
	require.NoError(t, err, "identity must be computed")
	c, err := entry{settings: map[string]any{"type": fakeType, "a": "1", "b": "3"}}.identity()
	require.NoError(t, err, "identity must be computed")

	assert.Equal(t, a, b, "equal entries must have the same identity")
	assert.NotEqual(t, a, c, "different entries must have different identities")
}

func TestResolveWithoutSecretStores(t *testing.T) {
	resetSharedStores(t)
	r := NewResolver(logptest.NewTestingLogger(t, ""))
	cfg := newTestConfig(t, map[string]any{"type": "http", "target": "fake:token", "x": "${unknown.ref}"})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "a config without secret stores must not fail")
	assert.Same(t, cfg, got, "a config without secret stores must be returned as is")
	assert.NoError(t, r.Validate(cfg), "references are not checked without secret stores")
	assert.Empty(t, sharedFakeStores(t), "no store is expected without secret stores")
}

func TestResolveReplacesOnlyInCopy(t *testing.T) {
	resetSharedStores(t)
	t.Setenv("SECRETSTORES_TEST_ENV", "from-env")
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	cfg := newTestConfig(t, map[string]any{
		"type":     "http",
		"username": "fake:user",
		"env":      "${SECRETSTORES_TEST_ENV}",
		ConfigKey:  fakeStores(),
	})

	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")

	var resolved struct {
		Username string `config:"username"`
		Env      string `config:"env"`
		Stores   any    `config:"secret_stores"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "secret-user", resolved.Username, "the store must replace its references")
	assert.Equal(t, "from-env", resolved.Env, "env references must still resolve")
	assert.Nil(t, resolved.Stores, "secret_stores must not be given to the plugin")

	original := unpackAsText(t, cfg)
	assert.Equal(t, "fake:user", original["username"], "the original config must keep the reference")
	assert.NotNil(t, original[ConfigKey], "the original config must keep secret_stores")
}

func TestWithoutSecretStoresKeepsConfig(t *testing.T) {
	t.Setenv("SECRETSTORES_TEST_ENV", "from-env")
	cfg := newTestConfig(t, map[string]any{"type": "browser", "env": "${SECRETSTORES_TEST_ENV}", ConfigKey: fakeStores()})
	// Under Elastic Agent params have literal dotted keys.
	params, err := ucfg.NewFrom(map[string]any{"site.url": "x", "plain": "${not.a.secret}"}, ucfg.PathSep(""))
	require.NoError(t, err, "params must be valid")
	require.NoError(t, cfg.SetChild("params", -1, (*conf.C)(params)), "params must be set")

	got, err := withoutSecretStores(cfg)
	require.NoError(t, err, "the config must be copied")

	text := unpackAsText(t, got)
	assert.Equal(t, "${SECRETSTORES_TEST_ENV}", text["env"], "references must not be resolved by the copy")
	assert.NotContains(t, text, ConfigKey, "secret_stores must be removed")
	var p struct {
		Params map[string]any `config:"params"`
	}
	require.NoError(t, (*ucfg.Config)(got).Unpack(&p, ucfg.PathSep(""), ucfg.ResolveNOOP), "params must unpack")
	assert.Equal(t, map[string]any{"site.url": "x", "plain": "${not.a.secret}"}, p.Params, "dotted params keys must stay literal")
	assert.Contains(t, unpackAsText(t, cfg), ConfigKey, "the original config must keep secret_stores")
}

func TestResolveRunsStoresInOrder(t *testing.T) {
	resetSharedStores(t)
	const otherType = "fake_other"
	require.NoError(t, Register(otherType, func(*conf.C, *logp.Logger) (Store, error) {
		return &fakeStore{id: "other"}, nil
	}), "the second fake store must register")
	r := NewResolver(logptest.NewTestingLogger(t, ""))

	// The first store returns "fake:-fake:x", another reference, which the
	// second store replaces.
	cfg := newTestConfig(t, map[string]any{
		"type":    "http",
		"chained": "fake:fake:x",
		ConfigKey: []any{map[string]any{"type": fakeType, "id": "fake:"}, map[string]any{"type": otherType}},
	})
	got, err := r.Resolve(cfg)
	require.NoError(t, err, "references must resolve")
	var resolved struct {
		Chained string `config:"chained"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "other--fake:x", resolved.Chained, "stores must run in the secret_stores order")
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

	stores := sharedFakeStores(t)
	require.Len(t, stores, 1, "monitors with the same connection must share one store")
	assert.Equal(t, 2, stores[0].callCount(), "the shared store must serve both monitors")
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
		{
			name:    "unknown store type",
			cfg:     map[string]any{"type": "http", ConfigKey: []any{map[string]any{"type": "nope"}}},
			wantErr: `unknown secret store type "nope"`,
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

	cfg := newTestConfig(t, map[string]any{"type": "http", "target": "fake:token", ConfigKey: fakeStores()})
	require.NoError(t, r.Validate(cfg), "valid config must pass")
	assert.Empty(t, sharedFakeStores(t), "validation must not create shared stores")

	bad := newTestConfig(t, map[string]any{"type": "http", "target": "fake:", ConfigKey: fakeStores()})
	assert.ErrorContains(t, r.Validate(bad), "invalid fake secret references: fake: empty reference", "the store must reject its invalid references")

	invalidStore := newTestConfig(t, map[string]any{"type": "http", ConfigKey: fakeStores(map[string]any{"invalid": true})})
	assert.ErrorContains(t, r.Validate(invalidStore), "invalid fake config", "an invalid store config must be rejected")
}
