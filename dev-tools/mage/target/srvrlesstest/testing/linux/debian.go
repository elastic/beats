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
	"path/filepath"
	"strings"
	"time"
)

// DebianRunner is a handler for running tests on Linux
type DebianRunner struct{}

// Prepare the test
func (DebianRunner) Prepare(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, arch string, goVersion string) error {
	// prepare build-essential and unzip
	//
	// apt-get update and install are so terrible that we have to place this in a loop, because in some cases the
	// apt-get update says it works, but it actually fails. so we add 3 tries here
	var err error
	for i := 0; i < 3; i++ {
		err = func() error {
			updateCtx, updateCancel := context.WithTimeout(ctx, 3*time.Minute)
			defer updateCancel()
			logger.Logf("Running apt-get update")
			// `-o APT::Update::Error-Mode=any` ensures that any warning is tried as an error, so the retry
			// will occur (without this we get random failures)
			stdOut, errOut, err := sshClient.ExecWithRetry(updateCtx, "sudo", []string{"apt-get", "update", "-o APT::Update::Error-Mode=any"}, 15*time.Second)
			if err != nil {
				return fmt.Errorf("failed to run apt-get update: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
			}
			return func() error {
				// golang is installed below and not using the package manager, ensures that the exact version
				// of golang is used for the running of the test
				installCtx, installCancel := context.WithTimeout(ctx, 1*time.Minute)
				defer installCancel()
				logger.Logf("Install build-essential and unzip")
				stdOut, errOut, err = sshClient.ExecWithRetry(installCtx, "sudo", []string{"apt-get", "install", "-y", "build-essential", "unzip"}, 5*time.Second)
				if err != nil {
					return fmt.Errorf("failed to install build-essential and unzip: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
				}
				return nil
			}()
		}()
		if err == nil {
			// installation was successful
			break
		}
		logger.Logf("Failed to install build-essential and unzip; will wait 15 seconds and try again")
		<-time.After(15 * time.Second)
	}
	if err != nil {
		// seems after 3 tries it still failed
		return err
	}

	// prepare golang
	logger.Logf("Install golang %s (%s)", goVersion, arch)
	downloadURL := fmt.Sprintf("https://go.dev/dl/go%s.linux-%s.tar.gz", goVersion, arch)
	filename := path.Base(downloadURL)
	stdOut, errOut, err := sshClient.Exec(ctx, "curl", []string{"-Ls", downloadURL, "--output", filename}, nil)
	if err != nil {
		return fmt.Errorf("failed to download go from %s with curl: %w (stdout: %s, stderr: %s)", downloadURL, err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(ctx, "sudo", []string{"tar", "-C", "/usr/local", "-xzf", filename}, nil)
	if err != nil {
		return fmt.Errorf("failed to extract go to /usr/local with tar: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(ctx, "sudo", []string{"ln", "-s", "/usr/local/go/bin/go", "/usr/bin/go"}, nil)
	if err != nil {
		return fmt.Errorf("failed to symlink /usr/local/go/bin/go to /usr/bin/go: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}
	stdOut, errOut, err = sshClient.Exec(ctx, "sudo", []string{"ln", "-s", "/usr/local/go/bin/gofmt", "/usr/bin/gofmt"}, nil)
	if err != nil {
		return fmt.Errorf("failed to symlink /usr/local/go/bin/gofmt to /usr/bin/gofmt: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}

	return nil
}

// Copy places the required files on the host.
func (DebianRunner) Copy(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, repoArchive string, builds []common.Build) error {
	return linuxCopy(ctx, sshClient, logger, repoArchive, builds)
}

// Run the test
func (DebianRunner) Run(ctx context.Context, verbose bool, sshClient ssh.SSHClient, logger common.Logger, agentVersion string, prefix string, batch define.Batch, env map[string]string) (common.OSRunnerResult, error) {
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
		vars := fmt.Sprintf(`GOPATH="$HOME/go" PATH="$HOME/go/bin:$PATH" AGENT_VERSION="%s" TEST_DEFINE_PREFIX="%s" TEST_DEFINE_TESTS="%s"`, agentVersion, prefix, strings.Join(sudoTests, ","))
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
func (DebianRunner) Diagnostics(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, destination string) error {
	return linuxDiagnostics(ctx, sshClient, logger, destination)
}

func runTests(ctx context.Context, logger common.Logger, name string, prefix string, script string, sshClient ssh.SSHClient, tests []define.BatchPackageTests) ([]common.OSRunnerPackageResult, error) {
	execTest := strings.NewReader(script)

	session, err := sshClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	session.Stdout = common.NewPrefixOutput(logger, fmt.Sprintf("Test output (%s) (stdout): ", name))
	session.Stderr = common.NewPrefixOutput(logger, fmt.Sprintf("Test output (%s) (stderr): ", name))
	session.Stdin = execTest

	// allowed to fail because tests might fail
	logger.Logf("Running %s tests...", name)
	err = session.Run("bash")
	if err != nil {
		logger.Logf("%s tests failed: %s", name, err)
	}
	// this seems to always return an error
	_ = session.Close()

	var result []common.OSRunnerPackageResult
	// fetch the contents for each package
	for _, pkg := range tests {
		resultPkg, err := getRunnerPackageResult(ctx, sshClient, pkg, prefix)
		if err != nil {
			return nil, err
		}
		result = append(result, resultPkg)
	}
	return result, nil
}

func getRunnerPackageResult(ctx context.Context, sshClient ssh.SSHClient, pkg define.BatchPackageTests, prefix string) (common.OSRunnerPackageResult, error) {
	var err error
	var resultPkg common.OSRunnerPackageResult
	resultPkg.Name = pkg.Name
	outputPath := fmt.Sprintf("$HOME/agent/build/TEST-go-remote-%s.%s", prefix, filepath.Base(pkg.Name))
	resultPkg.Output, err = sshClient.GetFileContents(ctx, outputPath+".out")
	if err != nil {
		return common.OSRunnerPackageResult{}, fmt.Errorf("failed to fetched test output at %s.out", outputPath)
	}
	resultPkg.JSONOutput, err = sshClient.GetFileContents(ctx, outputPath+".out.json")
	if err != nil {
		return common.OSRunnerPackageResult{}, fmt.Errorf("failed to fetched test output at %s.out.json", outputPath)
	}
	resultPkg.XMLOutput, err = sshClient.GetFileContents(ctx, outputPath+".xml")
	if err != nil {
		return common.OSRunnerPackageResult{}, fmt.Errorf("failed to fetched test output at %s.xml", outputPath)
	}
	return resultPkg, nil
}

func extendVars(vars string, env map[string]string) string {
	var envStr []string
	for k, v := range env {
		envStr = append(envStr, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return fmt.Sprintf("%s %s", vars, strings.Join(envStr, " "))
}
