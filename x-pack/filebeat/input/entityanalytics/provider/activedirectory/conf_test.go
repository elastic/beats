// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package activedirectory

import (
	"testing"
	"time"
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
