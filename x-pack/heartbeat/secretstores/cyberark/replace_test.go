// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cyberark

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

func newReplaceTestStore(t *testing.T) (*store, *fakeCCP) {
	t.Helper()
	fc := newFakeCCP(t)
	fc.accounts["AppSafe/Creds"] = map[string]any{
		"Content":  "s3cr3t",
		"UserName": "bob",
		"tricky":   "a${b}c",
	}
	s, _, _ := newTestStore(t, ccpEntry(fc.address), nil)
	return s, fc
}

func newMonitorConfig(t *testing.T, m map[string]any) *conf.C {
	t.Helper()
	c, err := ucfg.NewFrom(m, ucfg.PathSep(""), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	return (*conf.C)(c)
}

func toBase64(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

func textSettings(t *testing.T, cfg *conf.C) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, (*ucfg.Config)(cfg).Unpack(&m, ucfg.PathSep("."), ucfg.ResolveNOOP), "the config must unpack")
	return m
}

func TestReplaceReferences(t *testing.T) {
	tests := []struct{ in, want string }{
		{in: "cyberark://AppSafe/Creds#Content", want: "<AppSafe/Creds#Content>"},
		{in: "Bearer cyberark://AppSafe/Creds#password", want: "Bearer <AppSafe/Creds#password>"},
		{in: "cyberark://a/b#c,cyberark://d/e#f", want: "<a/b#c>,<d/e#f>"},
		{in: "${HOME}:cyberark://a/b#c", want: "${HOME}:<a/b#c>"},
		{in: "mycyberark://a/b#c", want: "mycyberark://a/b#c"},
		{in: "x-cyberark://a/b#c", want: "x-cyberark://a/b#c"},
		{in: "vault://a/b#c", want: "vault://a/b#c"},
		{in: "https://example.com/path#frag", want: "https://example.com/path#frag"},
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
	t.Setenv("CYBERARK_TEST_ENV", "from-env")
	t.Setenv("CYBERARK_TEST_REF", "cyberark://AppSafe/Creds#username")
	s, _ := newReplaceTestStore(t)
	const user = "cyberark://AppSafe/Creds#username"
	script := "step('login', () => page.fill('#pw', 'cyberark://AppSafe/Creds#password'))"

	tests := []struct {
		name      string
		cfg       map[string]any
		want      map[string]any
		unchanged bool
		wantErr   string
	}{
		{
			name: "embedded reference with alias",
			cfg:  map[string]any{"headers": map[string]any{"Authorization": "Bearer cyberark://AppSafe/Creds#password"}},
			want: map[string]any{"headers": map[string]any{"Authorization": "Bearer s3cr3t"}},
		},
		{
			name: "secret value is not expanded",
			cfg:  map[string]any{"v": "cyberark://AppSafe/Creds#tricky"},
			want: map[string]any{"v": "a${b}c"},
		},
		{
			name: "env and secret references",
			cfg:  map[string]any{"v": "${CYBERARK_TEST_ENV}:" + user},
			want: map[string]any{"v": "from-env:bob"},
		},
		{
			name: "reference in an environment variable",
			cfg:  map[string]any{"v": "${CYBERARK_TEST_REF}"},
			want: map[string]any{"v": "bob"},
		},
		{
			name: "base64 setting with references is decoded",
			cfg:  map[string]any{"ntlm": toBase64(`{"domain": "${CYBERARK_TEST_ENV}", "password": "cyberark://AppSafe/Creds#password"}`)},
			want: map[string]any{"ntlm": map[string]any{"domain": "from-env", "password": "s3cr3t"}},
		},
		{
			name: "inline script is kept",
			cfg:  map[string]any{"source": map[string]any{"inline": map[string]any{"script": script}}, "params": map[string]any{"account": user}},
			want: map[string]any{"source": map[string]any{"inline": map[string]any{"script": script}}, "params": map[string]any{"account": "bob"}},
		},
		{
			name:      "no references",
			cfg:       map[string]any{"urls": []any{"https://example.com/#frag"}},
			unchanged: true,
		},
		{
			name:    "missing field",
			cfg:     map[string]any{"v": "cyberark://AppSafe/Creds#nope"},
			wantErr: `cyberark: field "nope" not found in account AppSafe/Creds`,
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
			require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
			assert.Equal(t, tc.want, resolved, "unexpected resolved config")
		})
	}
}

func TestValidate(t *testing.T) {
	t.Setenv("CYBERARK_TEST_BAD_REF", "cyberark://bad")
	s, fc := newReplaceTestStore(t)

	tests := []struct {
		name    string
		cfg     map[string]any
		wantErr string
	}{
		{name: "valid", cfg: map[string]any{"target": "cyberark://AppSafe/Creds#password"}},
		{name: "unresolved env", cfg: map[string]any{"target": "${missing.env}:cyberark://AppSafe/Creds#password"}},
		{name: "inline script", cfg: map[string]any{"source": map[string]any{"inline": map[string]any{"script": "'cyberark://bad'"}}}},
		{name: "invalid", cfg: map[string]any{"target": "cyberark://AppSafe/Creds"}, wantErr: "cyberark: invalid reference cyberark://AppSafe/Creds"},
		{name: "invalid in an environment variable", cfg: map[string]any{"target": "${CYBERARK_TEST_BAD_REF}"}, wantErr: "invalid reference cyberark://bad"},
		{name: "invalid in base64 ntlm", cfg: map[string]any{"ntlm": toBase64("password: cyberark://bad\n")}, wantErr: "invalid reference cyberark://bad"},
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
	assert.Zero(t, fc.readCount(), "validation must not contact CyberArk")
}
