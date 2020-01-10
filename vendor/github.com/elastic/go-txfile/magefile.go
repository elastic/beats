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

//+build mage

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/urso/magetools/clitool"
	"github.com/urso/magetools/ctrl"
	"github.com/urso/magetools/fs"
	"github.com/urso/magetools/gotool"
	"github.com/urso/magetools/mgenv"

	"github.com/elastic/go-txfile/dev-tools/lib/mage/xbuild"
)

// Info namespace is used to print additional docs, help messages, and other info.
type Info mg.Namespace

// Prepare namespace is used to prepare/download/build common depenendencies for other tasks to run.
type Prepare mg.Namespace

// Check runs pre-build checks on the environment and source code. (e.g. linters)
type Check mg.Namespace

// Build namespace defines the set of build targets
type Build mg.Namespace

const buildHome = "build"

// environment variables
var (
	envBuildOS    = mgenv.String("BUILD_OS", runtime.GOOS, "(string) set compiler target GOOS")
	envBuildArch  = mgenv.String("BUILD_ARCH", runtime.GOARCH, "(string) set compiler target GOARCH")
	envTestUseBin = mgenv.Bool("TEST_USE_BIN", false, "(bool) reuse prebuild test binary when running tests")
	envTestShort  = mgenv.Bool("TEST_SHORT", false, "(bool) run tests with -short flag")
	envFailFast   = mgenv.Bool("FAIL_FAST", false, "(bool) do not run other tasks on failure")
)

var xProviders = xbuild.NewRegistry(map[xbuild.OSArch]xbuild.Provider{
	xbuild.OSArch{"linux", "arm"}: &xbuild.DockerImage{
		Image:   "balenalib/revpi-core-3-alpine-golang:latest-edge-build",
		Workdir: "/go/src/github.com/elastic/go-txfile",
		Volumes: map[string]string{
			filepath.Join(os.Getenv("GOPATH"), "src"): "/go/src",
		},
	},
	xbuild.OSArch{"linux", runtime.GOARCH}: &xbuild.DockerImage{
		Image:   golangVersionImage(),
		Workdir: "/go/src/github.com/elastic/go-txfile",
		Volumes: map[string]string{
			filepath.Join(os.Getenv("GOPATH"), "src"): "/go/src",
		},
	},
})

// targets

// Env prints environment info
func (Info) Env() {
	printTitle("Mage environment variables")
	for _, k := range mgenv.Keys() {
		v, _ := mgenv.Find(k)
		fmt.Printf("%v=%v\n", k, v.Get())
	}
	fmt.Println()

	printTitle("Go info")
	sh.RunV(mg.GoCmd(), "env")
	fmt.Println()

	printTitle("docker info")
	sh.RunV("docker", "version")
}

// Vars prints the list of registered environment variables
func (Info) Vars() {
	for _, k := range mgenv.Keys() {
		v, _ := mgenv.Find(k)
		fmt.Printf("%v=%v  : %v\n", k, v.Default(), v.Doc())
	}
}

// All runs all Prepare tasks
func (Prepare) All() { mg.Deps(Prepare.Dirs) }

// Dirs creates requires build directories for storing artifacts
func (Prepare) Dirs() error { return fs.MakeDirs("build") }

// Lint runs golint
func (Check) Lint() error {
	return errors.New("TODO: implement me")
}

// Clean removes build artifacts
func Clean() error {
	return sh.Rm(buildHome)
}

// Test builds the per package unit test executables.
func (Build) Test() error {
	mg.Deps(Prepare.Dirs)

	goRun := gotool.New(clitool.NewCLIExecutor(true), mg.GoCmd())
	return ctrl.ForEachFrom(goRun.List.ProjectPackages, failFastEach, func(pkg string) error {
		fmt.Println("Compile test binary for package", pkg)

		tst := goRun.Test
		return tst(
			context.Background(),
			tst.OS(envBuildOS),
			tst.ARCH(envBuildArch),
			tst.Create(true),
			tst.Out(path.Join(buildHome, pkg, path.Base(pkg))),
			tst.WithCoverage(""),
			tst.Package(pkg),
		)
	})
}

// Shell tries to start an interactive shell if a crossbuild environment is configured.
func (Build) Shell() error {
	if !crossBuild() {
		return errors.New("No cross build environment configured")
	}
	return withXProvider((xbuild.Provider).Shell)
}

// Test runs the unit tests.
func Test() error {
	mg.Deps(Prepare.Dirs)
	return withExecEnv(func(local, runner clitool.Executor) error {
		testUseBin := envTestUseBin
		if crossBuild() {
			mg.Deps(Build.Test)
			testUseBin = true
		}

		goLocal := gotool.New(local, mg.GoCmd())
		goRun := gotool.New(runner, mg.GoCmd())

		return ctrl.ForEachFrom(goLocal.List.ProjectPackages, failFastEach, func(pkg string) error {
			fmt.Println("Test:", pkg)
			if b, err := goLocal.List.HasTests(pkg); !b {
				fmt.Printf("Skipping %v: No tests found\n", pkg)
				return err
			}

			home := path.Join(buildHome, pkg)
			if err := fs.MakeDirs(home); err != nil {
				return err
			}

			tst := goRun.Test
			bin := path.Join(home, path.Base(pkg))
			useBinary := fs.ExistsFile(bin) && testUseBin
			fmt.Printf("Run test for package '%v' (binary: %v)\n", pkg, useBinary)

			return tst(
				context.Background(),
				tst.UseBinaryIf(bin, useBinary),
				tst.WithCoverage(path.Join(home, "cover.out")),
				tst.Short(envTestShort),
				tst.Out(bin),
				tst.Package(pkg),
				tst.Verbose(true),
			)
		})
	})
}

// helpers

func failFastEach(ops ...ctrl.Operation) error {
	mode := ctrl.Each
	if envFailFast {
		mode = ctrl.Sequential
	}
	return mode(ops...)
}

func printTitle(s string) {
	fmt.Println(s)
	for range s {
		fmt.Print("=")
	}
	fmt.Println()
}

func crossBuild() bool {
	return envBuildArch != runtime.GOARCH || envBuildOS != runtime.GOOS
}

func withXProvider(fn func(xbuild.Provider) error) error {
	return xProviders.With(envBuildOS, envBuildArch, fn)
}

func withExecEnv(fn func(local, remote clitool.Executor) error) error {
	local := clitool.NewCLIExecutor(mg.Verbose())

	if crossBuild() {
		return withXProvider(func(p xbuild.Provider) error {
			e, err := p.Executor(mg.Verbose())
			if err != nil {
				return err
			}

			return fn(local, e)
		})
	}

	return fn(local, local)
}

func golangVersionImage() string {
	version := runtime.Version()
	if strings.HasPrefix(version, "go") {
		version = version[2:]
	} else {
		version = "latest"
	}
	return fmt.Sprintf("golang:%v", version)
}
