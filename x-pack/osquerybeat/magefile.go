// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build mage

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/osqd"

	osquerybeat "github.com/elastic/beats/v7/x-pack/osquerybeat/scripts/mage"

	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/pkg"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/unittest"
	// mage:import
	_ "github.com/elastic/beats/v7/dev-tools/mage/target/integtest/docker"
	// mage:import
	"github.com/elastic/beats/v7/dev-tools/mage/target/test"
)

func init() {
	test.RegisterDeps(IntegTest)

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

// IntegTest executes integration tests (it uses Docker to run the tests).
func IntegTest() {
	mg.SerialDeps(GoIntegTest)
}

// GoIntegTest starts the docker containers and executes the Go integration tests.
func GoIntegTest(ctx context.Context) error {
	mg.Deps(FetchOsquerydForTesting)

	args := devtools.DefaultGoTestIntegrationFromHostArgs(ctx)
	// ES_USER must be admin for the Go integration tests to function because they require
	// indices:data/read/search
	args.Env["ES_USER"] = args.Env["ES_SUPERUSER_USER"]
	args.Env["ES_PASS"] = args.Env["ES_SUPERUSER_PASS"]

	// On macOS with GCC as the default CGO compiler, the system headers in the macOS 26
	// SDK use Objective-C block syntax (^) which GCC doesn't support. Use clang instead.
	if runtime.GOOS == "darwin" {
		if clang, err := exec.LookPath("clang"); err == nil {
			args.Env["CC"] = clang
		}
	}

	// Expand ~/go/bin (and any other tilde paths) in PATH so that exec.LookPath finds
	// binaries installed by "go install" (like gotestsum). Shell configs often set PATH
	// with a literal "~" which is not expanded by exec.LookPath in Go, causing the lookup
	// to silently fail. We also pass the expanded PATH to the subprocess environment.
	if gopathOut, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
		gopathBin := filepath.Join(strings.TrimSpace(string(gopathOut)), "bin")
		currentPath := os.Getenv("PATH")
		if !strings.Contains(currentPath, gopathBin) {
			expandedPath := gopathBin + string(filepath.ListSeparator) + currentPath
			os.Setenv("PATH", expandedPath) //nolint:errcheck // best-effort
			args.Env["PATH"] = expandedPath
		}
	}

	// Tell osquerybeat (both the in-process OTel receiver and the subprocess
	// standalone beat) where to find the osqueryd binary, so tests don't
	// require it to be installed system-wide.
	osarch := distro.OSArch{OS: runtime.GOOS, Arch: runtime.GOARCH}
	binDir, err := filepath.Abs(distro.GetDataInstallDir(osarch))
	if err == nil {
		if _, statErr := os.Stat(osqd.OsquerydPathForPlatform(runtime.GOOS, binDir)); statErr == nil {
			args.Env["OSQUERYBEAT_BINARY_DIR"] = binDir
		}
		// osquerybeat requires osquery-extension alongside osqueryd; build and copy it if absent.
		if extErr := ensureExtensionInBinDir(binDir); extErr != nil {
			return fmt.Errorf("failed to ensure osquery-extension in %s: %w", binDir, extErr)
		}
	}

	return devtools.GoIntegTestFromHost(ctx, args)
}

// FetchOsquerydForTesting downloads the osqueryd binary for the current host
// platform using the same infrastructure as the package build. The binary is
// placed at build/data/install/{os}/{arch}/osqueryd.
func FetchOsquerydForTesting() error {
	prevPlatforms := devtools.Platforms
	devtools.Platforms = devtools.NewPlatformList(runtime.GOOS + "/" + runtime.GOARCH)
	defer func() { devtools.Platforms = prevPlatforms }()
	return osquerybeat.FetchOsqueryDistros()
}

// ensureExtensionInBinDir places osquery-extension in binDir, building it first if absent.
func ensureExtensionInBinDir(binDir string) error {
	extName := "osquery-extension.ext"
	if runtime.GOOS == "windows" {
		extName = "osquery-extension.exe"
	}
	destPath := filepath.Join(binDir, extName)
	if _, err := os.Stat(destPath); err == nil {
		return nil
	}
	if _, err := os.Stat(extName); os.IsNotExist(err) {
		if err := BuildExt(); err != nil {
			return err
		}
	}
	return copyFile(extName, destPath)
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
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
