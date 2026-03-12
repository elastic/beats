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
	"path/filepath"
	"regexp"
	"strings"
)

// UpdateVersion updates the version in libbeat/version/version.go
func UpdateVersion(newVersion string) error {
	versionFile := "libbeat/version/version.go"

	content, err := os.ReadFile(versionFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", versionFile, err)
	}

	// Pattern: const defaultBeatVersion = "X.Y.Z"
	pattern := regexp.MustCompile(`const defaultBeatVersion = ".*"`)
	replacement := fmt.Sprintf(`const defaultBeatVersion = "%s"`, newVersion)

	newContent := pattern.ReplaceAllString(string(content), replacement)

	if newContent == string(content) {
		return fmt.Errorf("version pattern not found in %s", versionFile)
	}

	err = os.WriteFile(versionFile, []byte(newContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", versionFile, err)
	}

	fmt.Printf("Updated version to %s in %s\n", newVersion, versionFile)
	return nil
}

// UpdateDocs updates version references in documentation and K8s manifests
func UpdateDocs(newVersion string) error {
	// Parse version (e.g., "9.3.0" -> major=9, minor=3, patch=0)
	parts := strings.Split(newVersion, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid version format: %s", newVersion)
	}
	majorMinor := parts[0] + "." + parts[1]

	files := map[string][]replacementRule{
		"libbeat/docs/version.asciidoc": {
			{
				pattern: regexp.MustCompile(`:stack-version:\s*\d+\.\d+\.\d+`),
				replacement: fmt.Sprintf(":stack-version: %s", newVersion),
			},
			{
				pattern: regexp.MustCompile(`:doc-branch:\s*\d+\.\d+`),
				replacement: fmt.Sprintf(":doc-branch: %s", majorMinor),
			},
		},
		"deploy/kubernetes/metricbeat-kubernetes.yaml": {
			{
				pattern: regexp.MustCompile(`docker\.elastic\.co/beats/metricbeat:\d+\.\d+\.\d+`),
				replacement: fmt.Sprintf("docker.elastic.co/beats/metricbeat:%s", newVersion),
			},
		},
		"deploy/kubernetes/filebeat-kubernetes.yaml": {
			{
				pattern: regexp.MustCompile(`docker\.elastic\.co/beats/filebeat:\d+\.\d+\.\d+`),
				replacement: fmt.Sprintf("docker.elastic.co/beats/filebeat:%s", newVersion),
			},
		},
		"deploy/kubernetes/auditbeat-kubernetes.yaml": {
			{
				pattern: regexp.MustCompile(`docker\.elastic\.co/beats/auditbeat:\d+\.\d+\.\d+`),
				replacement: fmt.Sprintf("docker.elastic.co/beats/auditbeat:%s", newVersion),
			},
		},
		"README.md": {
			{
				// Update branch references like /7.x/ -> /7.9/
				pattern: regexp.MustCompile(`/\d+\.x/`),
				replacement: fmt.Sprintf("/%s/", majorMinor),
			},
		},
	}

	for filePath, rules := range files {
		if err := applyReplacements(filePath, rules); err != nil {
			return err
		}
	}

	fmt.Printf("Updated documentation files to version %s\n", newVersion)
	return nil
}

// UpdateTestEnv updates docker-compose.yml files with new version
func UpdateTestEnv(latestVersion, currentVersion string) error {
	files := []string{
		"testing/environments/docker/elasticsearch_kerberos/Dockerfile",
		"testing/environments/latest.yml",
		"x-pack/metricbeat/docker-compose.yml",
		"metricbeat/module/logstash/docker-compose.yml",
		"metricbeat/docker-compose.yml",
	}

	// Pattern: docker.elastic.co/...:X.Y.Z
	pattern := regexp.MustCompile(fmt.Sprintf(`docker\.elastic\.co/([^:]+):%s`, regexp.QuoteMeta(latestVersion)))
	replacement := fmt.Sprintf("docker.elastic.co/$1:%s", currentVersion)

	for _, filePath := range files {
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("Skipping %s (file does not exist)\n", filePath)
			continue
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		newContent := pattern.ReplaceAllString(string(content), replacement)

		if newContent != string(content) {
			err = os.WriteFile(filePath, []byte(newContent), 0644)
			if err != nil {
				return fmt.Errorf("failed to write %s: %w", filePath, err)
			}
			fmt.Printf("Updated test environment in %s\n", filePath)
		}
	}

	fmt.Printf("Updated test environment files from %s to %s\n", latestVersion, currentVersion)
	return nil
}

// replacementRule defines a pattern and its replacement
type replacementRule struct {
	pattern     *regexp.Regexp
	replacement string
}

// applyReplacements applies a set of replacement rules to a file
func applyReplacements(filePath string, rules []replacementRule) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("Skipping %s (file does not exist)\n", filePath)
		return nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	newContent := string(content)
	for _, rule := range rules {
		newContent = rule.pattern.ReplaceAllString(newContent, rule.replacement)
	}

	if newContent != string(content) {
		err = os.WriteFile(filePath, []byte(newContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}
		fmt.Printf("Updated %s\n", filePath)
	}

	return nil
}

// RunMakeUpdate runs 'make update' in the repository
func RunMakeUpdate() error {
	fmt.Println("Running 'make update'...")
	// This will be implemented later with proper exec.Command
	// For now, just a placeholder
	return fmt.Errorf("not implemented yet - requires exec.Command integration")
}
