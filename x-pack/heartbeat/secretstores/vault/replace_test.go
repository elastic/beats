// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package vault

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	ucfg "github.com/elastic/go-ucfg"
)

// newReplaceTestStore returns a store backed by a fake Vault serving
// app/creds.
func newReplaceTestStore(t *testing.T) (*store, *fakeVault) {
	t.Helper()
	fv := newFakeVault(t)
	fv.secrets["app/creds"] = map[string]any{"token": "s3cr3t", "user": "bob", "tricky": "a${b}c"}
	s, _, _ := newTestStore(t, tokenEntry(fv.address), nil)
	return s, fv
}

// newMonitorConfig builds a config the way the process-wide options do.
func newMonitorConfig(t *testing.T, m map[string]any) *conf.C {
	t.Helper()
	c, err := ucfg.NewFrom(m, ucfg.PathSep("."), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	return (*conf.C)(c)
}

func TestReplaceReferences(t *testing.T) {
	tests := []struct{ in, want string }{
		{in: "vault://app#token", want: "<app#token>"},
		{in: "Bearer vault://app/creds#token", want: "Bearer <app/creds#token>"},
		{in: "vault://a#b,vault://c#d", want: "<a#b>,<c#d>"},
		{in: "https://api/?key=vault://kv/app#key&debug=1", want: "https://api/?key=<kv/app#key>&debug=1"},
		{in: `{"token":"vault://kv/app#token"}`, want: `{"token":"<kv/app#token>"}`},
		{in: "${HOME}:vault://a#b", want: "${HOME}:<a#b>"},
		{in: "vault://my%20app#TLS%20Cert", want: "<my%20app#TLS%20Cert>"},
		{in: "myvault://a#b", want: "myvault://a#b"},
		{in: "x-vault://a#b", want: "x-vault://a#b"},
		{in: "other://a#b", want: "other://a#b"},
		{in: "https://example.com/path#frag", want: "https://example.com/path#frag"},
		{in: "${vault/a#b}", want: "${vault/a#b}"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := replaceReferences(tc.in, func(content string) (string, error) { return "<" + content + ">", nil })
			require.NoError(t, err, "replacement must not fail")
			assert.Equal(t, tc.want, got, "unexpected replacement of %q", tc.in)
		})
	}
}

func TestReplace(t *testing.T) {
	t.Setenv("VAULT_TEST_ENV", "from-env")
	s, _ := newReplaceTestStore(t)

	cfg := newMonitorConfig(t, map[string]any{
		"type":          "http",
		"check.request": map[string]any{"headers": map[string]any{"Authorization": "Bearer vault://app/creds#token"}},
		"username":      "vault://app/creds#user",
		"tricky":        "vault://app/creds#tricky",
		"env":           "${VAULT_TEST_ENV}",
		"mixed":         "${VAULT_TEST_ENV}:vault://app/creds#user",
		"escaped":       "$${not.a.ref}",
	})

	got, err := s.Replace(context.Background(), cfg)
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
	}
	// The process-wide options use VarExp: resolved values must stay literal.
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack with the process-wide options")
	assert.Equal(t, "Bearer s3cr3t", resolved.Check.Request.Headers["Authorization"], "header reference must be resolved")
	assert.Equal(t, "bob", resolved.Username, "field reference must be resolved")
	assert.Equal(t, "a${b}c", resolved.Tricky, "a resolved value must not be expanded again")
	assert.Equal(t, "from-env", resolved.Env, "env references must still resolve")
	assert.Equal(t, "from-env:bob", resolved.Mixed, "env and secret references must resolve in the same value")
	assert.Equal(t, "${not.a.ref}", resolved.Escaped, "escaped expressions must stay escaped")

	text, err := unpackAsText(cfg)
	require.NoError(t, err, "the original config must unpack")
	assert.Equal(t, "vault://app/creds#user", text["username"], "the original config must keep the reference")
}

func TestReplaceWithoutReferences(t *testing.T) {
	s, fv := newReplaceTestStore(t)
	cfg := newMonitorConfig(t, map[string]any{"type": "http", "urls": []string{"https://example.com/#frag"}})

	got, err := s.Replace(context.Background(), cfg)
	require.NoError(t, err, "a config without references must not fail")
	assert.Same(t, cfg, got, "a config without references must be returned as is")
	_, reads := fv.counts()
	assert.Zero(t, reads, "Vault must not be contacted without references")
}

func TestReplaceUsesKeystore(t *testing.T) {
	s, _ := newReplaceTestStore(t)
	ks := mapKeystore{"KS_USER": "from-keystore"}
	// Libbeat sets these process-wide options when a keystore exists.
	conf.OverwriteConfigOpts([]ucfg.Option{ucfg.PathSep("."), ucfg.Resolve(keystore.ResolverWrap(ks)), ucfg.ResolveEnv, ucfg.VarExp})
	t.Cleanup(func() {
		conf.OverwriteConfigOpts([]ucfg.Option{ucfg.PathSep("."), ucfg.ResolveEnv, ucfg.VarExp})
	})

	cfg := newMonitorConfig(t, map[string]any{"type": "http", "username": "${KS_USER}", "target": "vault://app/creds#token"})
	got, err := s.Replace(context.Background(), cfg)
	require.NoError(t, err, "references must resolve")
	var resolved struct {
		Username string `config:"username"`
		Target   string `config:"target"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "from-keystore", resolved.Username, "keystore references must still resolve")
	assert.Equal(t, "s3cr3t", resolved.Target, "secret reference must be resolved")
}

func TestReplaceParams(t *testing.T) {
	s, _ := newReplaceTestStore(t)
	cfg := newMonitorConfig(t, map[string]any{"type": "browser"})
	// Under Elastic Agent params are parsed without variable expansion and
	// with literal dotted keys.
	params, err := ucfg.NewFrom(map[string]any{
		"site.url": "vault://app/creds#token",
		"nested":   map[string]any{"list": []any{"x=vault://app/creds#user"}},
		"plain":    "${not.a.secret}",
	}, ucfg.PathSep(""))
	require.NoError(t, err, "params must be valid")
	require.NoError(t, cfg.SetChild("params", -1, (*conf.C)(params)), "params must be set")

	got, err := s.Replace(context.Background(), cfg)
	require.NoError(t, err, "params references must resolve")

	var resolved struct {
		Params map[string]any `config:"params"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, "s3cr3t", resolved.Params["site.url"], "dotted params keys must stay literal and be resolved")
	assert.Equal(t, map[string]any{"list": []any{"x=bob"}}, resolved.Params["nested"], "nested params must be resolved")
	assert.Equal(t, "${not.a.secret}", resolved.Params["plain"], "other expressions in params must stay untouched")
}

func TestReplaceEncodedSettings(t *testing.T) {
	t.Setenv("VAULT_TEST_DOMAIN", "CORP")
	s, _ := newReplaceTestStore(t)
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	plainKerberos := b64("auth_type: password\nusername: alice\n")
	cfg := newMonitorConfig(t, map[string]any{
		"type":     "http",
		"ntlm":     b64(`{"username": "bob", "domain": "${VAULT_TEST_DOMAIN}", "password": "vault://app/creds#token"}`),
		"kerberos": plainKerberos,
	})

	got, err := s.Replace(context.Background(), cfg)
	require.NoError(t, err, "references in base64 settings must resolve")

	var resolved struct {
		NTLM     map[string]any `config:"ntlm"`
		Kerberos string         `config:"kerberos"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, map[string]any{"username": "bob", "domain": "CORP", "password": "s3cr3t"}, resolved.NTLM,
		"a base64 setting with references must be given decoded and resolved")
	assert.Equal(t, plainKerberos, resolved.Kerberos, "a base64 setting without references must be kept as is")

	invalid := newMonitorConfig(t, map[string]any{"type": "http", "ntlm": "not base64!", "target": "vault://app/creds#user"})
	got, err = s.Replace(context.Background(), invalid)
	require.NoError(t, err, "an invalid base64 setting is left to the plugin")
	var kept struct {
		NTLM string `config:"ntlm"`
	}
	require.NoError(t, got.Unpack(&kept), "resolved config must unpack")
	assert.Equal(t, "not base64!", kept.NTLM, "an invalid base64 setting must be kept as is")
}

func TestReplaceSkipsInlineScripts(t *testing.T) {
	s, _ := newReplaceTestStore(t)
	script := "step('login', () => page.fill('#pw', 'vault://app/creds#token'))"

	cfg := newMonitorConfig(t, map[string]any{
		"type":          "browser",
		"source.inline": map[string]any{"script": script},
		"params":        map[string]any{"account": "vault://app/creds#user"},
	})

	got, err := s.Replace(context.Background(), cfg)
	require.NoError(t, err, "references must resolve")
	var resolved struct {
		Script string            `config:"source.inline.script"`
		Params map[string]string `config:"params"`
	}
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, script, resolved.Script, "inline scripts must not be modified")
	assert.Equal(t, "bob", resolved.Params["account"], "params must be resolved")

	scriptOnly := newMonitorConfig(t, map[string]any{"type": "browser", "source.inline": map[string]any{"script": script}})
	got, err = s.Replace(context.Background(), scriptOnly)
	require.NoError(t, err, "a config with only an inline script must not fail")
	assert.Same(t, scriptOnly, got, "a reference in an inline script is not a reference")
}

func TestReplaceError(t *testing.T) {
	s, _ := newReplaceTestStore(t)
	cfg := newMonitorConfig(t, map[string]any{"type": "http", "target": "vault://app/creds#nope"})
	_, err := s.Replace(context.Background(), cfg)
	assert.EqualError(t, err, `vault: field "nope" not found at secret/data/app/creds`, "a missing field must be reported")
}

func TestValidate(t *testing.T) {
	s, fv := newReplaceTestStore(t)
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{name: "valid", cfg: map[string]any{"type": "http", "target": "vault://app/creds#token"}},
		{name: "unresolved env", cfg: map[string]any{"type": "http", "target": "${missing.env}:vault://app/creds#token"}},
		{name: "inline script", cfg: map[string]any{"type": "browser", "source.inline.script": "'vault://bad'"}},
		{name: "invalid", cfg: map[string]any{"type": "http", "target": "vault://app/creds"}, wantErr: "vault: invalid reference vault://app/creds: must have the form"},
		{name: "invalid in base64 ntlm", cfg: map[string]any{"type": "http", "ntlm": b64("password: vault://bad\n")}, wantErr: "invalid reference vault://bad"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Validate(newMonitorConfig(t, tc.cfg))
			if tc.wantErr == "" {
				assert.NoError(t, err, "config must be valid")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "config must be rejected")
		})
	}
	logins, reads := fv.counts()
	assert.Zero(t, logins+reads, "validation must not contact Vault")
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
