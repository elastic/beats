// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package upgrade

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"

	"github.com/stretchr/testify/require"
)

func TestFallbackIsAppended(t *testing.T) {
	testCases := []struct {
		name        string
		passedBytes []string
		expectedLen int
	}{
		{"nil input", nil, 1},
		{"empty input", []string{}, 1},
		{"valid input", []string{"pgp-bytes"}, 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := appendFallbackPGP(tc.passedBytes)
			// check default fallback is passed and is very last
			require.NotNil(t, res)
			require.Equal(t, tc.expectedLen, len(res))
			require.Equal(t, download.PgpSourceURIPrefix+defaultUpgradeFallbackPGP, res[len(res)-1])
		})
	}
}
