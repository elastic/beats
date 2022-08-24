// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"testing"

	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfig(t *testing.T) {
	testCases := []struct {
		name        string
		userConfig  map[string]interface{}
		expected    *config
		expectedErr string
	}{
		{
			"defaults",
			map[string]interface{}{},
			&config{
				ListenAddress: "localhost:5044",
				Versions:      []string{"v1", "v2"},
			},
			"",
		},
		{
			"validate version",
			map[string]interface{}{
				"versions": []string{"v3"},
			},
			nil,
			`invalid lumberjack version "v3"`,
		},
		{
			"validate keepalive",
			map[string]interface{}{
				"keepalive": "-1s",
			},
			nil,
			`requires duration >= 0`,
		},
		{
			"validate max_connections",
			map[string]interface{}{
				"max_connections": -1,
			},
			nil,
			`requires value >= 0 accessing 'max_connections'`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			c := conf.MustNewConfigFrom(tc.userConfig)

			var ljConf config
			err := c.Unpack(&ljConf)

			if tc.expectedErr != "" {
				require.Error(t, err, "expected error: %s", tc.expectedErr)
				require.Contains(t, err.Error(), tc.expectedErr)
				return
			}

			require.Equal(t, *tc.expected, ljConf)
		})
	}
}
