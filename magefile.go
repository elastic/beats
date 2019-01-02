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
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/pkg/errors"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"go.uber.org/multierr"

	"github.com/elastic/beats/dev-tools/mage"
)

var (
	projects = projectList{
		{"libbeat", build | fields | docs | unitTest | integTest | linuxCI | macosCI},
		{"auditbeat", build | fields | update | docs | packaging | unitTest | integTest | linuxCI | macosCI},
		{"filebeat", build | fields | update | docs | packaging | unitTest | integTest | linuxCI | macosCI},
		{"heartbeat", build | fields | update | docs | packaging | dashboards | unitTest | integTest | linuxCI | macosCI},
		{"journalbeat", build | fields | update | docs | packaging | dashboards | integTest | linuxCI},
		{"metricbeat", build | fields | update | docs | packaging | dashboards | unitTest | integTest | linuxCI | macosCI},
		{"packetbeat", build | fields | update | docs | packaging | dashboards | unitTest | linuxCI | macosCI},
		{"winlogbeat", build | fields | update | docs | packaging | dashboards | unitTest | linuxCI},
		{"x-pack/libbeat", build | unitTest | linuxCI},
		{"x-pack/auditbeat", build | fields | update | packaging | dashboards | unitTest | integTest | linuxCI | macosCI},
		{"x-pack/filebeat", build | fields | update | packaging | dashboards | unitTest | integTest | linuxCI | macosCI},
		{"x-pack/functionbeat", build | fields | update | packaging | dashboards | unitTest | integTest | linuxCI},
		{"x-pack/heartbeat", build | fields | update | packaging | linuxCI},
		{"x-pack/journalbeat", build | fields | update | packaging | linuxCI},
		{"x-pack/metricbeat", build | fields | update | packaging | update | linuxCI},
		{"x-pack/packetbeat", build | fields | update | packaging | linuxCI},
		{"x-pack/winlogbeat", build | fields | update | packaging | linuxCI},
		{"dev-tools/packaging/preference-pane", build | macosCI},
		{"deploy/kubernetes", update},
		{"docs", docs},

		// TODO: Add generators.
	}

	Aliases = map[string]interface{}{
		"check":   Check.All,
		"fmt":     Check.Fmt,
		"package": Package.All,
		"test":    Test.All,
		"update":  Update.All,
		"vet":     Check.Vet,
	}
)

type project struct {
	Dir   string
	Attrs attribute
}

func (p project) HasAttribute(a attribute) bool {
	return p.Attrs&a > 0
}

type attribute uint16

const (
	none  attribute = 0
	build attribute = 1 << iota
	update
	dashboards
	docs
	fields
	packaging
	unitTest
	integTest

	linuxCI
	macosCI

	any attribute = math.MaxUint16
)

type projectList []project

func (l projectList) ForEach(attr attribute, f func(proj project) error) error {
	for _, proj := range l {
		if proj.Attrs&attr > 0 {
			if err := f(proj); err != nil {
				return err
			}
		}
	}
	return nil
}

// --- Targets ---

func Clean() error {
	paths := []string{
		"build",
		"docs/build",
		"generator/beat/build",
		"generator/metricbeat/build",
	}

	_ = projects.ForEach(any, func(proj project) error {
		if strings.HasSuffix(filepath.Base(proj.Dir), "beat") {
			beatName := filepath.Base(proj.Dir)
			for _, path := range mage.DefaultCleanPaths {
				path = mage.MustExpand(path, map[string]interface{}{
					"BeatName": beatName,
				})
				paths = append(paths, filepath.Join(proj.Dir, path))
			}
		}
		return nil
	})

	return mage.Clean(paths)
}

type Check mg.Namespace

// Check checks that code is formatted and generated files are up-to-date.
func (Check) All() {
	mg.SerialDeps(Check.Fmt, Check.Targets, Update.All, mage.Check)
}

// Fmt formats code and adds license headers.
func (Check) Fmt() {
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

func (Check) Vet() error {
	return mage.GoVet()
}

var commonBeatTargets = []string{
	"check",
	"clean",
	"dumpVariables",
	"fmt",
	"build",
	"buildGoDaemon",
	"crossBuild",
	"crossBuildGoDaemon",
	"crossBuildGoDaemon",
	"golangCrossBuild",
	"update:fields",
}

func (Check) Targets() error {
	mageCmd := sh.OutCmd("mage", "-d")
	var errs []error
	err := projects.ForEach(any, func(proj project) error {
		fmt.Println("> check:targets:", proj.Dir)
		out, err := mageCmd(proj.Dir, "-l")
		if err != nil {
			return errors.Wrapf(err, "failed checking mage targets of project %v", proj.Dir)
		}
		targets, err := parseTargets(out)
		if err != nil {
			return errors.Wrapf(err, "failed parsing mage -l output of project %v", proj.Dir)
		}

		var expectedTargets []string
		if strings.HasSuffix(proj.Dir, "beat") {
			// Build list of expected targets based on attributes.
			expectedTargets = make([]string, len(commonBeatTargets))
			copy(expectedTargets, commonBeatTargets)
		}
		if proj.HasAttribute(build) {
			expectedTargets = append(expectedTargets, "build")
		}
		if proj.HasAttribute(fields) {
			expectedTargets = append(expectedTargets, "update:fields")
		}
		if proj.HasAttribute(update) {
			expectedTargets = append(expectedTargets, "update")
		}
		if proj.HasAttribute(dashboards) {
			expectedTargets = append(expectedTargets, "update:dashboards", "dashboards:import", "dashboards:export")
		}
		if proj.HasAttribute(docs) {
			expectedTargets = append(expectedTargets, "docs")
		}
		if proj.HasAttribute(packaging) {
			expectedTargets = append(expectedTargets, "package", "packageTest")
		}
		if proj.HasAttribute(unitTest) {
			expectedTargets = append(expectedTargets, "unitTest")
		}
		if proj.HasAttribute(integTest) {
			expectedTargets = append(expectedTargets, "integTest")
		}

		// Check for missing targets.
		var missing []string
		for _, target := range expectedTargets {
			if _, found := targets[target]; !found {
				missing = append(missing, target)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			err = errors.Errorf("failed checking mage targets of project "+
				"%v: missing [%v]", proj.Dir, strings.Join(missing, ", "))
			errs = append(errs, err)
		}
		// Check for missing descriptions.
		var badDescription []string
		for target, desc := range targets {
			desc := strings.TrimSpace(desc)
			if desc == "" || !strings.HasSuffix(desc, ".") {
				badDescription = append(badDescription, target)
			}
		}
		if len(badDescription) > 0 {
			sort.Strings(badDescription)
			err = errors.Errorf("failed checking mage targets of project "+
				"%v: no descriptions or missing period for [%v]", proj.Dir, strings.Join(badDescription, ", "))
			errs = append(errs, err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return multierr.Combine(errs...)
}

func parseTargets(rawOutput string) (map[string]string, error) {
	targets := map[string]string{}
	s := bufio.NewScanner(bytes.NewBufferString(rawOutput))
	for s.Scan() {
		line := s.Text()
		if line == "Targets:" || strings.HasPrefix(line, "*") {
			continue
		}
		if parts := strings.Fields(line); len(parts) > 0 {
			targets[parts[0]] = strings.Join(parts[1:], " ")
		}
	}
	return targets, s.Err()
}

func Docs() error {
	return projects.ForEach(docs, func(proj project) error {
		fmt.Println("> docs:", proj.Dir)
		return mage.Mage(proj.Dir, "docs")
	})
}

// DumpVariables writes the template variables and values to stdout.
func DumpVariables() error {
	return mage.DumpVariables()
}

type Update mg.Namespace

// All updates all Beats.
func (Update) All() error {
	mg.Deps(Update.Notice, Update.TravisCI)
	return projects.ForEach(update, func(proj project) error {
		fmt.Println("> update:all:", proj.Dir)
		return errors.Wrapf(mage.Mage(proj.Dir, "update"), "failed updating project %v", proj.Dir)
	})
}

// Fields updates the fields for each Beat.
func (Update) Fields() error {
	return projects.ForEach(fields, func(proj project) error {
		fmt.Println("> update:fields:", proj.Dir)
		return errors.Wrapf(mage.Mage(proj.Dir, "fields"), "failed updating project %v", proj.Dir)
	})
}

// Dashboards updates the dashboards for each Beat.
func (Update) Dashboards() error {
	return projects.ForEach(dashboards, func(proj project) error {
		fmt.Println("> update:dashboards:", proj.Dir)
		return errors.Wrapf(mage.Mage(proj.Dir, "update:dashboards"), "failed updating project %v", proj.Dir)
	})
}

func (Update) Notice() error {
	ve, err := mage.PythonVirtualenv()
	if err != nil {
		return err
	}
	pythonPath, err := mage.LookVirtualenvPath(ve, "python")
	if err != nil {
		return err
	}
	return sh.RunV(pythonPath, filepath.Clean("dev-tools/generate_notice.py"), ".")
}

func (Update) TravisCI() error {
	var data TravisCITemplateData

	// Check
	data.Jobs = append(data.Jobs, TravisCIJob{
		OS:    "linux",
		Stage: "check",
		Env: []string{
			"BUILD_CMD=" + strconv.Quote("mage"),
			"TARGETS=" + strconv.Quote("check"),
		},
	})

	_ = projects.ForEach(any, func(proj project) error {
		if proj.HasAttribute(linuxCI) && (proj.HasAttribute(unitTest) || proj.HasAttribute(integTest)) {
			var targets []string
			if proj.HasAttribute(unitTest) {
				targets = append(targets, "unitTest")
			}
			if proj.HasAttribute(integTest) {
				targets = append(targets, "integTest")
			}
			data.Jobs = append(data.Jobs, TravisCIJob{
				OS:    "linux",
				Stage: "test",
				Env: []string{
					"BUILD_CMD=" + strconv.Quote("mage -d "+filepath.ToSlash(proj.Dir)),
					"TARGETS=" + strconv.Quote(strings.Join(targets, " ")),
				},
			})
		}

		// We don't run the integTest on OSX because they require Docker.
		if proj.HasAttribute(macosCI) && proj.HasAttribute(unitTest) {
			data.Jobs = append(data.Jobs, TravisCIJob{
				OS:    "osx",
				Stage: "test",
				Env: []string{
					"BUILD_CMD=" + strconv.Quote("mage -d "+filepath.ToSlash(proj.Dir)),
					"TARGETS=" + strconv.Quote("unitTest"),
				},
			})
		}
		return nil
	})

	// Docs
	data.Jobs = append(data.Jobs, TravisCIJob{
		OS:    "linux",
		Stage: "test",
		Env: []string{
			"BUILD_CMD=" + strconv.Quote("mage"),
			"TARGETS=" + strconv.Quote("docs"),
		},
	})

	_ = projects.ForEach(any, func(proj project) error {
		if !strings.HasSuffix(filepath.Base(proj.Dir), "beat") {
			return nil
		}

		data.Jobs = append(data.Jobs, TravisCIJob{
			OS:    "linux",
			Stage: "crosscompile",
			Env: []string{
				"BUILD_CMD=" + strconv.Quote("make -C "+proj.Dir),
				"TARGETS=" + strconv.Quote("gox"),
			},
		})
		return nil
	})

	elasticBeats, err := mage.ElasticBeatsDir()
	if err != nil {
		return err
	}

	t, err := template.ParseFiles(filepath.Join(elasticBeats, "dev-tools/ci/templates/travis.yml.tmpl"))
	if err != nil {
		return err
	}

	out, err := os.OpenFile(".travis.yml", os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	return t.Execute(out, data)
}

type TravisCITemplateData struct {
	Jobs []TravisCIJob
}

type TravisCIJob struct {
	OS    string
	Env   []string
	Stage string
}

type Package mg.Namespace

// All packages all Beats and generates the dashboards zip package.
func (Package) All() {
	mg.SerialDeps(Package.Dashboards, Package.Beats)
}

// Dashboards packages the dashboards from all Beats into a zip file.
func (Package) Dashboards() error {
	mg.Deps(Update.Dashboards)

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

	_ = projects.ForEach(dashboards, func(proj project) error {
		beat := filepath.Base(proj.Dir)
		spec.Files[beat] = mage.PackageFile{
			Source: filepath.Join(proj.Dir, "build/kibana"),
		}
		return nil
	})

	return mage.PackageZip(spec.Evaluate())
}

// Beats packages each Beat.
//
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func (Package) Beats() (err error) {
	return projects.ForEach(packaging, func(proj project) error {
		fmt.Println("> package:beats:", proj.Dir)
		if err := mage.Mage(proj.Dir, "package"); err != nil {
			return errors.Wrapf(err, "failed packaging project %v", proj.Dir)
		}

		// Copy files to build/distributions.
		const distDir = "build/distributions"
		if err = os.MkdirAll(distDir, 0755); err != nil {
			return err
		}
		files, err := mage.FindFiles(filepath.Join(proj.Dir, distDir, "*"))
		if err != nil {
			return err
		}
		for _, f := range files {
			if err = os.Rename(f, filepath.Join(distDir, filepath.Base(f))); err != nil {
				return errors.Wrap(err, "failed moving packages to top-level build dir")
			}
		}
		return nil
	})
}

type Test mg.Namespace

func (Test) All() error {
	start := time.Now()
	defer func() { fmt.Println("test:all ran for", time.Since(start)) }()

	return projects.ForEach(any, func(proj project) error {
		fmt.Println("> test:all:", proj.Dir)
		if !proj.HasAttribute(unitTest) && !proj.HasAttribute(integTest) {
			return nil
		}
		return errors.Wrapf(mage.Mage(proj.Dir, "test"), "failed testing project %v", proj.Dir)
	})
}

func (Test) Unit() error {
	start := time.Now()
	defer func() { fmt.Println("test:unit ran for", time.Since(start)) }()

	return projects.ForEach(unitTest, func(proj project) error {
		fmt.Println("> test:unit:", proj.Dir)
		return errors.Wrapf(mage.Mage(proj.Dir, "unitTest"), "failed testing project %v", proj.Dir)
	})
}

func (Test) Integ() error {
	start := time.Now()
	defer func() { fmt.Println("test:integ ran for", time.Since(start)) }()

	return projects.ForEach(integTest, func(proj project) error {
		fmt.Println("> test:integ:", proj.Dir)
		return errors.Wrapf(mage.Mage(proj.Dir, "integTest"), "failed testing project %v", proj.Dir)
	})
}
