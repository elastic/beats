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
	"log"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

// Package packages the Beat for distribution. It generates packages based on
// the set of target platforms and registered packaging specifications.
func Package() error {
	if len(Platforms) == 0 {
		fmt.Println(">> package: Skipping because the platform list is empty")
		return nil
	}

	if len(Packages) == 0 {
		return errors.New("no package specs are registered. Call " +
			"UseCommunityBeatPackaging, UseElasticBeatPackaging or USeElasticBeatWithoutXPackPackaging first.")
	}

	var tasks []interface{}
	for _, target := range Platforms {
		for _, pkg := range Packages {
			if pkg.OS != target.GOOS() {
				continue
			}

			for _, pkgType := range pkg.Types {
				if pkgType == DMG && runtime.GOOS != "darwin" {
					log.Printf("Skipping DMG package type because build host isn't darwin")
					continue
				}

				packageArch, err := getOSArchName(target, pkgType)
				if err != nil {
					log.Printf("Skipping arch %v for package type %v: %v", target.Arch(), pkgType, err)
					continue
				}

				spec := pkg.Spec.Clone()
				spec.OS = target.GOOS()
				spec.Arch = packageArch
				spec.Snapshot = Snapshot
				spec.evalContext = map[string]interface{}{
					"GOOS":        target.GOOS(),
					"GOARCH":      target.GOARCH(),
					"GOARM":       target.GOARM(),
					"Platform":    target,
					"PackageType": pkgType.String(),
					"BinaryExt":   binaryExtension(target.GOOS()),
				}
				spec.packageDir = packageStagingDir + "/" + pkgType.AddFileExtension(spec.Name+"-"+target.GOOS()+"-"+target.Arch())
				spec = spec.Evaluate()

				tasks = append(tasks, packageBuilder{target, spec, pkgType}.Build)
			}
		}
	}

	Parallel(tasks...)
	return nil
}

type packageBuilder struct {
	Platform BuildPlatform
	Spec     PackageSpec
	Type     PackageType
}

func (b packageBuilder) Build() error {
	fmt.Printf(">> package: Building %v type=%v for platform=%v\n", b.Spec.Name, b.Type, b.Platform.Name)
	log.Printf("Package spec: %+v", b.Spec)
	return errors.Wrapf(b.Type.Build(b.Spec), "failed building %v type=%v for platform=%v",
		b.Spec.Name, b.Type, b.Platform.Name)
}

type testPackagesParams struct {
	HasModules   bool
	HasMonitorsD bool
	HasModulesD  bool
}

// TestPackagesOption defines a option to the TestPackages target.
type TestPackagesOption func(params *testPackagesParams)

// WithModules enables modules folder contents testing
func WithModules() func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		params.HasModules = true
	}
}

// WithMonitorsD enables monitors folder contents testing.
func WithMonitorsD() func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		params.HasMonitorsD = true
	}
}

// WithModulesD enables modules.d folder contents testing
func WithModulesD() func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		params.HasModulesD = true
	}
}

// TestPackages executes the package tests on the produced binaries. These tests
// inspect things like file ownership and mode.
func TestPackages(options ...TestPackagesOption) error {
	params := testPackagesParams{}
	for _, opt := range options {
		opt(&params)
	}

	fmt.Println(">> Testing package contents")
	goTest := sh.OutCmd("go", "test")

	var args []string
	if mg.Verbose() {
		args = append(args, "-v")
	}

	args = append(args, MustExpand("{{ elastic_beats_dir }}/dev-tools/packaging/package_test.go"))

	if params.HasModules {
		args = append(args, "--modules")
	}

	if params.HasMonitorsD {
		args = append(args, "--monitors.d")
	}

	if params.HasModulesD {
		args = append(args, "--modules.d")
	}

	if BeatUser == "root" {
		args = append(args, "-root-owner")
	}
	args = append(args, "-files", MustExpand("{{.PWD}}/build/distributions/*"))

	if out, err := goTest(args...); err != nil {
		if !mg.Verbose() {
			fmt.Println(out)
		}
		return err
	}

	return nil
}
