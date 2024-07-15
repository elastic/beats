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
	"os/exec"
	"regexp"
	"strings"
)

func DefineModules() {
	beatPath := os.Getenv("BEAT_PATH")
	if beatPath == "" {
		fmt.Errorf("argument required: beatPath")
		os.Exit(1)
	}

	var modulePattern string
	//if strings.Contains(beatPath, "x-pack") {
	//	modulePattern = "^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
	//} else {
	//	modulePattern = "^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
	//}
	modulePattern = fmt.Sprintf("^%s\\/module\\/([^\\/]+)\\/.*", beatPath)

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

	err = os.Setenv("MODULE", strings.Join(keys, ","))
}

func isAsciiOrPng(file string) bool {
	return strings.HasSuffix(file, ".asciidoc") || strings.HasSuffix(file, ".png")
}

func getDiff() []string {
	commitRange := getCommitRange()
	cmd := exec.Command("git", "diff", "--name-only", commitRange)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Failed to execute 'git diff --name-only %s': %s", commitRange, err)
		os.Exit(1)
	}

	return strings.Split(string(output), "\n")
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

	return commit
}

func getPreviousCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD^")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Failed to execute 'git rev-parse HEAD^': ", err)
		os.Exit(1)
	}

	return strings.TrimSpace(string(output))
}

func getCommitRange() string {
	commit := os.Getenv("BUILDKITE_COMMIT")

	return fmt.Sprintf("%s...%s", getFromCommit(), commit)
}
