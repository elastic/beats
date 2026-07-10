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
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const docsURLMarker = "# Docs: https://www.elastic.co/guide/en/beats/"

var (
	k8sImageVersionPattern = regexp.MustCompile(`(image: docker\.elastic\.co/[^:]+):\d+\.\d+\.\d+`)
	dockerImageTagPattern  = regexp.MustCompile(`(docker\.elastic\.co/[^:]+):\d+\.\d+\.\d+`)
	composeDefaultPattern  = regexp.MustCompile(`:-\d+\.\d+\.\d+}`)
)

// DocsUpdateOptions configures documentation and manifest updates.
type DocsUpdateOptions struct {
	BaseBranch     string
	CurrentVersion string
	ReleaseBranch  string
}

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
	targetLine := replacement

	if strings.Contains(string(content), targetLine) {
		fmt.Printf("Version already set to %s in %s\n", newVersion, versionFile)
		return nil
	}

	newContent := pattern.ReplaceAllString(string(content), replacement)

	if newContent == string(content) {
		return fmt.Errorf("version pattern not found in %s", versionFile)
	}

	err = os.WriteFile(versionFile, []byte(newContent), 0644) //nolint:gosec // G703: fixed release tooling path, not user-controlled
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", versionFile, err)
	}

	fmt.Printf("Updated version to %s in %s\n", newVersion, versionFile)
	return nil
}

// UpdateDocs updates version references using release-branch defaults.
func UpdateDocs(newVersion string) error {
	releaseBranch := inferReleaseBranch(newVersion)
	return UpdateDocsWithOptions(DocsUpdateOptions{
		BaseBranch:     releaseBranch,
		CurrentVersion: newVersion,
		ReleaseBranch:  releaseBranch,
	})
}

// UpdateDocsWithOptions updates documentation, K8s manifests, and docs URLs.
// Mirrors ingest-dev release_scripts/beats.mak update-docs.
func UpdateDocsWithOptions(opts DocsUpdateOptions) error {
	if opts.CurrentVersion == "" {
		return fmt.Errorf("CurrentVersion is required")
	}
	if opts.ReleaseBranch == "" {
		opts.ReleaseBranch = inferReleaseBranch(opts.CurrentVersion)
	}
	if opts.BaseBranch == "" {
		opts.BaseBranch = opts.ReleaseBranch
	}

	docBranch := opts.BaseBranch
	if docBranch == "main" || docBranch == "current" {
		docBranch = opts.ReleaseBranch
	}

	if err := updateDocsURLs(opts.BaseBranch, opts.ReleaseBranch); err != nil {
		return err
	}

	versionRules := []replacementRule{
		{
			pattern:     regexp.MustCompile(`:stack-version:\s*\d+\.\d+\.\d+`),
			replacement: fmt.Sprintf(":stack-version: %s", opts.CurrentVersion),
		},
		{
			pattern:     regexp.MustCompile(`:doc-branch:\s*\S+`),
			replacement: fmt.Sprintf(":doc-branch: %s", docBranch),
		},
	}
	if err := applyReplacements("libbeat/docs/version.asciidoc", versionRules); err != nil {
		return err
	}

	k8sFiles := []string{
		"deploy/kubernetes/metricbeat-kubernetes.yaml",
		"deploy/kubernetes/filebeat-kubernetes.yaml",
		"deploy/kubernetes/heartbeat-kubernetes.yaml",
		"deploy/kubernetes/auditbeat-kubernetes.yaml",
	}
	k8sRule := replacementRule{
		pattern:     k8sImageVersionPattern,
		replacement: fmt.Sprintf("$1:%s", opts.CurrentVersion),
	}
	for _, filePath := range k8sFiles {
		if err := applyReplacements(filePath, []replacementRule{k8sRule}); err != nil {
			return err
		}
	}

	readmeRule := replacementRule{
		pattern:     regexp.MustCompile(regexp.QuoteMeta("/"+opts.BaseBranch+"/")),
		replacement: "/" + opts.ReleaseBranch + "/",
	}
	if err := applyReplacements("README.md", []replacementRule{readmeRule}); err != nil {
		return err
	}

	fmt.Printf("Updated documentation files to version %s\n", opts.CurrentVersion)
	return nil
}

// UpdateTestEnv updates test environment files.
// latestVersion maps to LATEST and currentVersion maps to CURRENT in beats.mak update-test-env.
func UpdateTestEnv(latestVersion, currentVersion string) error {
	kerberosFile := "testing/environments/docker/elasticsearch_kerberos/Dockerfile"
	kerberosRule := replacementRule{
		pattern:     dockerImageTagPattern,
		replacement: fmt.Sprintf("$1:%s", currentVersion),
	}
	if err := applyReplacements(kerberosFile, []replacementRule{kerberosRule}); err != nil {
		return err
	}

	latestYmlRule := replacementRule{
		pattern:     dockerImageTagPattern,
		replacement: fmt.Sprintf("$1:%s", latestVersion),
	}
	if err := applyReplacements("testing/environments/latest.yml", []replacementRule{latestYmlRule}); err != nil {
		return err
	}

	composeDefaultRule := replacementRule{
		pattern:     composeDefaultPattern,
		replacement: fmt.Sprintf(":-%s}", latestVersion),
	}
	composeFiles := []string{
		"x-pack/metricbeat/docker-compose.yml",
		"metricbeat/module/logstash/docker-compose.yml",
		"metricbeat/docker-compose.yml",
	}
	for _, filePath := range composeFiles {
		if err := applyReplacements(filePath, []replacementRule{composeDefaultRule}); err != nil {
			return err
		}
	}

	fmt.Printf("Updated test environment files (latest=%s, current=%s)\n", latestVersion, currentVersion)
	return nil
}

func updateDocsURLs(baseBranch, releaseBranch string) error {
	if baseBranch == "" || releaseBranch == "" || baseBranch == releaseBranch {
		return nil
	}

	oldSegment := "/" + baseBranch + "/"
	newSegment := "/" + releaseBranch + "/"
	updatedFiles := 0

	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(content), docsURLMarker) {
			return nil
		}

		newContent := strings.ReplaceAll(string(content), oldSegment, newSegment)
		if newContent == string(content) {
			return nil
		}

		if err := os.WriteFile(path, []byte(newContent), 0644); err != nil { //nolint:gosec // G703: fixed release tooling path
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Printf("Updated docs URLs in %s\n", path)
		updatedFiles++
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update docs URLs: %w", err)
	}
	if updatedFiles > 0 {
		fmt.Printf("Updated docs URLs in %d files (%s -> %s)\n", updatedFiles, baseBranch, releaseBranch)
	}
	return nil
}

// replacementRule defines a pattern and its replacement
type replacementRule struct {
	pattern     *regexp.Regexp
	replacement string
}

// applyReplacements applies a set of replacement rules to a file
func applyReplacements(filePath string, rules []replacementRule) error {
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
		err = os.WriteFile(filePath, []byte(newContent), 0644) //nolint:gosec // G703: fixed release tooling path, not user-controlled
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}
		fmt.Printf("Updated %s\n", filePath)
	}

	return nil
}

// RunMakeUpdate runs 'make update' in the repository.
func RunMakeUpdate() error {
	fmt.Println("Running 'make update'...")
	cmd := exec.Command("make", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make update failed: %w", err)
	}
	fmt.Println("Completed 'make update'")
	return nil
}
