// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// Only the cluster info from a serverless project reports the serverless build flavor.
func TestClusterInfoBuildFlavorFromFixtures(t *testing.T) {
	versionRegexp := regexp.MustCompile(`cluster_info\.(\d+\.\d+\.\d+)\.json$`)

	files, err := filepath.Glob("../_meta/test/cluster_info.*.json")

	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			content, err := os.ReadFile(file)
			require.NoError(t, err)

			info, err := DeserializeData[ClusterInfo](content)
			require.NoError(t, err)

			// the serverless fixture is the only one for 8.11.0
			isServerlessFixture := versionRegexp.FindStringSubmatch(file)[1] == "8.11.0"

			require.Equal(t, isServerlessFixture, info.IsServerless())

			if isServerlessFixture {
				require.Equal(t, BuildFlavorServerless, info.Version.BuildFlavor)
			} else {
				require.Equal(t, "default", info.Version.BuildFlavor)
			}
		})
	}
}

func TestClusterInfoWithoutBuildFlavorIsNotServerless(t *testing.T) {
	info, err := DeserializeData[ClusterInfo]([]byte(`{"cluster_name": "c", "cluster_uuid": "u", "version": {"number": "7.17.0"}}`))

	require.NoError(t, err)
	require.Empty(t, info.Version.BuildFlavor)
	require.False(t, info.IsServerless())
}

func TestIsServerlessWithoutClusterInfo(t *testing.T) {
	var info *ClusterInfo

	require.False(t, info.IsServerless())
}
