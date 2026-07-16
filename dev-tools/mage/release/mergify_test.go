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
	if err := os.WriteFile(mergifyFile, []byte(mergifyFixture), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore cwd: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := UpdateMergify("9.5"); err != nil {
		t.Fatalf("UpdateMergify failed: %v", err)
	}

	content, err := os.ReadFile(mergifyFile)
	if err != nil {
		t.Fatalf("failed to read mergify file: %v", err)
	}
	body := string(content)
	if !strings.Contains(body, "label=backport-9.5") {
		t.Errorf("expected backport-9.5 label in mergify file, got:\n%s", body)
	}
	if !strings.Contains(body, `branches:`) || !strings.Contains(body, `- "9.5"`) {
		t.Errorf("expected backport branch 9.5 in mergify file, got:\n%s", body)
	}
}

func TestUpdateMergifyIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	mergifyFile := filepath.Join(tmpDir, ".mergify.yml")
	if err := os.WriteFile(mergifyFile, []byte(mergifyFixture), 0644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore cwd: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	if err := UpdateMergify("9.5"); err != nil {
		t.Fatalf("first UpdateMergify failed: %v", err)
	}
	afterFirst, err := os.ReadFile(mergifyFile)
	if err != nil {
		t.Fatalf("failed to read mergify file: %v", err)
	}

	if err := UpdateMergify("9.5"); err != nil {
		t.Fatalf("second UpdateMergify failed: %v", err)
	}
	afterSecond, err := os.ReadFile(mergifyFile)
	if err != nil {
		t.Fatalf("failed to read mergify file: %v", err)
	}

	if string(afterFirst) != string(afterSecond) {
		t.Errorf("idempotent UpdateMergify should not modify file twice")
	}
}
