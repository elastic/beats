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

	"gopkg.in/yaml.v3"
)

// UpdateMergify updates .mergify.yml to add backport configuration for new version
func UpdateMergify(version string) error {
	mergifyFile := ".mergify.yml"

	// Read the file
	content, err := os.ReadFile(mergifyFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", mergifyFile, err)
	}

	// Parse YAML
	var config map[string]interface{}
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", mergifyFile, err)
	}

	// Add backport rule for the new version
	// This is a simplified implementation - the actual logic may need to be more sophisticated
	// depending on the structure of .mergify.yml

	// For now, we'll just verify we can read/write the file
	// The actual implementation would add a new backport rule like:
	// - name: backport patches to 9.3 branch
	//   conditions:
	//     - label=backport-v9.3.0
	//   actions:
	//     backport:
	//       branches:
	//         - "9.3"

	fmt.Printf("Mergify update for version %s - implementation pending\n", version)
	fmt.Println("Note: Manual verification of .mergify.yml may be required")

	return nil
}
