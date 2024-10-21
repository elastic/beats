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

package common

import (
	"context"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ssh"
)

// OSRunnerPackageResult is the result for each package.
type OSRunnerPackageResult struct {
	// Name is the package name.
	Name string
	// Output is the raw test output.
	Output []byte
	// XMLOutput is the XML Junit output.
	XMLOutput []byte
	// JSONOutput is the JSON output.
	JSONOutput []byte
}

// OSRunnerResult is the result of the test run provided by a OSRunner.
type OSRunnerResult struct {
	// Packages is the results for each package.
	Packages []OSRunnerPackageResult

	// SudoPackages is the results for each package that need to run as sudo.
	SudoPackages []OSRunnerPackageResult
}

// OSRunner provides an interface to run the tests on the OS.
type OSRunner interface {
	// Prepare prepares the runner to actual run on the host.
	Prepare(ctx context.Context, sshClient ssh.SSHClient, logger Logger, arch string, goVersion string) error
	// Copy places the required files on the host.
	Copy(ctx context.Context, sshClient ssh.SSHClient, logger Logger, repoArchive string, builds []Build) error
	// Run runs the actual tests and provides the result.
	Run(ctx context.Context, verbose bool, sshClient ssh.SSHClient, logger Logger, agentVersion string, prefix string, batch define.Batch, env map[string]string) (OSRunnerResult, error)
	// Diagnostics gathers any diagnostics from the host.
	Diagnostics(ctx context.Context, sshClient ssh.SSHClient, logger Logger, destination string) error
}
