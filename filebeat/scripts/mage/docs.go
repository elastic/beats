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

// CollectDocs collects and generates documentation from each filebeat module.
// Replaces filebeat/scripts/docs_collector.py.
func CollectDocs() error {
	ossModules, err := filepath.Glob(devtools.OSSBeatDir("module", "*"))
	if err != nil {
		return fmt.Errorf("failed to glob OSS modules: %w", err)
	}
	xpackModules, err := filepath.Glob(devtools.XPackBeatDir("module", "*"))
	if err != nil {
		return fmt.Errorf("failed to glob x-pack modules: %w", err)
	}

	allModules := make([]string, 0, len(ossModules)+len(xpackModules))
	allModules = append(allModules, ossModules...)
	allModules = append(allModules, xpackModules...)
	sort.Strings(allModules)

	docsDir, err := devtools.DocsDir()
	if err != nil {
		return err
	}
	outputDir := filepath.Join(docsDir, "reference", "filebeat")

	type moduleInfo struct {
		title     string
		appliesTo string
	}
	modulesList := make(map[string]moduleInfo)

	for _, modPath := range allModules {
		modName := filepath.Base(modPath)
		moduleDoc := filepath.Join(modPath, "_meta", "docs.md")

		if _, err := os.Stat(moduleDoc); os.IsNotExist(err) {
			continue
		}

		fieldsPath := filepath.Join(modPath, "_meta", "fields.yml")
		title, appliesTo, err := devtools.LoadModuleMeta(fieldsPath)
		if err != nil {
			return fmt.Errorf("module %s: %w", modName, err)
		}

		docContent, err := os.ReadFile(moduleDoc)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", moduleDoc, err)
		}

		var b strings.Builder

		fmt.Fprintf(&b, "---\nmapped_pages:\n  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-%s.html\n", modName)
		if appliesTo != "" {
			fmt.Fprintf(&b, "applies_to:\n  stack: %s\n  serverless: %s\n",
				appliesTo, devtools.GetServerlessLifecycleFromString(appliesTo))
		}
		fmt.Fprint(&b, "---\n\n")
		fmt.Fprint(&b, "% This file is generated! See filebeat/scripts/mage/docs.go\n\n")
		fmt.Fprintf(&b, "# %s module [filebeat-module-%s]\n\n", title, modName)
		b.Write(docContent)
		fmt.Fprintf(&b, "\n## Fields [_fields]\n\nFor a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-%s.md) section.\n", modName)

		outPath := filepath.Join(outputDir, fmt.Sprintf("filebeat-module-%s.md", modName))
		if err := os.WriteFile(outPath, []byte(b.String()), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", outPath, err)
		}

		modulesList[modName] = moduleInfo{title: title, appliesTo: appliesTo}
	}

	var listB strings.Builder
	listB.WriteString(`---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-modules.html
applies_to:
  stack: ga
  serverless: ga
---

# Modules [filebeat-modules]

This section contains an [overview](/reference/filebeat/filebeat-modules-overview.md) of the Filebeat modules feature as well as details about each of the currently supported modules.

Filebeat modules require Elasticsearch 5.2 or later.

::::{note}
While {{filebeat}} modules are still supported, we recommend {{agent}} integrations over {{filebeat}} modules. Integrations provide a streamlined way to connect data from a variety of vendors to the {{stack}}. Refer to the [full list of integrations](https://www.elastic.co/integrations/data-integrations). For more information, please refer to the [{{beats}} vs {{agent}} comparison documentation](docs-content://reference/fleet/index.md).
::::


* [*Modules overview*](/reference/filebeat/filebeat-modules-overview.md)
`)

	sortedModules := make([]string, 0, len(modulesList))
	for name := range modulesList {
		sortedModules = append(sortedModules, name)
	}
	sort.Strings(sortedModules)

	for _, m := range sortedModules {
		info := modulesList[m]
		fmt.Fprintf(&listB, "* [*%s module*](/reference/filebeat/filebeat-module-%s.md)", info.title, m)
		if info.appliesTo != "" && info.appliesTo != "ga" {
			fmt.Fprintf(&listB, " {applies_to}`stack: %s`", info.appliesTo)
		}
		listB.WriteString("\n")
	}
	listB.WriteString("\n")

	listPath := filepath.Join(outputDir, "filebeat-modules.md")
	return os.WriteFile(listPath, []byte(listB.String()), 0o644)
}
