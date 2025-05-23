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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.uber.org/multierr"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	//mage:import
	"github.com/elastic/elastic-agent-libs/dev-tools/mage"

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

	// Beats are all beats projects, including libbeat
	Beats = []string{
		"auditbeat",
		"filebeat",
		"heartbeat",
		"libbeat",
		"metricbeat",
		"packetbeat",
		"winlogbeat",
	}

	// XPack are all x-pack beats projects, including libbeat
	XPack = []string{
		"agentbeat",
		"auditbeat",
		"dockerlogbeat",
		"filebeat",
		"heartbeat",
		"libbeat",
		"metricbeat",
		"osquerybeat",
		"packetbeat",
		"winlogbeat",
	}
)

// Aliases are shortcuts to long target names.
// nolint: deadcode // it's used by `mage`.
var Aliases = map[string]interface{}{
	"llc":  mage.Linter.LastChange,
	"lint": mage.Linter.All,
}

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

// UnitTest runs unit tests for all OSS and x-pack beats projects
// (Go and Python).
func UnitTest() error {
	fmt.Println(">> running UnitTest for all beats")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not ger current working directory: %v", err)
	}
	defer func() {
		err = os.Chdir(wd)
		if err != nil {
			err = fmt.Errorf("could not restore work directory: %w", err)
		}
	}()

	var beats []string
	for _, d := range Beats {
		beats = append(beats, filepath.Join(wd, d))
	}
	for _, d := range XPack {
		if d == "agentbeat" {
			fmt.Println(">> skipping x-pack/agentbeat")
			continue
		}
		beats = append(beats, filepath.Join(wd, "x-pack", d))
	}

	return runOnEveryBeat(beats, "unitTest")
}

// IntegTest runs integration tests for all OSS and x-pack beats projects (it
// uses Docker to run the tests).
func IntegTest() error {
	fmt.Println(">> running IntegTest for all beats")

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("could not ger current working directory: %v", err)
	}
	defer func() {
		err = os.Chdir(wd)
		if err != nil {
			err = fmt.Errorf("could not restore work directory: %w", err)
		}
	}()

	var beats []string

	for _, d := range Beats {
		if d == "winlogbeat" {
			fmt.Println(">> skipping winlogbeat")
			continue
		}
		beats = append(beats, filepath.Join(wd, d))
	}

	xpackSkip := []string{"packetbeat", "winlogbeat"}
	for _, d := range XPack {
		if slices.Contains(xpackSkip, d) {
			fmt.Printf(">> skipping x-pack/%s\n", d)
			continue
		}
		beats = append(beats, filepath.Join(wd, "x-pack", d))
	}

	return runOnEveryBeat(beats, "integTest")
}

func runOnEveryBeat(beatDirs []string, mageTarget string) error {
	failedBeats := map[string]error{}

	fmt.Println("")
	for _, p := range beatDirs {
		fmt.Println(">> entering", p)
		err := os.Chdir(p)
		if err != nil {
			return fmt.Errorf("could not change to %q: %v", p, err)
		}

		err = sh.RunV("mage", mageTarget)
		if err != nil {
			failedBeats[p] = err
		}
		fmt.Println("")
	}

	if len(failedBeats) > 0 {
		fmt.Print("\n\n")
		fmt.Println(">> some tests failed:\n")
		var errs error

		for p, err := range failedBeats {
			fmt.Printf("%s failed: %v\n", p, err)
			errs = errors.Join(errs, err)
		}

		fmt.Println("")
		return errs
	}

	return nil
}
