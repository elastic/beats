// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package mage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitAlternateObjectDirMounts(t *testing.T) {
	tmp := t.TempDir()
	objectsDir := filepath.Join(tmp, "repo", ".git", "objects")
	containerObjectsDir := "/go/src/github.com/elastic/beats/.git/objects"
	absoluteAlternate := filepath.Join(tmp, "mirror.git", "objects")
	relativeAlternate := filepath.Join(tmp, "other.git", "objects")
	missingAlternate := filepath.Join(tmp, "missing.git", "objects")

	require.NoError(t, os.MkdirAll(objectsDir, 0755), "objects dir setup must succeed")
	require.NoError(t, os.MkdirAll(absoluteAlternate, 0755), "absolute alternate setup must succeed")
	require.NoError(t, os.MkdirAll(relativeAlternate, 0755), "relative alternate setup must succeed")

	relativeEntry, err := filepath.Rel(objectsDir, relativeAlternate)
	require.NoError(t, err, "relative alternate path must be computed")
	missingEntry, err := filepath.Rel(objectsDir, missingAlternate)
	require.NoError(t, err, "missing alternate path must be computed")

	mounts := gitAlternateObjectDirMounts(
		objectsDir,
		containerObjectsDir,
		[]byte(absoluteAlternate+"\n"+relativeEntry+"\n"+missingEntry+"\n# comment\n\n"+absoluteAlternate+"\n"),
	)

	expectedRelativeContainerPath := filepath.ToSlash(filepath.Clean(filepath.Join(containerObjectsDir, relativeEntry)))
	assert.Equal(
		t,
		[]dockerVolumeMount{
			{
				hostPath:      filepath.Clean(absoluteAlternate),
				containerPath: filepath.ToSlash(filepath.Clean(absoluteAlternate)),
				readOnly:      true,
			},
			{
				hostPath:      filepath.Clean(relativeAlternate),
				containerPath: expectedRelativeContainerPath,
				readOnly:      true,
			},
		},
		mounts,
		"expected existing alternates to be mounted read-only without duplicates",
	)
}

func TestContainerPathForHostPath(t *testing.T) {
	repoRoot := filepath.Join(string(filepath.Separator), "tmp", "checkout")
	containerRepoRoot := "/go/src/github.com/elastic/beats"

	tests := []struct {
		name     string
		hostPath string
		expected string
	}{
		{
			name:     "inside repo",
			hostPath: filepath.Join(repoRoot, ".git", "objects"),
			expected: "/go/src/github.com/elastic/beats/.git/objects",
		},
		{
			name:     "outside repo",
			hostPath: filepath.Join(string(filepath.Separator), "opt", "git-mirrors", "beats.git", "objects"),
			expected: "/opt/git-mirrors/beats.git/objects",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(
				t,
				tc.expected,
				containerPathForHostPath(tc.hostPath, repoRoot, containerRepoRoot),
				"expected host paths under the repo to map into the container checkout",
			)
		})
	}
}
