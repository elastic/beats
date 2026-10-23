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
	"fmt"
	"os"
	"strings"
)

// UpdateMergify updates .mergify.yml to add a backport rule for releaseBranch.
// It is idempotent: existing rules for backport-{releaseBranch} are left unchanged.
func UpdateMergify(releaseBranch string) error {
	mergifyFile := ".mergify.yml"

	content, err := os.ReadFile(mergifyFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", mergifyFile, err)
	}

	labelCondition := fmt.Sprintf("label=backport-%s", releaseBranch)
	if strings.Contains(string(content), labelCondition) {
		fmt.Printf("Mergify backport rule for %s already exists in %s\n", releaseBranch, mergifyFile)
		return nil
	}

	ruleName := fmt.Sprintf("backport patches to %s branch", releaseBranch)
	newRule := fmt.Sprintf(`  - name: %s
    conditions:
      - merged
      - %s
    actions:
      backport:
        branches:
          - "%s"
`, ruleName, labelCondition, releaseBranch)

	trimmed := strings.TrimRight(string(content), "\n")
	updated := trimmed + "\n" + newRule

	err = os.WriteFile(mergifyFile, []byte(updated), 0644) //nolint:gosec // G703: fixed release tooling path
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", mergifyFile, err)
	}

	fmt.Printf("Added Mergify backport rule for branch %s in %s\n", releaseBranch, mergifyFile)
	return nil
}
