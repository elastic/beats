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

package integration

import (
	"bufio"
	"context"
	"fmt"
<<<<<<< HEAD
=======
	"os"
	"regexp"
	"strings"
>>>>>>> e56b7d5e1 (Extend the integration testing framework (#48948))
	"testing"

	"github.com/elastic/beats/v7/libbeat/testing/integration"
)

// EnsureCompiled ensures that Filebeat is compiled and ready
// to run.
func EnsureCompiled(ctx context.Context, t *testing.T) (path string) {
	return integration.EnsureCompiled(ctx, t, "filebeat")
}

// Test describes all operations for testing Filebeat
//
// Due to interface composition all Filebeat-specific functions
// must be used first in the call-chain.
type Test interface {
	integration.BeatTest
	// ExpectEOF sets an expectation that Filebeat will read the given
	// files to EOF.
	ExpectEOF(...string) Test
	// ExpectIngestedToConsole sets an expectation that the given
	// range of lines from a given file will be ingested and printed to the console.
	//
	// It's based on the `ExpectOutput` function, so use the `console` output
	// when setting this expectation.
	// Make sure `output.console.bulk_max_size` is set to `0`
	ExpectIngestedToConsole(file string, offset, count int) Test
}

// TestOptions describes all available options for the test.
type TestOptions struct {
	// Config for the Beat written in YAML
	Config string
	// Args sets additional arguments to pass when running the binary.
	Args []string
}

// NewTest creates a new integration test for Filebeat.
func NewTest(t *testing.T, opts TestOptions) Test {
	return &test{
		BeatTest: integration.NewBeatTest(t, integration.BeatTestOptions{
			Beatname: "filebeat",
			Config:   opts.Config,
			Args:     opts.Args,
		}),
	}
}

type test struct {
	integration.BeatTest
}

// ExpectEOF implements the Test interface.
func (fbt *test) ExpectEOF(files ...string) Test {
	// Ensuring we completely ingest every file
	for _, filename := range files {
		line := fmt.Sprintf("End of file reached: %s; Backoff now.", filename)
		fbt.ExpectOutput(line)
	}

	return fbt
}

// ExpectIngestedToConsole implements the Test interface.
func (fbt *test) ExpectIngestedToConsole(file string, offset, count int) Test {
	f, err := os.Open(file)
	if err != nil {
		fbt.T().Fatalf("failed to open %q: %s", file, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if offset != 0 {
			offset--
			continue
		}
		if count == 0 {
			break
		}

		lines = append(lines, scanner.Text())
		count--
	}

	if err := scanner.Err(); err != nil {
		fbt.T().Fatalf("failed to read lines from %q: %s", file, err)
	}

	fbt.ExpectOutput(lines...)

	return fbt
}
