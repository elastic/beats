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
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

func DefineModules() {
	// If MODULE is set in Buildkite pipeline step, skip variable further definition
	if os.Getenv("MODULE") != "" {
		return
	}

	if mg.Verbose() {
		fmt.Printf("Detecting changes in modules\n")
	}

	beatPath := os.Getenv("BEAT_PATH")

	var modulePattern = fmt.Sprintf("^%s\\/module\\/([^\\/]+)\\/.*", beatPath)

	moduleRegex, err := regexp.Compile(modulePattern)
	if err != nil {
		log.Fatal("failed to compile regex: " + err.Error())
	}

	modules := make(map[string]bool)
	for _, line := range getDiff() {
		if !isAsciiOrPng(line) {
			matches := moduleRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				modules[matches[1]] = true
			}
		}
	}

	keys := make([]string, len(modules))
	i := 0
	for k := range modules {
		keys[i] = k
		i++
	}

	moduleVar := strings.Join(keys, ",")

	if moduleVar != "" {
		err = os.Setenv("MODULE", moduleVar)
		if err != nil {
			return
		}

		_, _ = fmt.Fprintf(os.Stderr, "Detected changes in module(s): %s\n", moduleVar)
	}
}

func isAsciiOrPng(file string) bool {
	return strings.HasSuffix(file, ".asciidoc") || strings.HasSuffix(file, ".png")
}

func getDiff() []string {
	commitRange := getCommitRange()
	var output, _ = sh.Output("git", "diff", "--name-only", commitRange)

	if mg.Verbose() {
		_ = fmt.Sprintf("Git Diff result: %s\n", output)
	}

	return strings.Split(output, "\n")
}

func getFromCommit() string {
	baseBranch := os.Getenv("BUILDKITE_PULL_REQUEST_BASE_BRANCH")
	branch := os.Getenv("BUILDKITE_BRANCH")
	commit := os.Getenv("BUILDKITE_COMMIT")

	if baseBranch != "" {
		return fmt.Sprintf("origin/%s", baseBranch)
	}

	if branch != "" {
		return fmt.Sprintf("origin/%s", branch)
	}

	previousCommit := getPreviousCommit()
	if previousCommit != "" {
		return previousCommit
	}

	if mg.Verbose() {
		_ = fmt.Sprintf("Git from commit: %s", commit)
	}

	return commit
}

func getPreviousCommit() string {
	var output, _ = sh.Output("git", "rev-parse", "HEAD^")
	if mg.Verbose() {
		_ = fmt.Sprintf("Git previous commit: %s\n", output)
	}

	return strings.TrimSpace(output)
}

func getCommitRange() string {
	commit := os.Getenv("BUILDKITE_COMMIT")

	return fmt.Sprintf("%s...%s", getFromCommit(), commit)
}
