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
	_ "embed"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elastic/beats/v7/dev-tools/mage"
)

const moduleDocsGlob = "module/*/_meta/docs.md"

var moduleNameRegex = regexp.MustCompile(`module\/(.*)\/_meta\/docs.md`)

//go:embed templates/moduleList.tmpl
var modulesListTmpl string

// var modulesListTmpl = `
// ---
// mapped_pages:
//   - https://www.elastic.co/guide/en/beats/winlogbeat/current/winlogbeat-modules.html
// ---

// # Modules [winlogbeat-modules]

// ::::{note}
// Winlogbeat modules have changed in 8.0.0 to use Elasticsearch Ingest Node for processing. If you are upgrading from 7.x please review the documentation and see the default configuration file.
// ::::

// This section contains detailed information about the available Windows event log processing modules contained in Winlogbeat. More details about each module can be found in the module’s documentation.

// Winlogbeat modules are implemented using Elasticsearch Ingest Node pipelines. The events receive their transformations within Elasticsearch. All events are sent through Winlogbeat’s "routing" pipeline that routes events to specific module pipelines based on their ` + "`winlog.channel`" + `value.

// Winlogbeat’s default config file contains the option to send all events to the routing pipeline. If you remove this option then the module processing will not be applied.

// ` + "```yaml" + `
// output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing
// ` + "```" + `

// The general goal of each module is to transform events by renaming fields to comply with the [Elastic Common Schema](ecs://reference/index.md) (ECS). The modules may also apply additional categorization, tagging, and parsing as necessary.

// ::::{note}
// The provided modules only support events in English. For more information about how to configure the language in ` + "`winlogbeat`" + `, refer to [Winlogbeat](/reference/winlogbeat/configuration-winlogbeat-options.md).
// ::::

// ## Setup of Ingest Node pipelines [winlogbeat-modules-setup]

// Winlogbeat’s Ingest Node pipelines must be installed to Elasticsearch if you want to apply the module processing to events. The simplest way to get started is to use the Elasticsearch output and Winlogbeat will automatically install the pipelines when it first connects to Elasticsearch.

// Installation Methods

// 1. [On connection to {{es}}](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-auto)
// 2. [setup command](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-setup)
// 3. [Manually install pipelines](/reference/winlogbeat/load-ingest-pipelines.md#winlogbeat-load-pipeline-manual)

// ## Usage with Forwarded Events [_usage_with_forwarded_events]

// No special configuration options are required when working with the ` + "`ForwardedEvents`" + ` channel. The events in this log retain the channel name of their origin (e.g. ` + "`winlog.channel: Security`" + `). And because the routing pipeline processes events based on the channel name no special config is necessary.

// ` + "```yaml" + `
// winlogbeat.event_logs:
// - name: ForwardedEvents
//   tags: [forwarded]
//   language: 0x0409 # en-US

// output.elasticsearch.pipeline: winlogbeat-%{[agent.version]}-routing
// ` + "```" + `

// ## Modules [_modules]
// `

func moduleDocs() error {
	searchPath := filepath.Join(mage.XPackBeatDir(moduleDocsGlob))

	// Find module docs.
	files, err := mage.FindFiles(searchPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("No modules found matching %v", searchPath)
	}

	// Extract module name from path and copy the file.
	var names []string
	for _, f := range files {
		matches := moduleNameRegex.FindStringSubmatch(filepath.ToSlash(f))
		if len(matches) != 2 {
			return fmt.Errorf("module path %v does not match regexp", f)
		}
		name := matches[1]
		names = append(names, name)
		modulesListTmpl += fmt.Sprintf("* [%s](/reference/winlogbeat/winlogbeat-module-%s.md)\n", strings.Title(name), name)

		// Copy to the docs dirs.
		dest := filepath.Join(mage.DocsDir(), "reference", "winlogbeat", fmt.Sprintf("winlogbeat-module-%s.md", name))
		if err = mage.Copy(f, mage.CreateDir(dest)); err != nil {
			return err
		}
	}

	fmt.Printf(">> update:moduleDocs: Collecting module documentation for %v.\n", strings.Join(names, ", "))
	return ioutil.WriteFile(filepath.Join(mage.DocsDir(), "reference", "winlogbeat", "winlogbeat-modules.md"), []byte(modulesListTmpl), 0o644)
}
