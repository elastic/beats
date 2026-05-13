// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package okta

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
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
		name: "invalid_path",
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
		wantErr: errors.New(`request tracer path must be within "okta" path`),
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
