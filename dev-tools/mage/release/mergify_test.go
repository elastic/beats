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

package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const mergifyFixture = `pull_request_rules:
  - name: backport patches to 9.4 branch
    conditions:
      - merged
      - label=backport-9.4
    actions:
      backport:
        branches:
          - "9.4"
`

func TestUpdateMergifyAppendsRule(t *testing.T) {
	tmpDir := t.TempDir()
	mergifyFile := filepath.Join(tmpDir, ".mergify.yml")
	require.NoError(t, os.WriteFile(mergifyFile, []byte(mergifyFixture), 0644), "failed to write fixture")

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get cwd")
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore cwd: %v", err)
		}
	}()
	require.NoError(t, os.Chdir(tmpDir), "failed to chdir")

	require.NoError(t, UpdateMergify("9.5"), "UpdateMergify failed")

	content, err := os.ReadFile(mergifyFile)
	require.NoError(t, err, "failed to read mergify file")
	body := string(content)
	require.Contains(t, body, "label=backport-9.5")
	if !strings.Contains(body, `branches:`) || !strings.Contains(body, `- "9.5"`) {
		t.Errorf("expected backport branch 9.5 in mergify file, got:\n%s", body)
	}
}

func TestUpdateMergifyIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	mergifyFile := filepath.Join(tmpDir, ".mergify.yml")
	require.NoError(t, os.WriteFile(mergifyFile, []byte(mergifyFixture), 0644), "failed to write fixture")

	origDir, err := os.Getwd()
	require.NoError(t, err, "failed to get cwd")
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore cwd: %v", err)
		}
	}()
	require.NoError(t, os.Chdir(tmpDir), "failed to chdir")

	require.NoError(t, UpdateMergify("9.5"), "first UpdateMergify failed")
	afterFirst, err := os.ReadFile(mergifyFile)
	require.NoError(t, err, "failed to read mergify file")

	require.NoError(t, UpdateMergify("9.5"), "second UpdateMergify failed")
	afterSecond, err := os.ReadFile(mergifyFile)
	require.NoError(t, err, "failed to read mergify file")

	if string(afterFirst) != string(afterSecond) {
		t.Errorf("idempotent UpdateMergify should not modify file twice")
	}
}
