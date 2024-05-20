package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Argument required: beatPath")
		os.Exit(1)
	}

	beatPath := os.Args[1]

	var modulePattern string
	if strings.Contains(beatPath, "x-pack") {
		modulePattern = "^x-pack\\/[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
	} else {
		modulePattern = "^[a-z0-9]+beat\\/module\\/([^\\/]+)\\/.*"
	}

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

	fmt.Println(strings.Join(keys, ","))
}

func isAsciiOrPng(file string) bool {
	return strings.HasSuffix(file, ".asciidoc") || strings.HasSuffix(file, ".png")
}

func getDiff() []string {
	cmd := exec.Command("git", "diff", "--name-only", getCommitRange())
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error: %s", err)
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
		fmt.Println("Error: ", err)
		os.Exit(1)
	}

	return string(output)
}

func getCommitRange() string {
	commit := os.Getenv("BUILDKITE_COMMIT")
	return fmt.Sprintf("%s...%s", getFromCommit(), commit)
}
