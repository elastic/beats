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
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/pkg/errors"
)

// BuildArgs are the arguments used for the "build" target and they define how
// "go build" is invoked.
type BuildArgs struct {
	Name       string // Name of binary. (On Windows '.exe' is appended.)
	OutputDir  string
	CGO        bool
	Static     bool
	Env        map[string]string
	LDFlags    []string
	Vars       map[string]string // Vars that are passed as -X key=value with the ldflags.
	ExtraFlags []string
}

// DefaultBuildArgs returns the default BuildArgs for use in builds.
func DefaultBuildArgs() BuildArgs {
	args := BuildArgs{
		Name: BeatName,
		CGO:  build.Default.CgoEnabled,
		Vars: map[string]string{
			"github.com/elastic/beats/libbeat/version.buildTime": "{{ date }}",
			"github.com/elastic/beats/libbeat/version.commit":    "{{ commit }}",
		},
	}

	if versionQualified {
		args.Vars["github.com/elastic/beats/libbeat/version.qualifier"] = "{{ .Qualifier }}"
	}

	repo, err := GetProjectRepoInfo()
	if err != nil {
		panic(errors.Wrap(err, "failed to determine project repo info"))
	}

	if !repo.IsElasticBeats() {
		// Assume libbeat is vendored and prefix the variables.
		prefix := repo.RootImportPath + "/vendor/"
		prefixedVars := map[string]string{}
		for k, v := range args.Vars {
			prefixedVars[prefix+k] = v
		}
		args.Vars = prefixedVars
	}

	return args
}

// DefaultGolangCrossBuildArgs returns the default BuildArgs for use in
// cross-builds.
func DefaultGolangCrossBuildArgs() BuildArgs {
	args := DefaultBuildArgs()
	args.Name += "-" + Platform.GOOS + "-" + Platform.Arch
	args.OutputDir = filepath.Join("build", "golang-crossbuild")
	if bp, found := BuildPlatforms.Get(Platform.Name); found {
		args.CGO = bp.Flags.SupportsCGO()
	}
	return args
}

// GolangCrossBuild invokes "go build" inside of the golang-crossbuild Docker
// environment.
func GolangCrossBuild(params BuildArgs) error {
	if os.Getenv("GOLANG_CROSSBUILD") != "1" {
		return errors.New("Use the crossBuild target. golangCrossBuild can " +
			"only be executed within the golang-crossbuild docker environment.")
	}

	defer DockerChown(filepath.Join(params.OutputDir, params.Name+binaryExtension(GOOS)))
	return Build(params)
}

// Build invokes "go build" to produce a binary.
func Build(params BuildArgs) error {
	fmt.Println(">> build: Building", params.Name)

	binaryName := params.Name + binaryExtension(GOOS)

	if params.OutputDir != "" {
		if err := os.MkdirAll(params.OutputDir, 0755); err != nil {
			return err
		}
	}

	// Environment
	env := params.Env
	if env == nil {
		env = map[string]string{}
	}
	cgoEnabled := "0"
	if params.CGO {
		cgoEnabled = "1"
	}
	env["CGO_ENABLED"] = cgoEnabled

	// Spec
	args := []string{
		"build",
		"-o",
		filepath.Join(params.OutputDir, binaryName),
	}
	args = append(args, params.ExtraFlags...)

	// ldflags
	ldflags := params.LDFlags
	if params.Static {
		ldflags = append(ldflags, `-extldflags '-static'`)
	}
	for k, v := range params.Vars {
		ldflags = append(ldflags, fmt.Sprintf("-X %v=%v", k, v))
	}
	if len(ldflags) > 0 {
		args = append(args, "-ldflags")
		args = append(args, MustExpand(strings.Join(ldflags, " ")))
	}

	log.Println("Adding build environment vars:", env)
	return sh.RunWith(env, "go", args...)
}
