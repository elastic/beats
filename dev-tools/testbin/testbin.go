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
// On Windows 386, DWARF is stripped (-ldflags=-w) and coverage is disabled
// to avoid out-of-memory failures.
package testbin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/execabs"
)

// Build compiles a test binary for the given beat using "go test -c".
// dir is the beat root directory where "go test -c" runs and where the
// resulting binary is written. It returns the absolute path of the built
// binary.
func Build(beatName, dir string) (string, error) {
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

	// On Windows 386 we run out of memory if we enable coverage and DWARF.
	win386 := runtime.GOOS == "windows" && runtime.GOARCH == "386"
	if win386 {
		args = append(args, "-ldflags=-w")
	}
	if testCoverage, _ := strconv.ParseBool(os.Getenv("TEST_COVERAGE")); testCoverage && !win386 {
		args = append(args, "-coverpkg", "./...")
	}

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
