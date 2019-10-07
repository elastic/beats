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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/elastic/go-txfile/dev-tools/lib/mage/gotool"
	"github.com/elastic/go-txfile/dev-tools/lib/mage/mgenv"
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
func (Prepare) Dirs() error { return mkdir("build") }

// Lint runs golint
func (Check) Lint() error {
	return errors.New("TODO: implement me")
}

// Clean removes build artifacts
func Clean() error {
	return sh.Rm(buildHome)
}

// Mage builds the magefile binary for reuse
func (Build) Mage() error {
	mg.Deps(Prepare.Dirs)

	goos := envBuildOS
	goarch := envBuildArch
	out := filepath.Join(buildHome, fmt.Sprintf("mage-%v-%v", goos, goarch))
	return sh.Run("mage", "-f", "-goos="+goos, "-goarch="+goarch, "-compile", out)
}

// Test builds the per package unit test executables.
func (Build) Test() error {
	mg.Deps(Prepare.Dirs)

	return withList(gotool.ListProjectPackages, failFastEach, func(pkg string) error {
		tst := gotool.Test
		return tst(
			tst.OS(envBuildOS),
			tst.ARCH(envBuildArch),
			tst.Create(),
			tst.WithCoverage(""),
			tst.Out(path.Join(buildHome, pkg, path.Base(pkg))),
			tst.Package(pkg),
		)
	})
}

// Test runs the unit tests.
func Test() error {
	mg.Deps(Prepare.Dirs)

	if crossBuild() {
		return withXProvider(func(p xbuild.Provider) error {
			mg.Deps(Build.Mage, Build.Test)

			env := mgenv.MakeEnv()
			env["TEST_USE_BIN"] = "true"
			return p.Run(env, "./build/mage-linux-arm", useIf("-v", mg.Verbose()), "test")
		})
	}

	return withList(gotool.ListProjectPackages, failFastEach, func(pkg string) error {
		fmt.Println("Test:", pkg)
		if b, err := gotool.HasTests(pkg); !b {
			fmt.Printf("Skipping %v: No tests found\n", pkg)
			return err
		}

		home := path.Join(buildHome, pkg)
		if err := mkdir(home); err != nil {
			return err
		}

		tst := gotool.Test
		bin := path.Join(home, path.Base(pkg))
		return tst(
			tst.Use(useIf(bin, existsFile(bin) && envTestUseBin)),
			tst.WithCoverage(path.Join(home, "cover.out")),
			tst.Short(envTestShort),
			tst.Out(bin),
			tst.Package(pkg),
			tst.Verbose(),
		)
	})
}

// helpers

func withList(
	gen func() ([]string, error),
	mode func(...func() error) error,
	fn func(string) error,
) error {
	list, err := gen()
	if err != nil {
		return err
	}

	ops := make([]func() error, len(list))
	for i, v := range list {
		v := v
		ops[i] = func() error { return fn(v) }
	}

	return mode(ops...)
}

func useIf(s string, b bool) string {
	if b {
		return s
	}
	return ""
}

func existsFile(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular()
}

func mkdirs(paths ...string) error {
	for _, p := range paths {
		if err := mkdir(p); err != nil {
			return err
		}
	}
	return nil
}

func mkdir(path string) error {
	return os.MkdirAll(path, os.ModeDir|0700)
}

func failFastEach(ops ...func() error) error {
	mode := each
	if envFailFast {
		mode = and
	}
	return mode(ops...)
}

func each(ops ...func() error) error {
	var errs []error
	for _, op := range ops {
		if err := op(); err != nil {
			errs = append(errs, err)
		}
	}
	return makeErrs(errs)
}

func and(ops ...func() error) error {
	for _, op := range ops {
		if err := op(); err != nil {
			return err
		}
	}
	return nil
}

type multiErr []error

func makeErrs(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	return multiErr(errs)
}

func (m multiErr) Error() string {
	var bld strings.Builder
	for _, err := range m {
		if bld.Len() > 0 {
			bld.WriteByte('\n')
			bld.WriteString(err.Error())
		}
	}
	return bld.String()
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

func withXProvider(fn func(p xbuild.Provider) error) error {
	return xProviders.With(envBuildOS, envBuildArch, fn)
}
