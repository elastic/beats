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
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

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
			if pkg.OS != target.GOOS() || pkg.Arch != "" && pkg.Arch != target.Arch() {
				continue
			}

			for _, pkgType := range pkg.Types {
				if !isPackageTypeSelected(pkgType) {
					log.Printf("Skipping %s package type because it is not selected", pkgType)
					continue
				}

				if pkgType == DMG && runtime.GOOS != "darwin" {
					log.Printf("Skipping DMG package type because build host isn't darwin")
					continue
				}

				if target.Name == "linux/arm64" && pkgType == Docker && runtime.GOARCH != "arm64" {
					log.Printf("Skipping Docker package type because build host isn't arm")
					continue
				}

				packageArch, err := getOSArchName(target, pkgType)
				if err != nil {
					log.Printf("Skipping arch %v for package type %v: %v", target.Arch(), pkgType, err)
					continue
				}

				agentPackageType := TarGz
				if pkg.OS == "windows" {
					agentPackageType = Zip
				}

				agentPackageArch, err := getOSArchName(target, agentPackageType)
				if err != nil {
					log.Printf("Skipping arch %v for package type %v: %v", target.Arch(), pkgType, err)
					continue
				}

				agentPackageDrop, _ := os.LookupEnv("AGENT_DROP_PATH")

				spec := pkg.Spec.Clone()
				spec.OS = target.GOOS()
				spec.Arch = packageArch
				spec.Snapshot = Snapshot
				spec.evalContext = map[string]interface{}{
					"GOOS":          target.GOOS(),
					"GOARCH":        target.GOARCH(),
					"GOARM":         target.GOARM(),
					"Platform":      target,
					"AgentArchName": agentPackageArch,
					"PackageType":   pkgType.String(),
					"BinaryExt":     binaryExtension(target.GOOS()),
					"AgentDropPath": agentPackageDrop,
				}

				spec.packageDir, err = pkgType.PackagingDir(packageStagingDir, target, spec)
				if err != nil {
					log.Printf("Skipping arch %v for package type %v: %v", target.Arch(), pkgType, err)
					continue
				}

				spec = spec.Evaluate()

				tasks = append(tasks, packageBuilder{target, spec, pkgType}.Build)
			}
		}
	}

	Parallel(tasks...)
	return nil
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	if runtime.GOARCH != "amd64" {
		fmt.Printf(">> IronBank images are only supported for amd64 arch (%s is not supported)\n", runtime.GOARCH)
		return nil
	}
	if err := prepareIronbankBuild(); err != nil {
		return fmt.Errorf("failed to prepare the IronBank context: %w", err)
	}
	if err := saveIronbank(); err != nil {
		return fmt.Errorf("failed to save the IronBank context: %w", err)
	}
	return nil
}

func getIronbankContextName() string {
	version, _ := BeatQualifiedVersion()
	ironbankBinaryName := "{{.Name}}-ironbank-{{.Version}}{{if .Snapshot}}-SNAPSHOT{{end}}-docker-build-context"
	// TODO: get the name of the project
	outputDir, _ := Expand(ironbankBinaryName, map[string]interface{}{
		"Name":    BeatName,
		"Version": version,
	})
	return outputDir
}

func prepareIronbankBuild() error {
	fmt.Println(">> prepareIronbankBuild: prepare the IronBank container context.")
	buildDir := filepath.Join("build", getIronbankContextName())
	beatsDir, err := ElasticBeatsDir()
	if err != nil {
		return fmt.Errorf("could not get the base dir: %w", err)
	}

	templatesDir := filepath.Join(beatsDir, "dev-tools", "packaging", "templates", "ironbank", BeatName)

	data := map[string]interface{}{
		"MajorMinor": BeatMajorMinorVersion(),
	}

	err = filepath.Walk(templatesDir, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			target := strings.TrimSuffix(
				filepath.Join(buildDir, filepath.Base(path)),
				".tmpl",
			)

			err := ExpandFile(path, target, data)
			if err != nil {
				return fmt.Errorf("expanding template '%s' to '%s': %w", path, target, err)
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("cannot create templates for the IronBank: %w", err)
	}

	// copy license
	sourceLicense := filepath.Join(beatsDir, "dev-tools", "packaging", "files", "ironbank", "LICENSE")
	targetLicense := filepath.Join(buildDir, "LICENSE")
	if err := CopyFile(sourceLicense, targetLicense); err != nil {
		return fmt.Errorf("cannot copy LICENSE file for the IronBank: %w", err)
	}

	// copy specific files for the given beat
	sourceBeatPath := filepath.Join(beatsDir, "dev-tools", "packaging", "files", "ironbank", BeatName)
	if _, err := os.Stat(sourceBeatPath); !os.IsNotExist(err) {
		if err := Copy(sourceBeatPath, buildDir); err != nil {
			return fmt.Errorf("cannot create files for the IronBank: %w", err)
		}
	}

	return nil
}

func saveIronbank() error {
	fmt.Println(">> saveIronbank: save the IronBank container context.")

	ironbank := getIronbankContextName()
	buildDir := filepath.Join("build", ironbank)
	if _, err := os.Stat(buildDir); os.IsNotExist(err) {
		return fmt.Errorf("cannot find the folder with the ironbank context: %+v", err)
	}

	distributionsDir := "build/distributions"
	if _, err := os.Stat(distributionsDir); os.IsNotExist(err) {
		err := os.MkdirAll(distributionsDir, 0750)
		if err != nil {
			return fmt.Errorf("cannot create folder for docker artifacts: %+v", err)
		}
	}
	tarGzFile := filepath.Join(distributionsDir, ironbank+".tar.gz")

	// Save the build context as tar.gz artifact
	err := TarWithOptions(buildDir, tarGzFile, true)
	if err != nil {
		return fmt.Errorf("cannot compress the tar.gz file: %+v", err)
	}

	return errors.Wrap(CreateSHA512File(tarGzFile), "failed to create .sha512 file")
}

// isPackageTypeSelected returns true if SelectedPackageTypes is empty or if
// pkgType is present on SelectedPackageTypes. It returns false otherwise.
func isPackageTypeSelected(pkgType PackageType) bool {
	if SelectedPackageTypes != nil {
		selected := false
		for _, t := range SelectedPackageTypes {
			if t == pkgType {
				selected = true
			}
		}
		return selected
	}
	return true
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
	HasModules           bool
	HasMonitorsD         bool
	HasModulesD          bool
	HasRootUserContainer bool
	MinModules           *int
}

// TestPackagesOption defines a option to the TestPackages target.
type TestPackagesOption func(params *testPackagesParams)

// WithModules enables modules folder contents testing
func WithModules() func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		params.HasModules = true
	}
}

// MinModules sets the minimum number of modules to require
func MinModules(n int) func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		minModules := n
		params.MinModules = &minModules
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

// WithRootUserContainer allows root when checking user in container
func WithRootUserContainer() func(params *testPackagesParams) {
	return func(params *testPackagesParams) {
		params.HasRootUserContainer = true
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

	if params.MinModules != nil {
		args = append(args, "--min-modules", strconv.Itoa(*params.MinModules))
	}

	if params.HasMonitorsD {
		args = append(args, "--monitors.d")
	}

	if params.HasModulesD {
		args = append(args, "--modules.d")
	}

	if params.HasRootUserContainer {
		args = append(args, "--root-user-container")
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

// TestLinuxForCentosGLIBC checks the GLIBC requirements of linux/amd64 and
// linux/386 binaries to ensure they meet the requirements for RHEL 6 which has
// glibc 2.12.
func TestLinuxForCentosGLIBC() error {
	switch Platform.Name {
	case "linux/amd64", "linux/386":
		return TestBinaryGLIBCVersion(filepath.Join("build/golang-crossbuild", BeatName+"-linux-"+Platform.GOARCH), "2.12")
	default:
		return nil
	}
}

func TestBinaryGLIBCVersion(elfPath, maxGlibcVersion string) error {
	requiredGlibc, err := ReadGLIBCRequirement(elfPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	upperBound, err := NewSemanticVersion(maxGlibcVersion)
	if err != nil {
		return err
	}

	if !requiredGlibc.LessThanOrEqual(upperBound) {
		return fmt.Errorf("dynamically linked binary %q requires glibc "+
			"%v, but maximum allowed glibc is %v",
			elfPath, requiredGlibc, upperBound)
	}
	fmt.Printf(">> testBinaryGLIBCVersion: %q requires glibc %v or greater\n", elfPath, requiredGlibc)
	return nil
}

// FixDRADockerArtifacts is a workaround for the DRA artifacts produced by the package target. We had to do
// because the initial unified release manager DSL code required specific names that the package does not produce,
// we wanted to keep backwards compatibility with the artifacts of the unified release and the DRA.
// this follows the same logic as https://github.com/elastic/beats/blob/2fdefcfbc783eb4710acef07d0ff63863fa00974/.ci/scripts/prepare-release-manager.sh
func FixDRADockerArtifacts() error {
	fmt.Println("--- Fixing Docker DRA artifacts")
	distributionsPath := filepath.Join("build", "distributions")
	// Find all the files with the given name
	matches, err := filepath.Glob(filepath.Join(distributionsPath, "*docker.tar.gz*"))
	if err != nil {
		return err
	}
	if mg.Verbose() {
		log.Printf("--- Found artifacts to rename %s %d", distributionsPath, len(matches))
	}
	// Match the artifact name and break down into groups so that we can reconstruct the names as its expected by the DRA DSL
	// As SNAPSHOT keyword or BUILDID are optional, capturing the separator - or + with the value.
	artifactRegexp, err := regexp.Compile(`([\w+-]+)-(([0-9]+)\.([0-9]+)\.([0-9]+))([-|\+][\w]+)?-([\w]+)-([\w]+)\.([\w]+)\.([\w.]+)`)
	if err != nil {
		return err
	}
	for _, m := range matches {
		artifactFile, err := os.Stat(m)
		if err != nil {
			return fmt.Errorf("failed stating file: %w", err)
		}
		if artifactFile.IsDir() {
			continue
		}
		match := artifactRegexp.FindAllStringSubmatch(artifactFile.Name(), -1)
		// The groups here is tightly coupled with the regexp above.
		// match[0][6] already contains the separator so no need to add before the variable
		targetName := fmt.Sprintf("%s-%s%s-%s-image-%s-%s.%s", match[0][1], match[0][2], match[0][6], match[0][9], match[0][7], match[0][8], match[0][10])
		if mg.Verbose() {
			fmt.Printf("%#v\n", match)
			fmt.Printf("Artifact: %s \n", artifactFile.Name())
			fmt.Printf("Renamed:  %s \n", targetName)
		}
		renameErr := os.Rename(filepath.Join(distributionsPath, artifactFile.Name()), filepath.Join(distributionsPath, targetName))
		if renameErr != nil {
			return renameErr
		}
		if mg.Verbose() {
			fmt.Println("Renamed artifact")
		}
	}
	return nil
}
