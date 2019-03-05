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
	"errors"
	"log"
	"os"
)

var (
	defaultCrossBuildGoDaemon = []CrossBuildOption{
		ForPlatforms("linux"),
		WithTarget("buildGoDaemon"),
	}
)

// BuildGoDaemon builds the go-deamon binary.
func BuildGoDaemon() error {
	if GOOS != "linux" {
		return errors.New("go-daemon only builds for linux")
	}

	if os.Getenv("GOLANG_CROSSBUILD") != "1" {
		return errors.New("Use the crossBuildGoDaemon target. buildGoDaemon can " +
			"only be executed within the golang-crossbuild docker environment.")
	}

	// Test if binaries are up-to-date.
	output := MustExpand("build/golang-crossbuild/god-{{.Platform.GOOS}}-{{.Platform.Arch}}")
	input := MustExpand("{{ elastic_beats_dir }}/dev-tools/vendor/github.com/tsg/go-daemon/god.c")
	if IsUpToDate(output, input) {
		log.Println(">>> buildGoDaemon is up-to-date for", Platform.Name)
		return nil
	}

	// Determine what compiler to use based on CC that is set by golang-crossbuild.
	cc := os.Getenv("CC")
	if cc == "" {
		cc = "cc"
	}

	compileCmd := []string{
		cc,
		input,
		"-o", createDir(output),
		"-lpthread", "-static",
	}
	switch Platform.Name {
	case "linux/amd64":
		compileCmd = append(compileCmd, "-m64")
	case "linux/386":
		compileCmd = append(compileCmd, "-m32")
	}

	defer DockerChown(output)
	return RunCmds(compileCmd)
}

// CrossBuildGoDaemon cross-build the go-daemon binary using the
// golang-crossbuild environment.
func CrossBuildGoDaemon(options ...CrossBuildOption) error {
	opts := append(defaultCrossBuildGoDaemon, options...)
	return CrossBuild(opts...)
}
