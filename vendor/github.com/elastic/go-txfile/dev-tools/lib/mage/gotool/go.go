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

package gotool

import (
	"os"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Args holds parameters, environment variables and flag information used to
// pass to the go tool.
type Args struct {
	extra map[string]string // extra flags one can pass to the command
	env   map[string]string
	flags map[string]string
	pos   []string
}

// ArgOpt is a functional option adding info to Args once executed.
type ArgOpt func(args *Args)

type goTest func(opts ...ArgOpt) error

// Test runs `go test` and provides optionals for adding command line arguments.
var Test goTest = runGoTest

// ListProjectPackages lists all packages in the current project
func ListProjectPackages() ([]string, error) {
	return ListPackages("./...")
}

// ListPackages calls `go list` for every package spec given.
func ListPackages(pkgs ...string) ([]string, error) {
	return getLines(callGo(nil, "list", pkgs...))
}

// ListTestFiles lists all go and cgo test files available in a package.
func ListTestFiles(pkg string) ([]string, error) {
	const tmpl = `{{ range .TestGoFiles }}{{ printf "%s\n" . }}{{ end }}` +
		`{{ range .XTestGoFiles }}{{ printf "%s\n" . }}{{ end }}`

	return getLines(callGo(nil, "list", "-f", tmpl, pkg))
}

// HasTests returns true if the given package contains test files.
func HasTests(pkg string) (bool, error) {
	files, err := ListTestFiles(pkg)
	if err != nil {
		return false, err
	}
	return len(files) > 0, nil
}

func (goTest) WithCoverage(to string) ArgOpt {
	return combine(flagArg("-cover", ""), flagArgIf("-test.coverprofile", to))
}
func (goTest) Short(b bool) ArgOpt        { return flagBoolIf("-test.short", b) }
func (goTest) Use(bin string) ArgOpt      { return extraArgIf("use", bin) }
func (goTest) OS(os string) ArgOpt        { return envArgIf("GOOS", os) }
func (goTest) ARCH(arch string) ArgOpt    { return envArgIf("GOARCH", arch) }
func (goTest) Create() ArgOpt             { return flagArg("-c", "") }
func (goTest) Out(path string) ArgOpt     { return flagArg("-o", path) }
func (goTest) Package(path string) ArgOpt { return posArg(path) }
func (goTest) Verbose() ArgOpt            { return flagArg("-test.v", "") }
func runGoTest(opts ...ArgOpt) error {
	args := buildArgs(opts)
	if bin := args.Val("use"); bin != "" {
		flags := map[string]string{}
		for k, v := range args.flags {
			if strings.HasPrefix(k, "-test.") {
				flags[k] = v
			}
		}

		useArgs := &Args{}
		*useArgs = *args
		useArgs.flags = flags

		_, err := sh.Exec(useArgs.env, os.Stdout, os.Stderr, bin, useArgs.build()...)
		return err
	}

	return runVGo("test", args)
}

func getLines(out string, err error) ([]string, error) {
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	res := lines[:0]
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			res = append(res, line)
		}
	}

	return res, nil
}

func callGo(env map[string]string, cmd string, opts ...string) (string, error) {
	args := []string{cmd}
	args = append(args, opts...)
	return sh.OutputWith(env, mg.GoCmd(), args...)
}

func runVGo(cmd string, args *Args) error {
	return execGoWith(func(env map[string]string, cmd string, args ...string) error {
		_, err := sh.Exec(env, os.Stdout, os.Stderr, cmd, args...)
		return err
	}, cmd, args)
}

func runGo(cmd string, args *Args) error {
	return execGoWith(sh.RunWith, cmd, args)
}

func execGoWith(
	fn func(map[string]string, string, ...string) error,
	cmd string, args *Args,
) error {
	cliArgs := []string{cmd}
	cliArgs = append(cliArgs, args.build()...)
	return fn(args.env, mg.GoCmd(), cliArgs...)
}

func posArg(value string) ArgOpt {
	return func(a *Args) { a.Add(value) }
}

func extraArg(k, v string) ArgOpt {
	return func(a *Args) { a.Extra(k, v) }
}

func extraArgIf(k, v string) ArgOpt {
	if v == "" {
		return nil
	}
	return extraArg(k, v)
}

func envArg(k, v string) ArgOpt {
	return func(a *Args) { a.Env(k, v) }
}

func envArgIf(k, v string) ArgOpt {
	if v == "" {
		return nil
	}
	return envArg(k, v)
}

func flagArg(flag, value string) ArgOpt {
	return func(a *Args) { a.Flag(flag, value) }
}

func flagArgIf(flag, value string) ArgOpt {
	if value == "" {
		return nil
	}
	return flagArg(flag, value)
}

func flagBoolIf(flag string, b bool) ArgOpt {
	if b {
		return flagArg(flag, "")
	}
	return nil
}

func combine(opts ...ArgOpt) ArgOpt {
	return func(a *Args) {
		for _, opt := range opts {
			if opt != nil {
				opt(a)
			}
		}
	}
}

func buildArgs(opts []ArgOpt) *Args {
	a := &Args{}
	combine(opts...)(a)
	return a
}

// Extra sets a special k/v pair to be interpreted by the execution function.
func (a *Args) Extra(k, v string) {
	if a.extra == nil {
		a.extra = map[string]string{}
	}
	a.extra[k] = v
}

// Val returns a special functions value for a given key.
func (a *Args) Val(k string) string {
	if a.extra == nil {
		return ""
	}
	return a.extra[k]
}

// Env sets an environmant variable to be passed to the child process on exec.
func (a *Args) Env(k, v string) {
	if a.env == nil {
		a.env = map[string]string{}
	}
	a.env[k] = v
}

// Flag adds a flag to be passed to the child process on exec.
func (a *Args) Flag(flag, value string) {
	if a.flags == nil {
		a.flags = map[string]string{}
	}
	a.flags[flag] = value
}

// Add adds a positional argument to be passed to the child process on exec.
func (a *Args) Add(p string) {
	a.pos = append(a.pos, p)
}

func (a *Args) build() []string {
	args := make([]string, 0, 2*len(a.flags)+len(a.pos))
	for k, v := range a.flags {
		args = append(args, k)
		if v != "" {
			args = append(args, v)
		}
	}

	args = append(args, a.pos...)
	return args
}
