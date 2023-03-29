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
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	goBenchstat = "golang.org/x/perf/cmd/benchstat@v0.0.0-20230227161431-f7320a6d63e8"
)

var (
	benchmarkCount = 8
)

// Benchmark namespace for mage to group all the related targets under this namespace
type Benchmark mg.Namespace

// Deps installs required plugins for reading benchmarks results
func (Benchmark) Deps() error {
	err := gotool.Install(gotool.Install.Package(goBenchstat))
	if err != nil {
		return err
	}
	return nil
}

// Run execute the go benchmark tests for this repository, by defining the variable OUTPUT you write the results
// into a file. Optional you can set BENCH_COUNT to how many benchmark iteration you want to execute, default is 8
func (Benchmark) Run() error {
	mg.Deps(Benchmark.Deps)
	log.Println(">> go Test: Benchmark")
	outputFile := os.Getenv("OUTPUT")
	benchmarkCountOverride := os.Getenv("BENCH_COUNT")
	if benchmarkCountOverride != "" {
		var overrideErr error
		benchmarkCount, overrideErr = strconv.Atoi(benchmarkCountOverride)
		if overrideErr != nil {
			return fmt.Errorf("failed to parse BENCH_COUNT, verify that you set the right value: , %w", overrideErr)
		}
	}
	projectPackages, er := gotool.ListProjectPackages()
	if er != nil {
		return fmt.Errorf("failed to list package dependencies: %w", er)
	}
	cmdArg := fmt.Sprintf("test -count=%d -bench=Bench -run=Bench", benchmarkCount)
	cmdArgs := strings.Split(cmdArg, " ")
	for _, pkg := range projectPackages {
		cmdArgs = append(cmdArgs, filepath.Join(pkg, "/..."))
	}
	_, err := runCommand(nil, "go", outputFile, cmdArgs...)

	var goTestErr *exec.ExitError
	switch {
	case goTestErr == nil:
		return nil
	case errors.As(err, &goTestErr):
		return fmt.Errorf("failed to execute go test -bench command: %w", err)
	default:
		return fmt.Errorf("failed to execute go test -bench command %w", err)
	}
}

// Diff parse one or more benchmark outputs, Required environment variables are BASE for parsing results
// and NEXT to compare the base results with. Optional you can define OUTPUT to write the results into a file
func (Benchmark) Diff() error {
	mg.Deps(Benchmark.Deps)
	log.Println(">> running: benchstat")
	outputFile := os.Getenv("OUTPUT")
	baseFile := os.Getenv("BASE")
	nextFile := os.Getenv("NEXT")
	var args []string
	if baseFile == "" {
		log.Printf("Missing required parameter BASE parameter to parse the results. Please set this to a filepath of the benchmark results")
		return fmt.Errorf("missing required parameter BASE parameter to parse the results. Please set this to a filepath of the benchmark results")
	} else {
		args = append(args, baseFile)
	}
	if nextFile == "" {
		log.Printf("Missing NEXT parameter, we are not going to compare results")
	} else {
		args = append(args, nextFile)
	}

	_, err := runCommand(nil, "benchstat", outputFile, args...)

	var goTestErr *exec.ExitError
	switch {
	case goTestErr == nil:
		return nil
	case errors.As(err, &goTestErr):
		return fmt.Errorf("failed to execute benchstat command: %w", err)
	default:
		return fmt.Errorf("failed to execute benchstat command!! %w", err)
	}

}

// runCommand is executing a command that is represented by cmd.
// when defining an outputFile it will write the stdErr, stdOut of that command to the output file
// otherwise it will capture it to stdErr, stdOut of the console used and return true, nil, if succeed
func runCommand(env map[string]string, cmd string, outputFile string, args ...string) (bool, error) {
	var stdOut io.Writer
	var stdErr io.Writer
	if outputFile != "" {
		fileOutput, err := os.Create(createDir(outputFile))
		if err != nil {
			return false, fmt.Errorf("failed to create %s output file: %w", cmd, err)
		}
		defer func(fileOutput *os.File) {
			err := fileOutput.Close()
			if err != nil {
				log.Fatalf("Failed to close file %s", err)
			}
		}(fileOutput)
		stdOut = io.MultiWriter(os.Stdout, fileOutput)
		stdErr = io.MultiWriter(os.Stderr, fileOutput)
	} else {
		stdOut = os.Stdout
		stdErr = os.Stderr
	}
	return sh.Exec(env, stdOut, stdErr, cmd, args...)
}

// createDir creates the parent directory for the given file.
func createDir(file string) string {
	// Create the output directory.
	if dir := filepath.Dir(file); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create parent dir for %s", file)
		}
	}
	return file
}
