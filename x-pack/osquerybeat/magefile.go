// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/magefile/mage/mg"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/dev-tools/mage/target/build"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/command"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/distro"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/fileutil"

	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/scripts/mage"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/notests"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	devtools.BeatDescription = "Osquerybeat is a beat implementation for osquery."
	devtools.BeatLicense = "Elastic License"
}

func Fmt() {
	mg.Deps(devtools.Format)
}

func AddLicenseHeaders() {
	mg.Deps(devtools.AddLicenseHeaders)
}

func Check() error {
	mg.Deps(Generate)
	return devtools.Check()
}

// Generate runs osquery-extension code generators and metadata generators.
func Generate() error {
	ctx := context.Background()

	// Generate tables/views/docs/README from specs.
	if err := execCommand(ctx, "bash", "-c", "cd ext/osquery-extension/cmd/gentables && go generate ./..."); err != nil {
		return err
	}

	// Generate jumplists lookup maps (outputs remain windows-tagged).
	jumplistGenCmd := "cd ext/osquery-extension/pkg/jumplists && go run ./generate"
	if strings.EqualFold(os.Getenv("JUMPLISTS_REFRESH_SOURCES"), "true") {
		jumplistGenCmd += " -refresh-sources"
	}
	if err := execCommand(ctx, "bash", "-c", jumplistGenCmd); err != nil {
		return err
	}

	// Ensure jumplists generated files are consistently formatted.
	if err := execCommand(ctx, "bash", "-c", "cd ext/osquery-extension/pkg/jumplists && gofmt -w generated_app_ids.go generated_guid_mappings.go"); err != nil {
		return err
	}
	if err := execCommand(ctx, "bash", "-c", "cd ext/osquery-extension/pkg/jumplists && if command -v goimports >/dev/null 2>&1; then goimports -w generated_app_ids.go generated_guid_mappings.go; else go run golang.org/x/tools/cmd/goimports@latest -w generated_app_ids.go generated_guid_mappings.go; fi"); err != nil {
		return err
	}

	return nil
}

func Build() error {
	// Building osquerybeat
	err := devtools.Build(devtools.DefaultBuildArgs())
	if err != nil {
		return err
	}
	return BuildExt()
}

// BuildExt builds the osquery-extension.
func BuildExt() error {
	params := devtools.DefaultBuildArgs()
	params.InputFiles = []string{"./ext/osquery-extension/."}
	params.Name = "osquery-extension"
	params.CGO = true
	err := devtools.Build(params)
	if err != nil {
		return err
	}

	// Rename osquery-extension to osquery-extension.ext on non windows platforms
	if runtime.GOOS != "windows" {
		err = os.Rename("osquery-extension", "osquery-extension.ext")
		if err != nil {
			return err
		}
	}
	return nil
}

// Clean cleans all generated files and build artifacts.
func Clean() error {
	paths := devtools.DefaultCleanPaths
	paths = append(paths, []string{
		"osquery-extension",
		"osquery-extension.exe",
		filepath.Join("ext", "osquery-extension", "build"),
	}...)
	return devtools.Clean(paths)
}

func execCommand(ctx context.Context, name string, args ...string) error {
	ps := strings.Join(append([]string{name}, args...), " ")
	fmt.Println("Executing command: ", ps)
	output, err := command.Execute(ctx, name, args...)
	if err != nil {
		fmt.Println(ps, ", failed: ", err)
		return err
	}
	fmt.Print(output)
	return err
}

// stripLinuxOsqueryd Strips osqueryd binary, that is not stripped in linux tar.gz distro
func stripLinuxOsqueryd() error {
	if os.Getenv("GOOS") != "linux" {
		return nil
	}

	// Check that this step is called during x-pack/osquerybeat/ext/osquery-extension build
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Strip osqueryd only once when osquery-extension is built
	if !strings.HasSuffix(cwd, "/osquery-extension") {
		return nil
	}

	ctx := context.Background()

	osArchs := osquerybeat.OSArchs(devtools.Platforms)

	strip := func(oquerydPath string, target distro.OSArch) error {
		ok, err := fileutil.FileExists(oquerydPath)
		if err != nil {
			return err
		}
		if ok {
			if err := execCommand(ctx, stripCommand(target), oquerydPath); err != nil {
				return err
			}
		}
		return nil
	}

	for _, osarch := range osArchs {
		// Skip everything but matching linux arch
		if osarch.OS != os.Getenv("GOOS") || osarch.Arch != os.Getenv("GOARCH") {
			continue
		}

		// Strip osqueryd

		// This returns something like build/data/install/linux/amd64/osqueryd
		querydRelativePath := distro.OsquerydPath(distro.GetDataInstallDir(osarch))

		// Checking and stripping osqueryd binary
		osquerybeatPath := filepath.Clean(filepath.Join(cwd, "../..", querydRelativePath))
		err = strip(osquerybeatPath, osarch)
		if err != nil {
			return err
		}
	}

	return nil
}

// GolangCrossBuild build the Beat binary inside of the golang-builder.
// Do not use directly, use crossBuild instead.
func GolangCrossBuild() error {
	// Strip linux osqueryd binary
	if err := stripLinuxOsqueryd(); err != nil {
		return err
	}

	return devtools.GolangCrossBuild(devtools.DefaultGolangCrossBuildArgs())
}

// CrossBuild cross-builds the beat for all target platforms.
func CrossBuild() error {
	// Building osquerybeat
	err := devtools.CrossBuild()
	if err != nil {
		return err
	}
	return CrossBuildExt()
}

// CrossBuildExt cross-builds the osquery-extension.
func CrossBuildExt() error {
	return devtools.CrossBuild(devtools.InDir("x-pack", "osquerybeat", "ext", "osquery-extension"))
}

// AssembleDarwinUniversal merges the darwin/amd64 and darwin/arm64 into a single
// universal binary using `lipo`. It assumes the darwin/amd64 and darwin/arm64
// were built and only performs the merge.
func AssembleDarwinUniversal() error {
	return build.AssembleDarwinUniversal()
}

// Package packages the Beat for distribution.
// Use SNAPSHOT=true to build snapshots.
// Use PLATFORMS to control the target platforms.
// Use VERSION_QUALIFIER to control the version qualifier.
func Package() {
	start := time.Now()
	defer func() { fmt.Println("package ran for", time.Since(start)) }()

	devtools.MustUsePackaging("osquerybeat", "x-pack/osquerybeat/dev-tools/packaging/packages.yml")

	// Add osquery distro binaries
	osquerybeat.CustomizePackaging()

	mg.Deps(Update, osquerybeat.FetchOsqueryDistros)
	mg.Deps(CrossBuild)
	mg.SerialDeps(devtools.Package, TestPackages)
}

// Package packages the Beat for IronBank distribution.
//
// Use SNAPSHOT=true to build snapshots.
func Ironbank() error {
	fmt.Println(">> Ironbank: this module is not subscribed to the IronBank releases.")
	return nil
}

// TestPackages tests the generated packages (i.e. file modes, owners, groups).
func TestPackages() error {
	return devtools.TestPackages()
}

// Update is an alias for update:all. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Update() { mg.Deps(osquerybeat.Update.All) }

// Fields is an alias for update:fields. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Fields() { mg.Deps(osquerybeat.Update.Fields) }

// Config is an alias for update:config. This is a workaround for
// https://github.com/magefile/mage/issues/217.
func Config() { mg.Deps(osquerybeat.Update.Config) }

func stripCommand(target distro.OSArch) string {
	if target.OS != "linux" {
		return "strip" // fallback
	}

	switch target.Arch {
	case "arm64":
		return "aarch64-linux-gnu-strip"
	case "amd64":
		return "x86_64-linux-gnu-strip"
	}

	return "strip"
}
