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
	"errors"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	ucfg "github.com/elastic/go-ucfg"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/libbeat/beat"
)

// swapType is a store that sets the "host" setting to its id, without
// having references. It stands for a connection whose resolved value changes
// a destination.
const swapType = "fake_swap"

type swapStore struct{ id string }

func (s *swapStore) Validate(*conf.C, Scope) error { return nil }

func (s *swapStore) Replace(_ context.Context, cfg *conf.C, _ Scope) (*conf.C, error) {
	c := conf.NewConfig()
	if err := c.Merge(cfg); err != nil {
		return nil, err
	}
	return c, c.SetString("host", -1, s.id)
}

func init() {
	mustRegister(swapType, func(cfg *conf.C, _ *logp.Logger) (Store, error) {
		var c struct {
			ID string `config:"id"`
		}
		return &swapStore{id: c.ID}, cfg.Unpack(&c)
	})
}

var errFakeDestinations = errors.New("redirects are not supported")

// fakeDestinations returns the "host" and "proxy" settings. A true "redirects"
// setting fails.
func fakeDestinations(cfg *conf.C) ([]string, error) {
	var c struct {
		Host      string `config:"host"`
		Proxy     string `config:"proxy"`
		Redirects bool   `config:"redirects"`
	}
	if err := cfg.Unpack(&c); err != nil {
		return nil, err
	}
	if c.Redirects {
		return nil, errFakeDestinations
	}
	hosts := []string{c.Host}
	if c.Proxy != "" {
		hosts = append(hosts, c.Proxy)
	}
	return hosts, nil
}

func TestParseAllowedHosts(t *testing.T) {
	tests := []struct {
		name    string
		raw     any
		want    map[string]bool
		wantErr string
	}{
		{name: "missing", raw: nil, want: nil},
		{name: "empty list", raw: []any{}, want: map[string]bool{}},
		{name: "host names", raw: []any{"API.Example.com.", "intranet", "bücher.example", "my_host"}, want: map[string]bool{"api.example.com": true, "intranet": true, "xn--bcher-kva.example": true, "my_host": true}},
		{name: "IPs", raw: []any{"10.0.0.1", "::FFFF:10.0.0.2", "2001:DB8::1"}, want: map[string]bool{"10.0.0.1": true, "10.0.0.2": true, "2001:db8::1": true}},
		{name: "not a list", raw: "api.example.com", wantErr: "allowed_hosts: expected a list of host names or IPs"},
		{name: "not a string", raw: []any{42}, wantErr: "allowed_hosts: expected a host name or IP, got int"},
		{name: "empty host", raw: []any{""}, wantErr: `"" must be a host name or IP`},
		{name: "spaces", raw: []any{" api.example.com"}, wantErr: "must be a host name or IP"},
		{name: "scheme", raw: []any{"https://api.example.com"}, wantErr: `"https://api.example.com" must be a host name or IP, without scheme, port or path`},
		{name: "port", raw: []any{"api.example.com:443"}, wantErr: "without scheme, port or path"},
		{name: "path", raw: []any{"api.example.com/v1"}, wantErr: "without scheme, port or path"},
		{name: "wildcard", raw: []any{"*.example.com"}, wantErr: "without scheme, port or path"},
		{name: "user info", raw: []any{"user@api.example.com"}, wantErr: "without scheme, port or path"},
		{name: "internationalized names in ASCII form", raw: []any{"BÜCHER.example", "ß.example"}, want: map[string]bool{"xn--bcher-kva.example": true, "xn--zca.example": true}},
		{name: "invalid internationalized name", raw: []any{"-ü.example"}, wantErr: `"-ü.example" is not a valid internationalized host name`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseAllowedHosts(tc.raw)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "allowed_hosts must be rejected")
				return
			}
			require.NoError(t, err, "allowed_hosts must be valid")
			assert.Equal(t, tc.want, got, "unexpected allowed hosts")
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := map[string]string{
		"API.Example.COM":  "api.example.com",
		"api.example.com.": "api.example.com",
		"::ffff:192.0.2.1": "192.0.2.1",
		"192.0.2.1.":       "192.0.2.1",
		// ASCII names are not converted with IDNA, as in net/http.
		"my_host": "my_host",
		// Non-ASCII names get the ASCII form that net/http dials.
		"bücher.example":        "xn--bcher-kva.example",
		"BÜCHER.example.":       "xn--bcher-kva.example",
		"xn--bcher-kva.example": "xn--bcher-kva.example",
		"ß.example":             "xn--zca.example",
		// Capital sharp s lowercases to "ß", but IDNA maps it to "ss".
		"ẞ.example": "ss.example",
		// Kelvin sign.
		"Kelvin.example": "kelvin.example",
	}
	for in, want := range tests {
		got, ok := normalizeHost(in)
		assert.True(t, ok, "%q must be normalized", in)
		assert.Equal(t, want, got, "unexpected normalized form of %q", in)
	}

	for _, in := range []string{"-ü.example", "\u00ad"} {
		_, ok := normalizeHost(in)
		assert.False(t, ok, "%q is not a valid internationalized host name and must not match", in)
	}
}

func TestParseEntriesAllowedHosts(t *testing.T) {
	entryWith := func(allowed any) map[string]any {
		m := map[string]any{"type": fakeType, "name": "prod", "id": "x"}
		if allowed != nil {
			m[allowedHostsKey] = allowed
		}
		return m
	}

	unrestricted, err := parseEntries([]any{entryWith(nil)})
	require.NoError(t, err, "entries must be valid")
	unrestrictedID, err := unrestricted[0].identity()
	require.NoError(t, err, "identity must be computed")

	tests := []struct {
		name           string
		allowed        any
		wantRestricted bool
		wantAllowed    map[string]bool
		wantErr        string
	}{
		{name: "restricted", allowed: []any{"API.example.com"}, wantRestricted: true, wantAllowed: map[string]bool{"api.example.com": true}},
		{name: "unrestricted", allowed: nil},
		{name: "empty allowed_hosts", allowed: []any{}, wantAllowed: map[string]bool{}},
		{name: "invalid allowed_hosts", allowed: []any{"*.example.com"}, wantErr: "secret_stores: entry 0: allowed_hosts:"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entries, err := parseEntries([]any{entryWith(tc.allowed)})
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr, "an invalid allowed_hosts must be rejected")
				return
			}
			require.NoError(t, err, "entries must be valid")
			require.Len(t, entries, 1, "one entry expected")
			e := entries[0]
			assert.Equal(t, tc.wantRestricted, e.isRestricted(), "unexpected restriction")
			assert.Equal(t, tc.wantAllowed, e.allowedHosts, "allowed hosts must be normalized")
			assert.False(t, e.config.HasField(allowedHostsKey), "the store must not receive allowed_hosts")
			id, err := e.identity()
			require.NoError(t, err, "identity must be computed")
			assert.Equal(t, unrestrictedID, id, "allowed_hosts must not change the identity, so the connections share a store")
		})
	}
}

func TestAllowedHosts(t *testing.T) {
	prod := func(allowed ...any) map[string]any {
		return map[string]any{"type": fakeType, "name": "prod", "id": "prod", allowedHostsKey: allowed}
	}
	other := map[string]any{"type": fakeType, "name": "other", "id": "other"}

	tests := []struct {
		name   string
		stores []any
		cfg    map[string]any
		// noDestinations stands for a monitor type without Destinations.
		noDestinations bool
		wantErr        string
	}{
		{
			name:   "allowed host",
			stores: []any{prod("api.example.com")},
			cfg:    map[string]any{"type": "dest", "host": "api.example.com", "header": "fake:prod@token"},
		},
		{
			name:   "allowed host and proxy, other case and trailing dot",
			stores: []any{prod("api.example.com", "proxy.example.com")},
			cfg:    map[string]any{"type": "dest", "host": "API.example.com.", "proxy": "proxy.example.com", "header": "fake:prod@token"},
		},
		{
			name:    "host not allowed",
			stores:  []any{prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "attacker.example", "header": "fake:prod@token"},
			wantErr: `secret_stores: connection "prod": not allowed by allowed_hosts: the monitor connects to a host that is not allowed`,
		},
		{
			name:   "internationalized name in another form",
			stores: []any{prod("bücher.example")},
			cfg:    map[string]any{"type": "dest", "host": "xn--bcher-kva.example", "header": "fake:prod@token"},
		},
		{
			// "ẞ" lowercases to "ß", but net/http dials "ss.example".
			name:    "name that lowercases to an allowed host but dials another",
			stores:  []any{prod("ß.example")},
			cfg:     map[string]any{"type": "dest", "host": "ẞ.example", "header": "fake:prod@token"},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "invalid internationalized name",
			stores:  []any{prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "-ü.example", "header": "fake:prod@token"},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "proxy not allowed",
			stores:  []any{prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "api.example.com", "proxy": "attacker.example", "header": "fake:prod@token"},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "reference in the host",
			stores:  []any{prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "fake:prod@token"},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "destinations error",
			stores:  []any{prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "api.example.com", "redirects": true, "header": "fake:prod@token"},
			wantErr: `connection "prod": not allowed by allowed_hosts: ` + errFakeDestinations.Error(),
		},
		{
			name:           "type without destinations",
			stores:         []any{prod("api.example.com")},
			cfg:            map[string]any{"type": "nodest", "header": "fake:prod@token"},
			noDestinations: true,
			wantErr:        `monitor type "nodest" is not supported`,
		},
		{
			name:           "unrestricted connection",
			stores:         []any{prod()},
			cfg:            map[string]any{"type": "nodest", "host": "attacker.example", "header": "fake:prod@token"},
			noDestinations: true,
		},
		{
			name:           "restricted connection without references",
			stores:         []any{prod("api.example.com"), maps.Clone(other)},
			cfg:            map[string]any{"type": "nodest", "host": "attacker.example", "header": "fake:other@token"},
			noDestinations: true,
		},
		{
			name: "reference added by an earlier connection",
			// other resolves "fake:other@x" to "fake:prod@-x", a reference to prod.
			stores:  []any{map[string]any{"type": fakeType, "name": "other", "id": "fake:prod@"}, prod("api.example.com")},
			cfg:     map[string]any{"type": "dest", "host": "attacker.example", "header": "fake:other@x"},
			wantErr: "a host that is not allowed",
		},
		{
			name:    "destination changed by a later connection",
			stores:  []any{prod("api.example.com"), map[string]any{"type": swapType, "name": "swap", "id": "attacker.example"}},
			cfg:     map[string]any{"type": "dest", "host": "api.example.com", "header": "fake:prod@token"},
			wantErr: "a host that is not allowed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resetSharedStores(t)
			r := NewResolver(logptest.NewTestingLogger(t, ""))
			m := map[string]any{ConfigKey: tc.stores}
			maps.Copy(m, tc.cfg)
			destinations := plugin.DestinationsFunc(fakeDestinations)
			if tc.noDestinations {
				destinations = nil
			}

			_, err := r.Resolve(newTestConfig(t, m), destinations)
			if tc.wantErr == "" {
				assert.NoError(t, err, "the monitor must be allowed")
				return
			}
			assert.ErrorContains(t, err, tc.wantErr, "the monitor must be rejected")
			assert.ErrorIs(t, err, ErrNotAllowed, "the error must be a policy error")
			assert.NotContains(t, err.Error(), "attacker.example", "a resolved host may be a secret and must not be named")
		})
	}
}

func TestWrapFactoryUsesPluginDestinations(t *testing.T) {
	resetSharedStores(t)
	logger := logptest.NewTestingLogger(t, "")
	r := NewResolver(logger)

	var made int
	pf := r.WrapFactory(plugin.PluginFactory{
		Name: "dest",
		Make: func(string, *conf.C, beat.Info) (plugin.Plugin, error) {
			made++
			return plugin.Plugin{}, nil
		},
		Destinations: fakeDestinations,
	})
	cfg, err := ucfg.NewFrom(map[string]any{
		"type":   "dest",
		"host":   "attacker.example",
		"header": "fake:prod@token",
		ConfigKey: []any{map[string]any{
			"type": fakeType, "name": "prod", "id": "prod", allowedHostsKey: []any{"api.example.com"},
		}},
	}, ucfg.PathSep("."))
	require.NoError(t, err, "test config must be valid")

	p, err := pf.Make("dest", (*conf.C)(cfg), beat.Info{Logger: logger})
	require.NoError(t, err, "making a secret plugin must not fail")
	t.Cleanup(func() { _ = p.Close() })
	require.Len(t, p.Jobs, 1, "without an inner plugin there must be one job")

	_, err = runJob(t, p.Jobs[0])
	assert.ErrorIs(t, err, ErrNotAllowed, "the job must report the policy error")
	assert.Zero(t, made, "the inner plugin must not be made with the secret")
}
