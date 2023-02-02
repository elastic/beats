// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package entityanalytics

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/provider/azuread"
)

func TestConf_Validate(t *testing.T) {
	tests := map[string]struct {
		In      conf
		WantErr string
	}{
		"ok-provider-azure": {
			In: conf{
				Provider: azuread.Name,
			},
		},
		"err-provider-unknown": {
			In: conf{
				Provider: "unknown",
			},
			WantErr: ErrProviderUnknown.Error(),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotErr := tc.In.Validate()

			if tc.WantErr != "" {
				require.ErrorContains(t, gotErr, tc.WantErr)
			} else {
				require.NoError(t, gotErr)
			}
		})
	}
}
