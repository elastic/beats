// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux || synthetics

package source

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProjectSource(t *testing.T) {
	t.Setenv("ELASTIC_SYNTHETICS_OFFLINE", "true")

	type testCase struct {
		name    string
		cfg     mapstr.M
		wantErr bool
	}
	testCases := []testCase{
		{
			"decode project content",
			mapstr.M{
				"content": "UEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAiAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c5VRPW/CMBDd+RWnLA0Sigt0KqJqpbZTN+iEGKzkIC6JbfkuiBTx3+uEEAGlgi7Rnf38viIESCLkR/FJ6Eis1VIjpanATBKr[...]",
			},
			false,
		},
		{
			"bad encoded content",
			mapstr.M{
				"content": "12312edasd",
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			psrc, err := dummyPSource(tc.cfg)
			if tc.wantErr {
				err = psrc.Fetch()
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			fetchAndValidate(t, psrc)
		})
	}
}

func TestFetchCaching(t *testing.T) {
	t.Setenv("ELASTIC_SYNTHETICS_OFFLINE", "true")

	cfg := mapstr.M{
		"content": "UEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAiAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c5VRPW/CMBDd+RWnLA0Sigt0KqJqpbZTN+iEGKzkIC6JbfkuiBTx3+uEEAGlgi7Rnf38viIESCLkR/FJ6Eis1VIjpanATBKrWF[...]",
	}
	psrc, err := dummyPSource(cfg)
	require.NoError(t, err)
	defer psrc.Close()

	err = psrc.Fetch()
	require.NoError(t, err)
	wdir := psrc.Workdir()
	err = psrc.Fetch()
	require.NoError(t, err)
	wdirNext := psrc.Workdir()
	require.Equal(t, wdir, wdirNext)
}

func validateFileContents(t *testing.T, dir string) {
	expected := []string{
		"examples/todos/helpers.ts",
		"examples/todos/advanced.journey.ts",
		"package.json",
	}
	for _, file := range expected {
		stat, err := os.Stat(path.Join(dir, file))
		assert.NoError(t, err)
		// Permissions should be (rwxrwx---), for running when process has changed its UID
		// note that the files themselves should not have the setuid bit set
		mode := stat.Mode().Perm()
		require.Equalf(t, mode, os.FileMode(0770), "file %v has wrong permissions: expected=%v actual=%v",
			stat.Name(), os.FileMode(0770), mode)
	}
}

func fetchAndValidate(t *testing.T, psrc *ProjectSource) {
	defer func() {
		_ = psrc.Close()
	}()
	err := psrc.Fetch()
	require.NoError(t, err)

	dir, err := os.Stat(psrc.Workdir())
	require.NoError(t, err)

	// Permissions should be (rwxrwx---), for running when process has changed its UID
	// note that the files themselves should not have the setuid bit set
	mode := dir.Mode().Perm()
	require.Equalf(t, mode, os.FileMode(0770), "file %v has wrong permissions: expected=%v actual=%v",
		dir.Name(), os.FileMode(0770), mode)

	validateFileContents(t, psrc.Workdir())
	// check if the working directory is deleted
	require.NoError(t, psrc.Close())
	_, err = os.Stat(psrc.TargetDirectory)
	require.True(t, os.IsNotExist(err), "TargetDirectory %s should have been deleted", psrc.TargetDirectory)
}

func dummyPSource(conf map[string]interface{}) (*ProjectSource, error) {
	psrc := &ProjectSource{}
	y, _ := yaml.Marshal(conf)
	c, err := config.NewConfigWithYAML(y, string(y))
	if err != nil {
		return nil, err
	}
	err = c.Unpack(psrc)
	return psrc, err
}
