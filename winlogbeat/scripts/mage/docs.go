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
	"path/filepath"
	"regexp"
	"strings"

	"github.com/elastic/beats/v7/dev-tools/mage"
)

const moduleDocsGlob = "module/*/_meta/docs.md"

var moduleNameRegex = regexp.MustCompile(`module\/(.*)\/_meta\/docs.md`)

//go:embed templates/moduleList.tmpl
var modulesListTmpl string

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

	// TODO(@VihasMakwana): Uncomment following when all the asciidocs are converted to markdown
	// As of now, this will not work and it will generate incomplete list.

	// fmt.Printf(">> update:moduleDocs: Collecting module documentation for %v.\n", strings.Join(names, ", "))
	// return ioutil.WriteFile(filepath.Join(mage.DocsDir(), "reference", "winlogbeat", "winlogbeat-modules.md"), []byte(modulesListTmpl), 0o644)

	return nil
}
