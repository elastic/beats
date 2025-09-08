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

// DefineModules checks which modules were changed and populates MODULE environment variable,
// so that CI would run tests only for the changed ones.
// If no modules were changed MODULE variable won't be defined.
func DefineModules() {
	// If MODULE is set in Buildkite pipeline step, skip variable further definition
	if os.Getenv("MODULE") != "" {
		return
	}

	if mg.Verbose() {
		fmt.Printf("Detecting changes in modules\n")
	}

	beatPath := os.Getenv("BEAT_PATH")
	if beatPath == "" {
		log.Fatal("BEAT_PATH is not defined")
	}

	var modulePattern = fmt.Sprintf("^%s\\/module\\/([^\\/]+)\\/.*", beatPath)

	moduleRegex, err := regexp.Compile(modulePattern)
	if err != nil {
		log.Fatal("failed to compile regex: " + err.Error())
	}

	modules := map[string]struct{}{}
	for _, line := range getDiff() {
		if !shouldIgnore(line) {
			matches := moduleRegex.FindStringSubmatch(line)
			if len(matches) > 0 {
				modules[matches[1]] = struct{}{}
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

		log.Printf("Detected changes in module(s): %s\n", moduleVar)
	} else {
		log.Printf("No changed modules found")
	}
}

func shouldIgnore(file string) bool {
	ignoreList := []string{".asciidoc", ".png"}
	for ext := range ignoreList {
		if strings.HasSuffix(file, ignoreList[ext]) {
			return true
		}
	}

	// if the file has been removed, we should ignore it
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return true
	}

	return false
}

func getDiff() []string {
	commitRange := getCommitRange()
	var output, err = sh.Output("git", "diff", "--name-only", commitRange)
	if err != nil {
		log.Fatal("git Diff failed: %w", err)
	}

	printWhenVerbose("Git Diff result: %s\n", output)

	return strings.Split(output, "\n")
}

func getFromCommit() string {
	if baseBranch := os.Getenv("BUILDKITE_PULL_REQUEST_BASE_BRANCH"); baseBranch != "" {
		printWhenVerbose("PR branch: %s\n", baseBranch)

		return getBranchName(baseBranch)
	}

	if previousCommit := getPreviousCommit(); previousCommit != "" {
		printWhenVerbose("Git from commit: %s\n", previousCommit)

		return previousCommit
	} else {
		commit, err := getBuildkiteCommit()
		if err != nil {
			log.Fatal(err)
		}
		printWhenVerbose("Git from commit: %s\n", commit)

		return commit
	}
}

func getPreviousCommit() string {
	var output, _ = sh.Output("git", "rev-parse", "HEAD^")
	printWhenVerbose("Git previous commit: %s\n", output)

	return strings.TrimSpace(output)
}

func getCommitRange() string {
	commit, err := getBuildkiteCommit()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s...%s", getFromCommit(), commit)
}

func getBranchName(branch string) string {
	return fmt.Sprintf("origin/%s", branch)
}

func printWhenVerbose(template string, parameter string) {
	if mg.Verbose() {
		fmt.Printf(template, parameter)
	}
}

func getBuildkiteCommit() (string, error) {
	commit := os.Getenv("BUILDKITE_COMMIT")
	if commit == "" {
		log.Fatal("BUILDKITE_COMMIT is not defined")
	}

	return commit, nil
}
