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

// Package testbin builds Go test binaries via "go test -c". It is the single
// source of truth for how beat test binaries are compiled, used by both the
// mage build system and the Go integration test framework.
//
// Environment variables respected:
//   - DEV=true: disables optimizations for debugging (-gcflags=all=-N -l)
//   - TEST_COVERAGE=true: enables coverage instrumentation (-coverpkg ./...)
//
// Platform-specific flags (e.g. stripping DWARF on Windows 386) should be
// passed by the caller via BuildOptions.ExtraFlags.
package testbin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/execabs"
)

// buildOptions holds the resolved build configuration.
type buildOptions struct {
	extraFlags []string
	inputFiles []string
}

// Option configures a Build invocation. Options are applied in order,
// so later options override earlier ones.
type Option func(*buildOptions)

// WithExtraFlags appends additional flags passed to 'go test'.
func WithExtraFlags(flags ...string) Option {
	return func(o *buildOptions) {
		o.extraFlags = append(o.extraFlags, flags...)
	}
}

// WithInputFiles appends specific files/packages after all flags.
func WithInputFiles(files ...string) Option {
	return func(o *buildOptions) {
		o.inputFiles = append(o.inputFiles, files...)
	}
}

// Build compiles a test binary for the given beat using "go test -c".
// dir is the beat root directory where "go test -c" runs and where the
// resulting binary is written. It returns the absolute path of the built
// binary.
func Build(beatName, dir string, opts ...Option) (string, error) {
	if !strings.HasSuffix(beatName, ".test") {
		beatName += ".test"
	}
	outputPath, err := filepath.Abs(filepath.Join(dir, beatName))
	if err != nil {
		return "", fmt.Errorf("failed to resolve output path: %w", err)
	}

	args := []string{"test", "-c", "-o", outputPath}

	if devBuild, _ := strconv.ParseBool(os.Getenv("DEV")); devBuild {
		args = append(args, `-gcflags=all=-N -l`)
	}

	if testCoverage, _ := strconv.ParseBool(os.Getenv("TEST_COVERAGE")); testCoverage {
		args = append(args, "-coverpkg", "./...")
	}

	var o buildOptions
	for _, fn := range opts {
		fn(&o)
	}
	args = append(args, o.extraFlags...)
	args = append(args, o.inputFiles...)

	cmd := execabs.Command("go", args...)
	cmd.Dir = dir

	start := time.Now()
	defer func() {
		log.Printf("testbin.Build (go %s) took %v.",
			strings.Join(args, " "), time.Since(start))
	}()

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("testbin.Build failed:\n%s", output)
		return "", fmt.Errorf("failed to build test binary %q: %w (see log output for details)", beatName, err)
	}

	return outputPath, nil
}
