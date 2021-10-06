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
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"

	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/common"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	common.RegisterCheckDeps(Update)

	devtools.BeatDescription = "Journalbeat ships systemd journal entries to Elasticsearch or Logstash."

	devtools.Platforms = devtools.Platforms.Filter("linux !linux/ppc64 !linux/mips64")
}

const (
	libsystemdDevPkgName = "libsystemd-dev"
	libsystemdPkgName    = "libsystemd0"
	libgcryptPkgName     = "libgcrypt20"
)

// Build builds the Beat binary.
func Build() error {
	return devtools.Build(devtools.DefaultBuildArgs())
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	mg.Deps(deps.Installer(devtools.Platform.Name))
	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
}

// BuildGoDaemon builds the go-daemon binary (use crossBuildGoDaemon).
func BuildGoDaemon() error {
	return devtools.BuildGoDaemon()
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	return devtools.CrossBuild(devtools.ImageSelector(selectImage))
}

// CrossBuildXPack cross-builds the beat with XPack for all target platforms.
func CrossBuildXPack() error {
	return devtools.CrossBuildXPack(devtools.ImageSelector(selectImage))
}

// CrossBuildGoDaemon cross-builds the go-daemon binary using Docker.
func CrossBuildGoDaemon() error {
	return devtools.CrossBuildGoDaemon(devtools.ImageSelector(selectImage))
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.UseElasticBeatPackaging()

	mg.Deps(Update)
	mg.Deps(CrossBuild, CrossBuildXPack, CrossBuildGoDaemon)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update updates the generated files (aka make update).
func Update() error {
	return sh.Run("make", "update")
}

// Fields generates a fields.yml for the Beat.
func Fields() error {
	return devtools.GenerateFieldsYAML()
}

// -----------------------------------------------------------------------------
// Customizations specific to Journalbeat.
// - Install required headers on builders for different architectures.

var (
	journaldPlatforms = []devtools.PlatformDescription{
		devtools.Linux386, devtools.LinuxAMD64,
		devtools.LinuxARM64, devtools.LinuxARM5, devtools.LinuxARM6, devtools.LinuxARM7,
		devtools.LinuxMIPS, devtools.LinuxMIPSLE, devtools.LinuxMIPS64LE,
		devtools.LinuxPPC64LE,
		devtools.LinuxS390x,
	}

	deps = devtools.NewPackageInstaller().
		AddEach(journaldPlatforms, libsystemdDevPkgName).
		Add(devtools.Linux386, libsystemdPkgName, libgcryptPkgName)
)

func selectImage(platform string) (string, error) {
	tagSuffix := "main"

	switch {
	case strings.HasPrefix(platform, "linux/armv7"):
		tagSuffix = "armhf"
	case strings.HasPrefix(platform, "linux/arm"):
		tagSuffix = "arm"
		if runtime.GOARCH == "arm64" {
			tagSuffix = "base-arm-debian9"
		}
	case strings.HasPrefix(platform, "linux/mips"):
		tagSuffix = "mips"
	case strings.HasPrefix(platform, "linux/ppc"):
		tagSuffix = "ppc"
	case platform == "linux/s390x":
		tagSuffix = "s390x"
	case strings.HasPrefix(platform, "linux"):
		tagSuffix = "main-debian8"
	}

	goVersion, err := devtools.GoVersion()
	if err != nil {
		return "", err
	}

	return devtools.BeatsCrossBuildImage + ":" + goVersion + "-" + tagSuffix, nil
}

// Config generates both the short/reference/docker configs.
func Config() error {
	p := devtools.DefaultConfigFileParams()
	p.Templates = append(p.Templates, devtools.OSSBeatDir("_meta/config/*.tmpl"))
	return devtools.Config(devtools.AllConfigTypes, p, ".")
}
