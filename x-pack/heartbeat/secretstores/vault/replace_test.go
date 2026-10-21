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

// newMonitorConfig builds a config with variable expansion, as the
// process-wide options do. PathSep("") keeps dotted keys, such as browser
// params, literal.
func newMonitorConfig(t *testing.T, m map[string]any) *conf.C {
	t.Helper()
	c, err := ucfg.NewFrom(m, ucfg.PathSep(""), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	return (*conf.C)(c)
}

func toBase64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

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

// textSettings unpacks cfg keeping ${...} expressions as text.
func textSettings(t *testing.T, cfg *conf.C) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP), "the config must unpack")
	return m
}

func TestReplace(t *testing.T) {
	t.Setenv("VAULT_TEST_ENV", "from-env")
	t.Setenv("VAULT_TEST_REF", "vault://app/creds#user")
	s, _ := newReplaceTestStore(t)
	const user = "vault://app/creds#user"
	script := "step('login', () => page.fill('#pw', 'vault://app/creds#token'))"
	plainKerberos := toBase64("auth_type: password\nusername: alice\n")

	tests := []struct {
		name string
		cfg  map[string]any
		// want is the resolved config, unpacked with the process-wide options.
		want map[string]any
		// unchanged means cfg itself must be returned.
		unchanged bool
		wantErr   string
	}{
		{
			name: "embedded reference",
			cfg:  map[string]any{"headers": map[string]any{"Authorization": "Bearer vault://app/creds#token"}},
			want: map[string]any{"headers": map[string]any{"Authorization": "Bearer s3cr3t"}},
		},
		{
			name: "secret value is not expanded",
			cfg:  map[string]any{"v": "vault://app/creds#tricky"},
			want: map[string]any{"v": "a${b}c"},
		},
		{
			name: "env and secret references",
			cfg:  map[string]any{"v": "${VAULT_TEST_ENV}:" + user, "escaped": "$${not.a.ref}"},
			want: map[string]any{"v": "from-env:bob", "escaped": "${not.a.ref}"},
		},
		{
			name: "dotted and nested params",
			cfg:  map[string]any{"params": map[string]any{"site.url": "vault://app/creds#token", "list": []any{"x=" + user}}},
			want: map[string]any{"params": map[string]any{"site.url": "s3cr3t", "list": []any{"x=bob"}}},
		},
		{
			name: "base64 setting with references is decoded",
			cfg:  map[string]any{"ntlm": toBase64(`{"domain": "${VAULT_TEST_ENV}", "password": "vault://app/creds#token"}`)},
			want: map[string]any{"ntlm": map[string]any{"domain": "from-env", "password": "s3cr3t"}},
		},
		{
			name: "other base64 settings are kept",
			cfg:  map[string]any{"kerberos": plainKerberos, "ntlm": "not base64!", "v": user},
			want: map[string]any{"kerberos": plainKerberos, "ntlm": "not base64!", "v": "bob"},
		},
		{
			name: "inline script is kept",
			cfg:  map[string]any{"source": map[string]any{"inline": map[string]any{"script": script}}, "params": map[string]any{"account": user}},
			want: map[string]any{"source": map[string]any{"inline": map[string]any{"script": script}}, "params": map[string]any{"account": "bob"}},
		},
		{
			name: "reference in an environment variable",
			cfg:  map[string]any{"v": "${VAULT_TEST_REF}"},
			want: map[string]any{"v": "bob"},
		},
		{
			name:      "unresolved environment variable without references",
			cfg:       map[string]any{"v": "${VAULT_TEST_MISSING}"},
			unchanged: true,
		},
		{
			name:    "unresolved environment variable with references",
			cfg:     map[string]any{"v": "${VAULT_TEST_MISSING}", "w": user},
			wantErr: "could not read monitor config: missing field accessing 'v'",
		},
		{
			name:      "no references",
			cfg:       map[string]any{"urls": []any{"https://example.com/#frag"}},
			unchanged: true,
		},
		{
			name:      "reference only in inline script",
			cfg:       map[string]any{"source": map[string]any{"inline": map[string]any{"script": script}}},
			unchanged: true,
		},
		{
			name:    "missing field",
			cfg:     map[string]any{"v": "vault://app/creds#nope"},
			wantErr: `vault: field "nope" not found at secret/data/app/creds`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newMonitorConfig(t, tc.cfg)
			before := textSettings(t, cfg)

			got, err := s.Replace(context.Background(), cfg)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "replacement must fail")
				return
			}
			require.NoError(t, err, "references must resolve")
			assert.Equal(t, before, textSettings(t, cfg), "the original config must not be modified")

			if tc.unchanged {
				assert.Same(t, cfg, got, "a config without references must be returned as is")
				return
			}
			var resolved map[string]any
			require.NoError(t, got.Unpack(&resolved), "resolved config must unpack with the process-wide options")
			assert.Equal(t, tc.want, resolved, "unexpected resolved config")
		})
	}
}

func TestValidate(t *testing.T) {
	t.Setenv("VAULT_TEST_BAD_REF", "vault://bad")
	s, fv := newReplaceTestStore(t)

	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{name: "valid", cfg: map[string]any{"target": "vault://app/creds#token"}},
		{name: "unresolved env", cfg: map[string]any{"target": "${missing.env}:vault://app/creds#token"}},
		{name: "inline script", cfg: map[string]any{"source": map[string]any{"inline": map[string]any{"script": "'vault://bad'"}}}},
		{name: "invalid", cfg: map[string]any{"target": "vault://app/creds"}, wantErr: "vault: invalid reference vault://app/creds: must have the form"},
		{name: "invalid in an environment variable", cfg: map[string]any{"target": "${VAULT_TEST_BAD_REF}"}, wantErr: "invalid reference vault://bad"},
		{name: "invalid in base64 ntlm", cfg: map[string]any{"ntlm": toBase64("password: vault://bad\n")}, wantErr: "invalid reference vault://bad"},
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
