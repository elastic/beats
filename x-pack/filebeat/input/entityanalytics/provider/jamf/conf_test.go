// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package jamf

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

var validateTests = []struct {
	name    string
	cfg     conf
	wantErr error
}{
	{
		name:    "default",
		cfg:     defaultConfig(),
		wantErr: nil,
	},
	{
		name: "invalid_sync_interval",
		cfg: conf{
			SyncInterval:   0,
			UpdateInterval: time.Second * 2,
		},
		wantErr: errInvalidSyncInterval,
	},
	{
		name: "invalid_update_interval",
		cfg: conf{
			SyncInterval:   time.Second,
			UpdateInterval: 0,
		},
		wantErr: errInvalidUpdateInterval,
	},
	{
		name: "invalid_relative_intervals",
		cfg: conf{
			SyncInterval:   time.Second,
			UpdateInterval: time.Second * 2,
		},
		wantErr: errSyncBeforeUpdate,
	},
}

func TestConfValidate(t *testing.T) {
	for _, test := range validateTests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.Validate()
			if err != test.wantErr {
				t.Errorf("unexpected error: got:%v want:%v", err, test.wantErr)
			}
		})
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
			conf.JamfTenant = "test.domain"
			conf.JamfUsername = "test_user"
			conf.JamfPassword = "test_password"
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
