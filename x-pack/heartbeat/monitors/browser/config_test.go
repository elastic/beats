// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browser

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/x-pack/heartbeat/monitors/browser/source"
)

func TestConfig_Validate(t *testing.T) {
	testSource := source.Source{Inline: &source.InlineSource{Script: "//something"}}

	tests := []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{
			"no error",
			&Config{Id: "myid", Name: "myname", Source: &testSource},
			nil,
		},
		{
			"no id",
			&Config{Name: "myname", Source: &testSource},
			ErrIdRequired,
		},
		{
			"no name",
			&Config{Id: "myid", Source: &testSource},
			ErrNameRequired,
		},
		{
			"no source",
			&Config{Id: "myid", Name: "myname"},
			ErrSourceRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
