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
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ssh"
	"os"
	"path/filepath"
	"strings"
)

func linuxDiagnostics(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, destination string) error {
	// take ownership, as sudo tests will create with root permissions (allow to fail in the case it doesn't exist)
	diagnosticDir := "$HOME/agent/build/diagnostics"
	_, _, _ = sshClient.Exec(ctx, "sudo", []string{"chown", "-R", "$USER:$USER", diagnosticDir}, nil)
	stdOut, _, err := sshClient.Exec(ctx, "ls", []string{"-1", diagnosticDir}, nil)
	if err != nil {
		//nolint:nilerr // failed to list the directory, probably don't have any diagnostics (do nothing)
		return nil
	}
	eachDiagnostic := strings.Split(string(stdOut), "\n")
	for _, filename := range eachDiagnostic {
		filename = strings.TrimSpace(filename)
		if filename == "" {
			continue
		}

		// don't use filepath.Join as we need this to work in Windows as well
		// this is because if we use `filepath.Join` on a Windows host connected to a Linux host
		// it will use a `\` and that will be incorrect for Linux
		fp := fmt.Sprintf("%s/%s", diagnosticDir, filename)
		// use filepath.Join on this path because it's a path on this specific host platform
		dp := filepath.Join(destination, filename)
		logger.Logf("Copying diagnostic %s", filename)
		out, err := os.Create(dp)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", dp, err)
		}
		err = sshClient.GetFileContentsOutput(ctx, fp, out)
		_ = out.Close()
		if err != nil {
			return fmt.Errorf("failed to copy file from remote host to %s: %w", dp, err)
		}
	}
	return nil
}

func linuxCopy(ctx context.Context, sshClient ssh.SSHClient, logger common.Logger, repoArchive string, builds []common.Build) error {
	// copy the archive and extract it on the host
	logger.Logf("Copying repo")
	destRepoName := filepath.Base(repoArchive)
	err := sshClient.Copy(repoArchive, destRepoName)
	if err != nil {
		return fmt.Errorf("failed to SCP repo archive %s: %w", repoArchive, err)
	}

	// remove build paths, on cases where the build path is different from agent.
	for _, build := range builds {
		for _, remoteBuildPath := range []string{build.Path, build.SHA512Path} {
			relativeAgentDir := filepath.Join("agent", remoteBuildPath)
			_, _, err := sshClient.Exec(ctx, "sudo", []string{"rm", "-rf", relativeAgentDir}, nil)
			// doesn't need to be a fatal error.
			if err != nil {
				logger.Logf("error removing build dir %s: %w", relativeAgentDir, err)
			}
		}
	}

	// ensure that agent directory is removed (possible it already exists if instance already used)
	stdout, stderr, err := sshClient.Exec(ctx,
		"sudo", []string{"rm", "-rf", "agent"}, nil)
	if err != nil {
		return fmt.Errorf(
			"failed to remove agent directory before unziping new one: %w. stdout: %q, stderr: %q",
			err, stdout, stderr)
	}

	stdOut, errOut, err := sshClient.Exec(ctx, "unzip", []string{destRepoName, "-d", "agent"}, nil)
	if err != nil {
		return fmt.Errorf("failed to unzip %s to agent directory: %w (stdout: %s, stderr: %s)", destRepoName, err, stdOut, errOut)
	}

	// prepare for testing
	logger.Logf("Running make mage and prepareOnRemote")
	envs := `GOPATH="$HOME/go" PATH="$HOME/go/bin:$PATH"`
	installMage := strings.NewReader(fmt.Sprintf(`cd agent && %s make mage && %s mage integration:prepareOnRemote`, envs, envs))
	stdOut, errOut, err = sshClient.Exec(ctx, "bash", nil, installMage)
	if err != nil {
		return fmt.Errorf("failed to perform make mage and prepareOnRemote: %w (stdout: %s, stderr: %s)", err, stdOut, errOut)
	}

	// determine if the build needs to be replaced on the host
	// if it already exists and the SHA512 are the same contents, then
	// there is no reason to waste time uploading the build
	for _, build := range builds {
		copyBuild := true
		localSHA512, err := os.ReadFile(build.SHA512Path)
		if err != nil {
			return fmt.Errorf("failed to read local SHA52 contents %s: %w", build.SHA512Path, err)
		}
		hostSHA512Path := filepath.Base(build.SHA512Path)
		hostSHA512, err := sshClient.GetFileContents(ctx, hostSHA512Path)
		if err == nil {
			if string(localSHA512) == string(hostSHA512) {
				logger.Logf("Skipping copy agent build %s; already the same", filepath.Base(build.Path))
				copyBuild = false
			}
		}

		if copyBuild {
			// ensure the existing copies are removed first
			toRemove := filepath.Base(build.Path)
			stdOut, errOut, err = sshClient.Exec(ctx,
				"sudo", []string{"rm", "-f", toRemove}, nil)
			if err != nil {
				return fmt.Errorf("failed to remove %q: %w (stdout: %q, stderr: %q)",
					toRemove, err, stdOut, errOut)
			}

			toRemove = filepath.Base(build.SHA512Path)
			stdOut, errOut, err = sshClient.Exec(ctx,
				"sudo", []string{"rm", "-f", toRemove}, nil)
			if err != nil {
				return fmt.Errorf("failed to remove %q: %w (stdout: %q, stderr: %q)",
					toRemove, err, stdOut, errOut)
			}

			logger.Logf("Copying agent build %s", filepath.Base(build.Path))
		}

		for _, buildPath := range []string{build.Path, build.SHA512Path} {
			if copyBuild {
				err = sshClient.Copy(buildPath, filepath.Base(buildPath))
				if err != nil {
					return fmt.Errorf("failed to SCP build %s: %w", filepath.Base(buildPath), err)
				}
			}
			insideAgentDir := filepath.Join("agent", buildPath)
			stdOut, errOut, err = sshClient.Exec(ctx, "mkdir", []string{"-p", filepath.Dir(insideAgentDir)}, nil)
			if err != nil {
				return fmt.Errorf("failed to create %s directory: %w (stdout: %s, stderr: %s)", filepath.Dir(insideAgentDir), err, stdOut, errOut)
			}
			stdOut, errOut, err = sshClient.Exec(ctx, "ln", []string{filepath.Base(buildPath), insideAgentDir}, nil)
			if err != nil {
				return fmt.Errorf("failed to hard link %s to %s: %w (stdout: %s, stderr: %s)", filepath.Base(buildPath), insideAgentDir, err, stdOut, errOut)
			}
		}
	}

	return nil
}
