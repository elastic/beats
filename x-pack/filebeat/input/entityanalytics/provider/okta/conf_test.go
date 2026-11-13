// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/transport/httpcommon"
	"github.com/elastic/lumberjack"
)

var validateTests = []struct {
	name    string
	cfg     conf
	wantErr error
}{
	{
		name: "default",
		cfg: func() conf {
			cfg := defaultConfig()
			cfg.OktaDomain = "test.okta.com"
			cfg.OktaToken = "test-token"
			return cfg
		}(),
		wantErr: nil,
	},
	{
		name: "invalid_sync_interval",
		cfg: conf{
			OktaDomain:     "test.okta.com",
			OktaToken:      "test-token",
			SyncInterval:   0,
			UpdateInterval: time.Second * 2,
		},
		wantErr: errInvalidSyncInterval,
	},
	{
		name: "invalid_update_interval",
		cfg: conf{
			OktaDomain:     "test.okta.com",
			OktaToken:      "test-token",
			SyncInterval:   time.Second,
			UpdateInterval: 0,
		},
		wantErr: errInvalidUpdateInterval,
	},
	{
		name: "invalid_relative_intervals",
		cfg: conf{
			OktaDomain:     "test.okta.com",
			OktaToken:      "test-token",
			SyncInterval:   time.Second,
			UpdateInterval: time.Second * 2,
		},
		wantErr: errSyncBeforeUpdate,
	},
	{
		name: "tracer_disabled",
		cfg: func() conf {
			cfg := defaultConfig()
			cfg.OktaDomain = "test.okta.com"
			cfg.OktaToken = "test-token"
			cfg.Tracer = &tracerConfig{
				Enabled: ptrTo(false),
				Logger:  lumberjack.Logger{Filename: "/var/logs/path.log"},
			}
			return cfg
		}(),
		wantErr: nil,
	},
	{
		name: "valid_path",
		cfg: func() conf {
			cfg := defaultConfig()
			cfg.OktaDomain = "test.okta.com"
			cfg.OktaToken = "test-token"
			cfg.Tracer = &tracerConfig{
				Enabled: ptrTo(true),
				Logger:  lumberjack.Logger{Filename: "okta/logs/path.log"},
			}
			return cfg
		}(),
	},
	{
		name: "invalid_path_accepted_at_config_time",
		cfg: func() conf {
			cfg := defaultConfig()
			cfg.OktaDomain = "test.okta.com"
			cfg.OktaToken = "test-token"
			cfg.Tracer = &tracerConfig{
				Enabled: ptrTo(true),
				Logger:  lumberjack.Logger{Filename: "/var/logs/path.log"},
			}
			return cfg
		}(),
	},
}

func ptrTo[T any](v T) *T { return &v }

func TestConfValidate(t *testing.T) {
	for _, test := range validateTests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error: got:%v want:%v", err, test.wantErr)
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}

var keepAliveTests = []struct {
	name    string
	input   map[string]interface{}
	want    httpcommon.WithKeepaliveSettings
	wantErr error
}{
	{
		name:  "keep_alive_none", // Default to the old behaviour of true.
		input: map[string]interface{}{},
		want:  httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_true",
		input: map[string]interface{}{
			"request.keep_alive.disable": true,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: true},
	},
	{
		name: "keep_alive_false",
		input: map[string]interface{}{
			"request.keep_alive.disable": false,
		},
		want: httpcommon.WithKeepaliveSettings{Disable: false},
	},
	{
		name: "keep_alive_invalid_max",
		input: map[string]interface{}{
			"request.keep_alive.disable":              false,
			"request.keep_alive.max_idle_connections": -1,
		},
		wantErr: errors.New("max_idle_connections must not be negative accessing 'request.keep_alive'"),
	},
}

func TestKeepAliveSetting(t *testing.T) {
	for _, test := range keepAliveTests {
		t.Run(test.name, func(t *testing.T) {
			test.input["resource.url"] = "localhost"
			cfg := config.MustNewConfigFrom(test.input)
			conf := defaultConfig()
			conf.OktaDomain = "test.domain"
			conf.OktaToken = "test_token"
			err := cfg.Unpack(&conf)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error return from Unpack: got: %v want: %v", err, test.wantErr)
			}
			if err != nil {
				return
			}
			got := conf.Request.KeepAlive.settings()
			if got != test.want {
				t.Errorf("unexpected setting for %s: got: %#v\nwant:%#v", test.name, got, test.want)
			}
		})
	}
}

func TestSensitiveHeadersConfig(t *testing.T) {
	const base = "okta_domain: example.okta.com\nokta_token: t\n"
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{name: "unset_uses_defaults", yaml: "", want: []string{"Authorization", "Proxy-Authorization", "Cookie"}},
		{name: "null_uses_defaults", yaml: "request.redirect.sensitive_headers: ~\n", want: []string{"Authorization", "Proxy-Authorization", "Cookie"}},
		{name: "empty_disables", yaml: "request.redirect.sensitive_headers: []\n", want: []string{}},
		{name: "custom_replaces_defaults", yaml: "request.redirect.sensitive_headers: [X-Api-Key]\n", want: []string{"X-Api-Key"}},
		{name: "longer_custom_replaces_defaults", yaml: "request.redirect.sensitive_headers: [A, B, C, D]\n", want: []string{"A", "B", "C", "D"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := defaultConfig()
			u, err := config.NewConfigWithYAML([]byte(base+test.yaml), "")
			if err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}
			err = u.Unpack(&c)
			if err != nil {
				t.Fatalf("failed to unpack config: %v", err)
			}

			got := sensitiveHeaders(c.Request.RedirectSensitiveHeaders)
			if !slices.Equal(got, test.want) {
				t.Errorf("sensitive headers = %v; want %v", got, test.want)
			}
		})
	}
}
