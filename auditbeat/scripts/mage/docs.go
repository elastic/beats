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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

// ModuleDocs collects documentation from modules (both OSS and X-Pack).
func ModuleDocs() error {
	dirsWithModules := []string{
		devtools.OSSBeatDir(),
		devtools.XPackBeatDir(),
	}

	// Generate config.yml files for each module.
	var configFiles []string
	for _, path := range dirsWithModules {
		files, err := devtools.FindFiles(filepath.Join(path, configTemplateGlob))
		if err != nil {
			return fmt.Errorf("failed to find config templates: %w", err)
		}

		configFiles = append(configFiles, files...)
	}

	configs := make([]string, 0, len(configFiles))
	params := map[string]interface{}{
		"GOOS":      "linux",
		"GOARCH":    "amd64",
		"ArchBits":  archBits,
		"Reference": false,
	}
	for _, src := range configFiles {
		dst := strings.TrimSuffix(src, ".tmpl")
		configs = append(configs, dst)
		devtools.MustExpandFile(src, dst, params)
	}
	defer devtools.Clean(configs) //nolint:errcheck // Errors can safely be ignored here.

	// Remove old.
	for _, path := range dirsWithModules {
		if err := os.RemoveAll(filepath.Join(path, "docs/modules")); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(path, "docs/modules"), 0o755); err != nil {
			return err
		}
	}

	return collectAuditbeatDocs(dirsWithModules)
}

// collectAuditbeatDocs replaces auditbeat/scripts/docs_collector.py.
func collectAuditbeatDocs(basePaths []string) error {
	docsDir, err := devtools.DocsDir()
	if err != nil {
		return err
	}
	outputDir := filepath.Join(docsDir, "reference", "auditbeat")

	// Build module name -> module path map from all base paths.
	moduleDirs := make(map[string]string)
	for _, basePath := range basePaths {
		moduleDir := filepath.Join(basePath, "module")
		entries, err := os.ReadDir(moduleDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				moduleDirs[entry.Name()] = filepath.Join(moduleDir, entry.Name())
			}
		}
	}

	type moduleInfo struct {
		title     string
		appliesTo string
	}
	modulesList := make(map[string]moduleInfo)

	sortedModuleNames := make([]string, 0, len(moduleDirs))
	for name := range moduleDirs {
		sortedModuleNames = append(sortedModuleNames, name)
	}
	sort.Strings(sortedModuleNames)

	for _, modName := range sortedModuleNames {
		modDir := moduleDirs[modName]
		moduleDoc := filepath.Join(modDir, "_meta", "docs.md")

		if _, err := os.Stat(moduleDoc); os.IsNotExist(err) {
			continue
		}

		fieldsPath := filepath.Join(modDir, "_meta", "fields.yml")
		title, appliesTo, err := devtools.LoadModuleMeta(fieldsPath)
		if err != nil {
			return fmt.Errorf("module %s: %w", modName, err)
		}

		docContent, err := os.ReadFile(moduleDoc)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", moduleDoc, err)
		}

		var b strings.Builder

		fmt.Fprintf(&b, "---\nmapped_pages:\n  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-module-%s.html\n", modName)
		if appliesTo != "" {
			fmt.Fprintf(&b, "applies_to:\n  stack: %s\n  serverless: %s\n",
				appliesTo, devtools.GetServerlessLifecycleFromString(appliesTo))
		}
		fmt.Fprint(&b, "---\n\n")
		fmt.Fprint(&b, "% This file is generated! See auditbeat/scripts/mage/docs.go\n\n")
		fmt.Fprintf(&b, "# %s Module [auditbeat-module-%s]\n\n", title, modName)
		b.Write(docContent)

		modulesList[modName] = moduleInfo{title: title, appliesTo: appliesTo}

		// Add example config if present.
		configFile := filepath.Join(modDir, "_meta", "config.yml")
		if configData, err := os.ReadFile(configFile); err == nil {
			fmt.Fprintf(&b, "\n## Example configuration [_example_configuration]\n\n")
			fmt.Fprintf(&b, "The %s module supports the common configuration options that are described under [configuring Auditbeat](/reference/auditbeat/configuration-auditbeat.md). Here is an example configuration:\n\n", title)
			fmt.Fprint(&b, "```yaml\nauditbeat.modules:\n")
			b.WriteString(strings.TrimSpace(string(configData)))
			fmt.Fprint(&b, "\n```\n\n")
		}

		// Iterate over datasets.
		var moduleLinks strings.Builder
		datasetEntries, err := os.ReadDir(modDir)
		if err != nil {
			return err
		}
		sortedDatasets := make([]string, 0, len(datasetEntries))
		for _, e := range datasetEntries {
			if e.IsDir() {
				sortedDatasets = append(sortedDatasets, e.Name())
			}
		}
		sort.Strings(sortedDatasets)

		for _, dataset := range sortedDatasets {
			datasetDocs := filepath.Join(modDir, dataset, "_meta", "docs.md")
			if _, err := os.Stat(datasetDocs); os.IsNotExist(err) {
				continue
			}

			fmt.Fprintf(&moduleLinks, "* [%s](/reference/auditbeat/auditbeat-dataset-%s-%s.md)\n", dataset, modName, dataset)

			var dsB strings.Builder
			fmt.Fprintf(&dsB, "---\nmapped_pages:\n  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-dataset-%s-%s.html\n", modName, dataset)
			if appliesTo != "" {
				fmt.Fprintf(&dsB, "applies_to:\n  stack: %s\n  serverless: %s\n",
					appliesTo, devtools.GetServerlessLifecycleFromString(appliesTo))
			}
			fmt.Fprint(&dsB, "---\n\n")
			fmt.Fprint(&dsB, "% This file is generated! See auditbeat/scripts/mage/docs.go\n\n")
			fmt.Fprintf(&dsB, "# %s %s dataset [auditbeat-dataset-%s-%s]\n\n", title, dataset, modName, dataset)

			dsDocContent, err := os.ReadFile(datasetDocs)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", datasetDocs, err)
			}
			dsB.Write(dsDocContent)

			fmt.Fprintf(&dsB, "\n## Fields [_fields]\n\nFor a description of each field in the dataset, see the [exported fields](/reference/auditbeat/exported-fields-%s.md) section.\n\nHere is an example document generated by this dataset:\n\n```json\n", modName)

			dataFile := filepath.Join(modDir, dataset, "_meta", "data.json")
			if dataContent, err := os.ReadFile(dataFile); err == nil {
				dsB.WriteString(strings.TrimSpace(string(dataContent)))
				fmt.Fprint(&dsB, "\n```\n")
			}

			dsOutPath := filepath.Join(outputDir, fmt.Sprintf("auditbeat-dataset-%s-%s.md", modName, dataset))
			if err := os.WriteFile(dsOutPath, []byte(dsB.String()), 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", dsOutPath, err)
			}
		}

		if moduleLinks.Len() > 0 {
			fmt.Fprint(&b, "## Datasets [_datasets]\n\nThe following datasets are available:\n\n")
			b.WriteString(moduleLinks.String())
			b.WriteString("\n")
		}

		modOutPath := filepath.Join(outputDir, fmt.Sprintf("auditbeat-module-%s.md", modName))
		if err := os.WriteFile(modOutPath, []byte(b.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", modOutPath, err)
		}
	}

	// Write module list page.
	var listB strings.Builder
	listB.WriteString(`---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/auditbeat/current/auditbeat-modules.html
applies_to:
  stack: ga
  serverless: ga
---

% This file is generated! See auditbeat/scripts/mage/docs.go

# Modules [auditbeat-modules]

This section contains detailed information about the metric collecting modules contained in Auditbeat. More details about each module can be found under the links below.

`)

	sortedKeys := make([]string, 0, len(modulesList))
	for name := range modulesList {
		sortedKeys = append(sortedKeys, name)
	}
	sort.Strings(sortedKeys)

	for _, m := range sortedKeys {
		info := modulesList[m]
		fmt.Fprintf(&listB, "* [%s](/reference/auditbeat/auditbeat-module-%s.md)", info.title, m)
		if info.appliesTo != "" && info.appliesTo != "ga" {
			fmt.Fprintf(&listB, " {applies_to}`stack: %s`", info.appliesTo)
		}
		listB.WriteString("\n")
	}
	listB.WriteString("\n")

	listPath := filepath.Join(outputDir, "auditbeat-modules.md")
	return os.WriteFile(listPath, []byte(listB.String()), 0o644)
}

// FieldDocs generates exported-fields.md containing all fields
// (including x-pack).
func FieldDocs() error {
	inputs := []string{
		devtools.OSSBeatDir("module"),
		devtools.XPackBeatDir("module"),
	}
	output := devtools.CreateDir("build/fields/fields.all.yml")
	if err := devtools.GenerateFieldsYAMLTo(output, inputs...); err != nil {
		return err
	}
	return devtools.Docs.FieldDocs(output)
}
