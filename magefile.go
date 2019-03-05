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

// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/dev-tools/mage"
)

var (
	// Beats is a list of Beats to collect dashboards from.
	Beats = []string{
		"heartbeat",
		"journalbeat",
		"packetbeat",
		"winlogbeat",
		"x-pack/auditbeat",
		"x-pack/filebeat",
		"x-pack/metricbeat",
		"x-pack/functionbeat",
	}
)

// PackageBeatDashboards packages the dashboards from all Beats into a zip
// file. The dashboards must be generated first.
func PackageBeatDashboards() error {
	version, err := mage.BeatQualifiedVersion()
	if err != nil {
		return err
	}

	spec := mage.PackageSpec{
		Name:     "beats-dashboards",
		Version:  version,
		Snapshot: mage.Snapshot,
		Files: map[string]mage.PackageFile{
			".build_hash.txt": mage.PackageFile{
				Content: "{{ commit }}\n",
			},
		},
		OutputFile: "build/distributions/dashboards/{{.Name}}-{{.Version}}{{if .Snapshot}}-SNAPSHOT{{end}}",
	}

	for _, beatDir := range Beats {
		// The generated dashboard content is moving in the build dir, but
		// not all projects have been updated so detect which dir to use.
		dashboardDir := filepath.Join(beatDir, "build/kibana")
		legacyDir := filepath.Join(beatDir, "_meta/kibana.generated")
		beatName := filepath.Base(beatDir)

		if _, err := os.Stat(dashboardDir); err == nil {
			spec.Files[beatName] = mage.PackageFile{Source: dashboardDir}
		} else if _, err := os.Stat(legacyDir); err == nil {
			spec.Files[beatName] = mage.PackageFile{Source: legacyDir}
		} else {
			return errors.Errorf("no dashboards found for %v", beatDir)
		}
	}

	return mage.PackageZip(spec.Evaluate())
}

// Fmt formats code and adds license headers.
func Fmt() {
	mg.Deps(mage.GoImports, mage.PythonAutopep8)
	mg.Deps(addLicenseHeaders)
}

// addLicenseHeaders adds ASL2 headers to .go files outside of x-pack and
// add Elastic headers to .go files in x-pack.
func addLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Adding missing headers")

	if err := sh.Run("go", "get", mage.GoLicenserImportPath); err != nil {
		return err
	}

	return multierr.Combine(
		sh.RunV("go-licenser", "-license", "ASL2", "-exclude", "x-pack"),
		sh.RunV("go-licenser", "-license", "Elastic", "x-pack"),
	)
}

// DumpVariables writes the template variables and values to stdout.
func DumpVariables() error {
	return mage.DumpVariables()
}
