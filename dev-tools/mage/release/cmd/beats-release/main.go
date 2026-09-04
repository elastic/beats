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

// Command beats-release runs Beats release automation from a nested Go module
// so tooling dependencies stay out of the root module.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/dev-tools/mage/release"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "beats-release: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if err := chdirBeatsRoot(); err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("usage: beats-release <command> [args...]\n\n%s", usage())
	}

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "update-version":
		if len(rest) != 1 {
			return fmt.Errorf("usage: beats-release update-version <version>")
		}
		return release.UpdateVersion(rest[0])
	case "update-docs":
		if len(rest) != 1 {
			return fmt.Errorf("usage: beats-release update-docs <version>")
		}
		return release.UpdateDocs(rest[0])
	case "update-test-env":
		if len(rest) != 2 {
			return fmt.Errorf("usage: beats-release update-test-env <latest> <current>")
		}
		return release.UpdateTestEnv(rest[0], rest[1])
	case "update-mergify":
		if len(rest) != 1 {
			return fmt.Errorf("usage: beats-release update-mergify <version>")
		}
		return release.UpdateMergify(rest[0])
	case "run-major-minor":
		if len(rest) != 0 {
			return fmt.Errorf("usage: beats-release run-major-minor")
		}
		cfg, err := release.LoadConfigFromEnv()
		if err != nil {
			return err
		}
		return release.RunMajorMinorRelease(cfg)
	case "run-patch":
		if len(rest) != 0 {
			return fmt.Errorf("usage: beats-release run-patch")
		}
		cfg, err := release.LoadConfigFromEnv()
		if err != nil {
			return err
		}
		return release.RunPatchRelease(cfg)
	case "ensure-issue-tracker":
		if len(rest) != 0 {
			return fmt.Errorf("usage: beats-release ensure-issue-tracker")
		}
		cfg, err := release.LoadConfigFromEnv()
		if err != nil {
			return err
		}
		return release.EnsureReleaseIssueTracker(cfg, nil)
	case "help", "-h", "--help":
		fmt.Print(usage())
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage())
	}
}

func usage() string {
	return strings.TrimSpace(`
Commands:
  update-version <version>
  update-docs <version>
  update-test-env <latest> <current>
  update-mergify <version>
  run-major-minor
  run-patch
  ensure-issue-tracker

Environment for run-major-minor / run-patch / ensure-issue-tracker: see RELEASE.md and
dev-tools/mage/release/README.md (CURRENT_RELEASE, DRY_RUN, GITHUB_TOKEN, …).
`) + "\n"
}

// chdirBeatsRoot finds the Beats repository root and makes it the working
// directory. go run -C leaves cwd in the nested module, but release workflows
// expect to run from the repo root (OpenRepo("."), relative paths, make update).
func chdirBeatsRoot() error {
	if root := os.Getenv("BEATS_REPO_ROOT"); root != "" {
		return os.Chdir(root)
	}

	start, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	dir := start
	for {
		modPath := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err == nil && isBeatsRootModule(string(data)) {
			return os.Chdir(dir)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("beats repository root not found from %s (set BEATS_REPO_ROOT)", start)
		}
		dir = parent
	}
}

func isBeatsRootModule(goMod string) bool {
	for _, line := range strings.Split(goMod, "\n") {
		line = strings.TrimSpace(line)
		if line == "module github.com/elastic/beats/v7" {
			return true
		}
	}
	return false
}
