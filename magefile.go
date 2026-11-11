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

//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"go.uber.org/multierr"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/gotool"
)

var (
	// BeatsWithDashboards is a list of Beats to collect dashboards from.
	BeatsWithDashboards = []string{
		"heartbeat",
		"packetbeat",
		"winlogbeat",
		"x-pack/auditbeat",
		"x-pack/filebeat",
		"x-pack/metricbeat",
	}
)

// Aliases are shortcuts to long target names.
// nolint: deadcode // it's used by `mage`.
var Aliases = map[string]interface{}{
	"llc":  Linter.LastChange,
	"lint": Linter.All,
}

// Linter contains targets related to linting the Go code. It delegates to
// devtools.Linter; the namespace is redeclared here because mage only exposes
// targets declared in (or //mage:import-ed into) the magefile package.
type Linter mg.Namespace

// All runs the linter against the entire codebase.
func (Linter) All() error { return devtools.Linter{}.All() }

// LastChange runs the linter against all files changed since the fork point from `main`.
func (Linter) LastChange() error { return devtools.Linter{}.LastChange() }

// Install installs golangci-lint to `./build` if it is not there yet.
func (Linter) Install() error { return devtools.Linter{}.Install() }

// ForceInstall installs golangci-lint regardless of whether it exists or not.
func (Linter) ForceInstall() error { return devtools.Linter{}.ForceInstall() }

// CheckConfig makes sure that the `.golangci.yml` does not have uncommitted changes.
func (Linter) CheckConfig() error { return devtools.Linter{}.CheckConfig() }

// Version prints the version of the linter in use.
func (Linter) Version() error { return devtools.Linter{}.Version() }

// PackageBeatDashboards packages the dashboards from all Beats into a zip
// file. The dashboards must be generated first.
func PackageBeatDashboards() error {
	version, err := devtools.BeatQualifiedVersion()
	if err != nil {
		return err
	}

	spec := devtools.PackageSpec{
		Name:     "beats-dashboards",
		Version:  version,
		Snapshot: devtools.Snapshot,
		Files: map[string]devtools.PackageFile{
			".build_hash.txt": devtools.PackageFile{
				Content: "{{ commit }}\n",
			},
		},
		OutputFile: "build/distributions/dashboards/{{.Name}}-{{.Version}}{{if .Snapshot}}-SNAPSHOT{{end}}",
	}

	for _, beatDir := range BeatsWithDashboards {
		// The generated dashboard content is moving in the build dir, but
		// not all projects have been updated so detect which dir to use.
		dashboardDir := filepath.Join(beatDir, "build/kibana")
		legacyDir := filepath.Join(beatDir, "_meta/kibana.generated")
		beatName := filepath.Base(beatDir)

		if _, err := os.Stat(dashboardDir); err == nil {
			spec.Files[beatName] = devtools.PackageFile{Source: dashboardDir}
		} else if _, err := os.Stat(legacyDir); err == nil {
			spec.Files[beatName] = devtools.PackageFile{Source: legacyDir}
		} else {
			return fmt.Errorf("no dashboards found for %v", beatDir)
		}
	}

	return devtools.PackageZip(spec.Evaluate())
}

// Fmt formats code and adds license headers.
func Fmt() {
	mg.Deps(devtools.GoImports, devtools.PythonAutopep8)
	mg.Deps(AddLicenseHeaders)
}

// AddLicenseHeaders adds ASL2 headers to .go files outside of x-pack and
// add Elastic headers to .go files in x-pack.
func AddLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Adding missing headers")

	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return multierr.Combine(
		licenser(
			licenser.License("ASL2"),
			licenser.Exclude("x-pack"),
			licenser.Exclude("generator/_templates/beat/{beat}"),
			licenser.Exclude("generator/_templates/metricbeat/{beat}"),
		),
		licenser(
			licenser.License("Elastic"),
			licenser.Path("x-pack"),
		),
	)
}

// CheckLicenseHeaders checks ASL2 headers in .go files outside of x-pack and
// checks Elastic headers in .go files in x-pack.
func CheckLicenseHeaders() error {
	fmt.Println(">> fmt - go-licenser: Checking for missing headers")

	mg.Deps(devtools.InstallGoLicenser)

	licenser := gotool.Licenser

	return multierr.Combine(
		licenser(
			licenser.Check(),
			licenser.License("ASL2"),
			licenser.Exclude("x-pack"),
			licenser.Exclude("generator/_templates/beat/{beat}"),
			licenser.Exclude("generator/_templates/metricbeat/{beat}"),
		),
		licenser(
			licenser.Check(),
			licenser.License("Elastic"),
			licenser.Path("x-pack"),
		),
	)
}

// DumpVariables writes the template variables and values to stdout.
func DumpVariables() error {
	return devtools.DumpVariables()
}
