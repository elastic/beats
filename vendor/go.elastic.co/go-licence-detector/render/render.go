// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package render // import "go.elastic.co/go-licence-detector/render"

import (
	"bytes"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"go.elastic.co/go-licence-detector/dependency"
	"golang.org/x/mod/semver"
)

type extraLicenceTextFunc func(dependency.Info) string

var extraTextByLicence = map[string]extraLicenceTextFunc{
	"EPL-1.0": func(depInfo dependency.Info) string {
		headerStr := `Pursuant to Section 7.1 of EPL v1.0, this library is being distributed under EPL v2.0,
which is available at https://www.eclipse.org/legal/epl-2.0/.
The source code is available at %s with link to the repo for the source code.

`
		return fmt.Sprintf(headerStr, depInfo.URL)
	},
}

var goModCache = filepath.Join(build.Default.GOPATH, "pkg", "mod")

func Template(dependencies *dependency.List, templatePath, outputPath string) error {
	funcMap := template.FuncMap{
		"currentYear":      CurrentYear,
		"line":             Line,
		"licenceText":      LicenceText,
		"revision":         Revision,
		"canonicalVersion": CanonicalVersion,
	}
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(funcMap).ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("failed to parse template at %s: %w", templatePath, err)
	}

	w, cleanup, err := mkWriter(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %w", outputPath, err)
	}
	defer cleanup()

	if err := tmpl.Execute(w, dependencies); err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	return nil
}

func mkWriter(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}

	f, err := os.Create(path)
	return f, func() { f.Close() }, err
}

/* Template functions */

var regexCanonical = regexp.MustCompile(`^(?P<version>v[0-9.]+)`)
var regexRevision = regexp.MustCompile(`[^-]+-[^-]+-(?P<revision>[a-fA-Z0-9]+)`)

// Canonical ensures that the version string contains a minor and patch part
// and discards any additional metadata from the version string.
//
// For example:
//   v1    => v1.0.0
//   v1.2  => v1.2.0
//   v1.2.3+incompatible => v1.2.3
//   v1.2.3-20200707-123456abc => v1.2.3
func CanonicalVersion(in string) string {
	matches := regexCanonical.FindStringSubmatch(in)
	version := regexGroup(regexCanonical, "version", matches)
	return semver.Canonical(version)
}

// Revision returns the hash from version strings following this format: <version>-<timestamp>-<hash>
// If the string does not match this pattern an empty string is returned.
//
// For example:
//   v1.2.3  =>
//   v1.2.3-20200707-123456abc => 123456abc
func Revision(in string) string {
	matches := regexRevision.FindStringSubmatch(in)
	return regexGroup(regexRevision, "revision", matches)
}

func regexGroup(regex *regexp.Regexp, groupName string, matches []string) string {
	for i, name := range regex.SubexpNames() {
		if i > 0 && name == groupName && i < len(matches) {
			return matches[i]
		}
	}
	return ""
}

func CurrentYear() string {
	return strconv.Itoa(time.Now().Year())
}

func Line(ch string) string {
	return strings.Repeat(ch, 80)
}

func LicenceText(depInfo dependency.Info) string {
	if depInfo.LicenceFile == "" {
		return "No licence file provided."
	}

	var buf bytes.Buffer
	additonalLicenceText(&buf, depInfo)

	if depInfo.LicenceTextOverrideFile != "" {
		buf.WriteString("Contents of provided licence file")
	} else {
		buf.WriteString("Contents of probable licence file ")
		buf.WriteString(strings.Replace(depInfo.LicenceFile, goModCache, "$GOMODCACHE", -1))
	}
	buf.WriteString(":\n\n")

	f, err := os.Open(depInfo.LicenceFile)
	if err != nil {
		log.Fatalf("Failed to open licence file %s: %v", depInfo.LicenceFile, err)
	}
	defer f.Close()

	_, err = io.Copy(&buf, f)
	if err != nil {
		log.Fatalf("Failed to read licence file %s: %v", depInfo.LicenceFile, err)
	}

	return buf.String()
}

func additonalLicenceText(buf *bytes.Buffer, depInfo dependency.Info) {
	txtFunc, ok := extraTextByLicence[depInfo.LicenceType]
	if !ok {
		return
	}
	txt := txtFunc(depInfo)
	buf.WriteString(txt)
}
