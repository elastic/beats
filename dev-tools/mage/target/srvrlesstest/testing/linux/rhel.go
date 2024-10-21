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

package linux

import (
	"context"
	"fmt"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ssh"
	"path"
	"strings"
	"time"
)

// RhelRunner is a handler for running tests on SUSE Linux Enterpriser Server
type RhelRunner struct{}

// Prepare configures the host for running the test
func (RhelRunner) Prepare(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, arch string, goVersion string) error {
	logger.Logf("Install development tools")
	dnfCtx, dnfCancel := context.WithTimeout(ctx, 20*time.Minute)
	defer dnfCancel()
	stdOut, errOut, err := sshClient.ExecWithRetry(dnfCtx, "sudo", []string{"dnf", "-y", "-v", "group", "install", "\"Development Tools\""}, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to run 'dnf group install \"Development Tools\"': %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}

	// install golang
	logger.Logf("Install golang %s (%s)", goVersion, arch)
	goCtx, goCancel := context.WithTimeout(ctx, 20*time.Minute)
	defer goCancel()
	downloadURL := fmt.Sprintf("https://go.dev/dl/go%s.linux-%s.tar.gz", goVersion, arch)
	filename := path.Base(downloadURL)
	stdOut, errOut, err = sshClient.Exec(goCtx, "curl", []string{"-Ls", downloadURL, "--output", filename}, nil)
	if err != nil {
		return fmt.Errorf("failed to download go from %s with curl: %w (stdout: %s, stderr: %s)", downloadURL, err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(goCtx, "sudo", []string{"tar", "-C", "/usr/local", "-xzf", filename}, nil)
	if err != nil {
		return fmt.Errorf("failed to extract go to /usr/local with tar: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(goCtx, "sudo", []string{"ln", "-s", "/usr/local/go/bin/go", "/usr/bin/go"}, nil)
	if err != nil {
		return fmt.Errorf("failed to symlink /usr/local/go/bin/go to /usr/bin/go: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(goCtx, "sudo", []string{"ln", "-s", "/usr/local/go/bin/gofmt", "/usr/bin/gofmt"}, nil)
	if err != nil {
		return fmt.Errorf("failed to symlink /usr/local/go/bin/gofmt to /usr/bin/gofmt: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}

	return nil
}

// Copy places the required files on the host
func (RhelRunner) Copy(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, repoArchive string, builds []common.Build) error {
	return linuxCopy(ctx, sshClient, logger, repoArchive, builds)
}

// Run the test
func (RhelRunner) Run(ctx context.Context, verbose bool, sshClient ssh.SSHClient, logger common.Logger, agentVersion string, prefix string, batch define.Batch, env map[string]string) (common.OSRunnerResult, error) {
	var tests []string
	for _, pkg := range batch.Tests {
		for _, test := range pkg.Tests {
			tests = append(tests, fmt.Sprintf("%s:%s", pkg.Name, test.Name))
		}
	}
	var sudoTests []string
	for _, pkg := range batch.SudoTests {
		for _, test := range pkg.Tests {
			sudoTests = append(sudoTests, fmt.Sprintf("%s:%s", pkg.Name, test.Name))
		}
	}

	logArg := ""
	if verbose {
		logArg = "-v"
	}
	var result common.OSRunnerResult
	if len(tests) > 0 {
		vars := fmt.Sprintf(`GOPATH="$HOME/go" PATH="$HOME/go/bin:$PATH" AGENT_VERSION="%s" TEST_DEFINE_PREFIX="%s" TEST_DEFINE_TESTS="%s"`, agentVersion, prefix, strings.Join(tests, ","))
		vars = extendVars(vars, env)

		script := fmt.Sprintf(`cd agent && %s ~/go/bin/mage %s integration:testOnRemote`, vars, logArg)
		results, err := runTests(ctx, logger, "non-sudo", prefix, script, sshClient, batch.Tests)
		if err != nil {
			return common.OSRunnerResult{}, fmt.Errorf("error running non-sudo tests: %w", err)
		}
		result.Packages = results
	}

	if len(sudoTests) > 0 {
		prefix := fmt.Sprintf("%s-sudo", prefix)
		vars := fmt.Sprintf(`GOPATH="$HOME/go" PATH="$HOME/go/bin:$PATH:/usr/sbin" AGENT_VERSION="%s" TEST_DEFINE_PREFIX="%s" TEST_DEFINE_TESTS="%s"`, agentVersion, prefix, strings.Join(sudoTests, ","))
		vars = extendVars(vars, env)
		script := fmt.Sprintf(`cd agent && sudo %s ~/go/bin/mage %s integration:testOnRemote`, vars, logArg)

		results, err := runTests(ctx, logger, "sudo", prefix, script, sshClient, batch.SudoTests)
		if err != nil {
			return common.OSRunnerResult{}, fmt.Errorf("error running sudo tests: %w", err)
		}
		result.SudoPackages = results
	}

	return result, nil
}

// Diagnostics gathers any diagnostics from the host.
func (RhelRunner) Diagnostics(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, destination string) error {
	return linuxDiagnostics(ctx, sshClient, logger, destination)
}
