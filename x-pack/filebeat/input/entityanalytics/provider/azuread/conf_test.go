// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azuread

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConf_Validate(t *testing.T) {
	tests := map[string]struct {
		In      conf
		WantErr string
	}{
		"default": {
			In:      defaultConf(),
			WantErr: "",
		},
		"err-invalid-intervals": {
			In: conf{
				SyncInterval:   time.Second,
				UpdateInterval: time.Second * 2,
			},
			WantErr: "sync_interval must be longer than update_interval",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotErr := tc.In.Validate()

			if tc.WantErr == "" {
				require.NoError(t, gotErr)
			} else {
				require.ErrorContains(t, gotErr, tc.WantErr)
			}
		})
	}
}
