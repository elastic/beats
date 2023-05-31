// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package source

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestSimpleCases(t *testing.T) {
	type testCase struct {
		name         string
		cfg          mapstr.M
		tlsServer    bool
		wantFetchErr bool
	}
	testCases := []testCase{
		{
			"basics",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
			},
			false,
			false,
		},
		{
			"targetdir",
			mapstr.M{
				"folder":           "/",
				"retries":          3,
				"target_directory": filepath.Join(os.TempDir(), "synthetics", "blah"),
			},
			false,
			false,
		},
		{
			"auth success",
			mapstr.M{
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "testpass",
			},
			false,
			false,
		},
		{
			"auth failure",
			mapstr.M{
				"folder":   "/",
				"retries":  3,
				"username": "testuser",
				"password": "badpass",
			},
			false,
			true,
		},
		{
			"ssl ignore cert errors",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
				"ssl": mapstr.M{
					"enabled":           "true",
					"verification_mode": "none",
				},
			},
			true,
			false,
		},
		{
			"bad ssl",
			mapstr.M{
				"folder":  "/",
				"retries": 3,
				"ssl": mapstr.M{
					"enabled":                 "true",
					"certificate_authorities": []string{},
				},
			},
			true,
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.cfg["url"] = "gibberish"
			_, err := dummyZus(tc.cfg)
			require.Error(t, err)
			require.Regexp(t, ErrZipURLUnsupportedType, err)
		})
	}
}

func dummyZus(conf map[string]interface{}) (*ZipURLSource, error) {
	zus := &ZipURLSource{}
	y, _ := yaml.Marshal(conf)
	c, err := config.NewConfigWithYAML(y, string(y))
	if err != nil {
		return nil, err
	}
	err = c.Unpack(zus)
	return zus, err
}
