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
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"

	"github.com/elastic/beats/dev-tools/mage"
)

func init() {
	mage.BeatDescription = "Audit the activities of users and processes on your system."
}

// Build builds the Beat binary.
func Build() error {
	return mage.Build(mage.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	return mage.GolangCrossBuild(mage.DefaultGolangCrossBuildArgs())
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return mage.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return mage.CrossBuild()
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return mage.CrossBuildGoDaemon()
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	return mage.Clean()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	mage.UseElasticBeatPackaging()
	customizePackaging()

	mg.Deps(Update)
	mg.Deps(makeConfigTemplates, CrossBuild, CrossBuildGoDaemon)
	mg.SerialDeps(mage.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return mage.TestPackages()
}

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return mage.GenerateFieldsYAML("module")
}

// GoTestUnit executes the Go unit tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestUnit(ctx context.Context) error {
	return mage.GoTest(ctx, mage.DefaultGoTestUnitArgs())
}

// GoTestIntegration executes the Go integration tests.
// Use TEST_COVERAGE=true to enable code coverage profiling.
// Use RACE_DETECTOR=true to enable the race detector.
func GoTestIntegration(ctx context.Context) error {
	return mage.GoTest(ctx, mage.DefaultGoTestIntegrationArgs())
}

// -----------------------------------------------------------------------------
// Customizations specific to Auditbeat.
// - Config files are Go templates.

const (
	configTemplateGlob      = "module/*/_meta/config*.yml.tmpl"
	shortConfigTemplate     = "build/auditbeat.yml.tmpl"
	referenceConfigTemplate = "build/auditbeat.reference.yml.tmpl"
)

func makeConfigTemplates() error {
	configFiles, err := mage.FindFiles(configTemplateGlob)
	if err != nil {
		return errors.Wrap(err, "failed to find config templates")
	}

	var shortIn []string
	shortIn = append(shortIn, "_meta/common.p1.yml")
	shortIn = append(shortIn, configFiles...)
	shortIn = append(shortIn, "_meta/common.p2.yml")
	shortIn = append(shortIn, "../libbeat/_meta/config.yml")
	if !mage.IsUpToDate(shortConfigTemplate, shortIn...) {
		fmt.Println(">> Building", shortConfigTemplate)
		mage.MustFileConcat(shortConfigTemplate, 0600, shortIn...)
		mage.MustFindReplace(shortConfigTemplate, regexp.MustCompile("beatname"), "{{.BeatName}}")
		mage.MustFindReplace(shortConfigTemplate, regexp.MustCompile("beat-index-prefix"), "{{.BeatIndexPrefix}}")
	}

	var referenceIn []string
	referenceIn = append(referenceIn, "_meta/common.reference.yml")
	referenceIn = append(referenceIn, configFiles...)
	referenceIn = append(referenceIn, "../libbeat/_meta/config.reference.yml")
	if !mage.IsUpToDate(referenceConfigTemplate, referenceIn...) {
		fmt.Println(">> Building", referenceConfigTemplate)
		mage.MustFileConcat(referenceConfigTemplate, 0644, referenceIn...)
		mage.MustFindReplace(referenceConfigTemplate, regexp.MustCompile("beatname"), "{{.BeatName}}")
		mage.MustFindReplace(referenceConfigTemplate, regexp.MustCompile("beat-index-prefix"), "{{.BeatIndexPrefix}}")
	}

	return nil
}

// customizePackaging modifies the package specs to use templated config files
// instead of the defaults.
//
// Customizations specific to Auditbeat:
// - Include audit.rules.d directory in packages.
func customizePackaging() {
	var (
		shortConfig = mage.PackageFile{
			Mode:   0600,
			Source: "{{.PackageDir}}/auditbeat.yml",
			Dep:    generateShortConfig,
		}
		referenceConfig = mage.PackageFile{
			Mode:   0644,
			Source: "{{.PackageDir}}/auditbeat.reference.yml",
			Dep:    generateReferenceConfig,
		}
	)

	archiveRulesDir := "audit.rules.d"
	linuxPkgRulesDir := "/etc/{{.BeatName}}/audit.rules.d"
	rulesSrcDir := "module/auditd/_meta/audit.rules.d"
	sampleRules := mage.PackageFile{
		Mode:   0644,
		Source: rulesSrcDir,
		Dep: func(spec mage.PackageSpec) error {
			if spec.OS == "linux" {
				params := map[string]interface{}{
					"ArchBits": archBits,
				}
				rulesFile := spec.MustExpand(rulesSrcDir+"/sample-rules-linux-{{call .ArchBits .GOARCH}}bit.conf", params)
				if err := mage.Copy(rulesFile, spec.MustExpand("{{.PackageDir}}/audit.rules.d/sample-rules.conf.disabled")); err != nil {
					return errors.Wrap(err, "failed to copy sample rules")
				}
			}
			return nil
		},
	}

	for _, args := range mage.Packages {
		pkgType := args.Types[0]
		switch pkgType {
		case mage.TarGz, mage.Zip:
			args.Spec.ReplaceFile("{{.BeatName}}.yml", shortConfig)
			args.Spec.ReplaceFile("{{.BeatName}}.reference.yml", referenceConfig)
		case mage.Deb, mage.RPM, mage.DMG:
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.yml", shortConfig)
			args.Spec.ReplaceFile("/etc/{{.BeatName}}/{{.BeatName}}.reference.yml", referenceConfig)
		default:
			panic(errors.Errorf("unhandled package type: %v", pkgType))
		}
		if args.OS == "linux" {
			rulesDest := archiveRulesDir
			if pkgType != mage.TarGz {
				rulesDest = linuxPkgRulesDir
			}
			args.Spec.Files[rulesDest] = sampleRules
		}
	}
}

func generateReferenceConfig(spec mage.PackageSpec) error {
	params := map[string]interface{}{
		"Reference": true,
		"ArchBits":  archBits,
	}
	return spec.ExpandFile(referenceConfigTemplate,
		"{{.PackageDir}}/auditbeat.reference.yml", params)
}

func generateShortConfig(spec mage.PackageSpec) error {
	params := map[string]interface{}{
		"Reference": false,
		"ArchBits":  archBits,
	}
	return spec.ExpandFile(shortConfigTemplate,
		"{{.PackageDir}}/auditbeat.yml", params)
}

// archBits returns the number of bit width of the GOARCH architecture value.
// This function is used by the auditd module configuration templates to
// generate architecture specific audit rules.
func archBits(goarch string) int {
	switch goarch {
	case "386", "arm":
		return 32
	default:
		return 64
	}
}

// Configs generates the auditbeat.yml and auditbeat.reference.yml config files.
// Set DEV_OS and DEV_ARCH to change the target host for the generated configs.
// Defaults to linux/amd64.
func Configs() {
	mg.Deps(makeConfigTemplates)

	params := map[string]interface{}{
		"GOOS":      mage.EnvOr("DEV_OS", "linux"),
		"GOARCH":    mage.EnvOr("DEV_ARCH", "amd64"),
		"ArchBits":  archBits,
		"Reference": false,
	}
	fmt.Printf(">> Building auditbeat.yml for %v/%v\n", params["GOOS"], params["GOARCH"])
	mage.MustExpandFile(shortConfigTemplate, "auditbeat.yml", params)

	params["Reference"] = true
	fmt.Printf(">> Building auditbeat.reference.yml for %v/%v\n", params["GOOS"], params["GOARCH"])
	mage.MustExpandFile(referenceConfigTemplate, "auditbeat.reference.yml", params)
}
