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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

const moduleDocsGlob = "module/*/_meta/docs.asciidoc"

var moduleNameRegex = regexp.MustCompile(`module\/(.*)\/_meta\/docs.asciidoc`)

var modulesListTmpl = `
////
This file is generated! See scripts/mage/docs.go or run 'mage docs'.
////
{{range $module := .Modules}}
  * <<{beatname_lc}-module-{{$module}},{{$module | title}}>>
{{- end}}

--

{{range $module := .Modules}}
include::./modules/{{$module}}.asciidoc[]
{{- end}}
`[1:]

func moduleDocs() error {
	searchPath := filepath.Join(mage.XPackBeatDir(moduleDocsGlob))

	// Find module docs.
	files, err := mage.FindFiles(searchPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.Errorf("No modules found matching %v", searchPath)
	}

	// Clean existing files.
	if err := os.RemoveAll(mage.OSSBeatDir("docs/modules")); err != nil {
		return err
	}

	// Extract module name from path and copy the file.
	var names []string
	for _, f := range files {
		matches := moduleNameRegex.FindStringSubmatch(filepath.ToSlash(f))
		if len(matches) != 2 {
			return errors.Errorf("module path %v does not match regexp", f)
		}
		name := matches[1]
		names = append(names, name)

		// Copy to the docs dirs.
		if err = mage.Copy(f, mage.CreateDir(mage.OSSBeatDir("docs/modules", name+".asciidoc"))); err != nil {
			return err
		}
	}

	// Generate and write the docs/modules_list.asciidoc file.
	content, err := mage.Expand(modulesListTmpl, map[string]interface{}{
		"Modules": names,
	})
	if err != nil {
		return err
	}

	fmt.Printf(">> update:moduleDocs: Collecting module documentation for %v.\n", strings.Join(names, ", "))
	return ioutil.WriteFile(mage.OSSBeatDir("docs/modules_list.asciidoc"), []byte(content), 0644)
}
