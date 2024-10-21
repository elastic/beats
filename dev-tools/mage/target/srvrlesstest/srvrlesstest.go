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

package srvrlesstest

import (
	"context"
	"fmt"
	tcommon "github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/common"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/define"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ess"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/kubernetes/kind"
	multipass "github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/multipas"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/ogc"
	"github.com/elastic/beats/v7/dev-tools/mage/target/srvrlesstest/testing/runner"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/magefile/mage/mg"
)

type ProvisionerType uint32

var (
	goIntegTestTimeout        = 2 * time.Hour
	goProvisionAndTestTimeout = goIntegTestTimeout + 30*time.Minute
)

const (
	snapshotEnv = "SNAPSHOT"
)

// Integration namespace contains tasks related to operating and running integration tests.
type Integration mg.Namespace

func IntegRunner(ctx context.Context, matrix bool, singleTest string) error {
	if _, ok := ctx.Deadline(); !ok {
		// If the context doesn't have a timeout (usually via the mage -t option), give it one.
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, goProvisionAndTestTimeout)
		defer cancel()
	}

	for {
		failedCount, err := integRunnerOnce(ctx, matrix, singleTest)
		if err != nil {
			return err
		}
		if failedCount > 0 {
			if hasCleanOnExit() {
				mg.Deps(Integration.Clean)
			}
			os.Exit(1)
		}
		if !hasRunUntilFailure() {
			if hasCleanOnExit() {
				mg.Deps(Integration.Clean)
			}
			return nil
		}
	}
}

func hasCleanOnExit() bool {
	clean := os.Getenv("TEST_INTEG_CLEAN_ON_EXIT")
	b, _ := strconv.ParseBool(clean)
	return b
}

func hasRunUntilFailure() bool {
	runUntil := os.Getenv("TEST_RUN_UNTIL_FAILURE")
	b, _ := strconv.ParseBool(runUntil)
	return b
}

func integRunnerOnce(ctx context.Context, matrix bool, singleTest string) (int, error) {
	goTestFlags := os.Getenv("GOTEST_FLAGS")

	batches, err := define.DetermineBatches("testing/integration", goTestFlags, "integration")
	if err != nil {
		return 0, fmt.Errorf("failed to determine batches: %w", err)
	}
	r, err := createTestRunner(matrix, singleTest, goTestFlags, batches...)
	if err != nil {
		return 0, fmt.Errorf("error creating test runner: %w", err)
	}
	results, err := r.Run(ctx)
	if err != nil {
		return 0, fmt.Errorf("error running test: %w", err)
	}
	_ = os.Remove("build/TEST-go-integration.out")
	_ = os.Remove("build/TEST-go-integration.out.json")
	_ = os.Remove("build/TEST-go-integration.xml")
	err = writeFile("build/TEST-go-integration.out", results.Output, 0644)
	if err != nil {
		return 0, fmt.Errorf("error writing test out file: %w", err)
	}
	err = writeFile("build/TEST-go-integration.out.json", results.JSONOutput, 0644)
	if err != nil {
		return 0, fmt.Errorf("error writing test out json file: %w", err)
	}
	err = writeFile("build/TEST-go-integration.xml", results.XMLOutput, 0644)
	if err != nil {
		return 0, fmt.Errorf("error writing test out xml file: %w", err)
	}
	if results.Failures > 0 {
		r.Logger().Logf("Testing completed (%d failures, %d successful)", results.Failures, results.Tests-results.Failures)
	} else {
		r.Logger().Logf("Testing completed (%d successful)", results.Tests)
	}
	r.Logger().Logf("Console output written here: build/TEST-go-integration.out")
	r.Logger().Logf("Console JSON output written here: build/TEST-go-integration.out.json")
	r.Logger().Logf("JUnit XML written here: build/TEST-go-integration.xml")
	r.Logger().Logf("Diagnostic output (if present) here: build/diagnostics")
	return results.Failures, nil
}

// Clean cleans up the integration testing leftovers
func (Integration) Clean() error {
	fmt.Println("--- Clean mage artifacts")
	_ = os.RemoveAll(".agent-testing")

	// Clean out .integration-cache/.ogc-cache always
	defer os.RemoveAll(".integration-cache")
	defer os.RemoveAll(".ogc-cache")

	_, err := os.Stat(".integration-cache")
	if err == nil {
		// .integration-cache exists; need to run `Clean` from the runner
		r, err := createTestRunner(false, "", "")
		if err != nil {
			return fmt.Errorf("error creating test runner: %w", err)
		}
		err = r.Clean()
		if err != nil {
			return fmt.Errorf("error running clean: %w", err)
		}
	}

	return nil
}

func createTestRunner(matrix bool, singleTest string, goTestFlags string, batches ...define.Batch) (*runner.Runner, error) {
	goVersion, err := mage.DefaultBeatBuildVariableSources.GetGoVersion()
	if err != nil {
		return nil, err
	}

	agentVersion, agentStackVersion, err := getTestRunnerVersions()
	if err != nil {
		return nil, err
	}

	agentBuildDir := os.Getenv("AGENT_BUILD_DIR")
	if agentBuildDir == "" {
		agentBuildDir = filepath.Join("build", "distributions")
	}
	essToken, ok, err := ess.GetESSAPIKey()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("ESS api key missing; run 'mage integration:auth'")
	}

	// Possible to change the region for deployment, default is gcp-us-west2 which is
	// the CFT region.
	essRegion := os.Getenv("TEST_INTEG_AUTH_ESS_REGION")
	if essRegion == "" {
		essRegion = "gcp-us-west2"
	}

	serviceTokenPath, ok, err := getGCEServiceTokenPath()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("GCE service token missing; run 'mage integration:auth'")
	}
	datacenter := os.Getenv("TEST_INTEG_AUTH_GCP_DATACENTER")
	if datacenter == "" {
		// us-central1-a is used because T2A instances required for ARM64 testing are only
		// available in the central regions
		datacenter = "us-central1-a"
	}

	ogcCfg := ogc.Config{
		ServiceTokenPath: serviceTokenPath,
		Datacenter:       datacenter,
	}

	var instanceProvisioner tcommon.InstanceProvisioner
	instanceProvisionerMode := os.Getenv("INSTANCE_PROVISIONER")
	switch instanceProvisionerMode {
	case "", ogc.Name:
		instanceProvisionerMode = ogc.Name
		instanceProvisioner, err = ogc.NewProvisioner(ogcCfg)
	case multipass.Name:
		instanceProvisioner = multipass.NewProvisioner()
	case kind.Name:
		instanceProvisioner = kind.NewProvisioner()
	default:
		return nil, fmt.Errorf("INSTANCE_PROVISIONER environment variable must be one of 'ogc' or 'multipass', not %s", instanceProvisionerMode)
	}

	email, err := ogcCfg.ClientEmail()
	if err != nil {
		return nil, err
	}

	provisionCfg := ess.ProvisionerConfig{
		Identifier: fmt.Sprintf("at-%s", strings.Replace(strings.Split(email, "@")[0], ".", "-", -1)),
		APIKey:     essToken,
		Region:     essRegion,
	}

	var stackProvisioner tcommon.StackProvisioner
	stackProvisionerMode := os.Getenv("STACK_PROVISIONER")
	switch stackProvisionerMode {
	case "", ess.ProvisionerStateful:
		stackProvisionerMode = ess.ProvisionerStateful
		stackProvisioner, err = ess.NewProvisioner(provisionCfg)
		if err != nil {
			return nil, err
		}
	case ess.ProvisionerServerless:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		stackProvisioner, err = ess.NewServerlessProvisioner(ctx, provisionCfg)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("STACK_PROVISIONER environment variable must be one of %q or %q, not %s",
			ess.ProvisionerStateful,
			ess.ProvisionerServerless,
			stackProvisionerMode)
	}

	timestamp := timestampEnabled()

	extraEnv := map[string]string{}
	if agentCollectDiag := os.Getenv("AGENT_COLLECT_DIAG"); agentCollectDiag != "" {
		extraEnv["AGENT_COLLECT_DIAG"] = agentCollectDiag
	}
	if agentKeepInstalled := os.Getenv("AGENT_KEEP_INSTALLED"); agentKeepInstalled != "" {
		extraEnv["AGENT_KEEP_INSTALLED"] = agentKeepInstalled
	}

	extraEnv["TEST_LONG_RUNNING"] = os.Getenv("TEST_LONG_RUNNING")
	extraEnv["LONG_TEST_RUNTIME"] = os.Getenv("LONG_TEST_RUNTIME")

	// these following two env vars are currently not used by anything, but can be used in the future to test beats or
	// other binaries, see https://github.com/elastic/elastic-agent/pull/3258
	binaryName := os.Getenv("TEST_BINARY_NAME")
	if binaryName == "" {
		binaryName = "elastic-agent"
	}

	repoDir := os.Getenv("TEST_INTEG_REPO_PATH")
	if repoDir == "" {
		repoDir = "."
	}

	diagDir := filepath.Join("build", "diagnostics")
	_ = os.MkdirAll(diagDir, 0755)

	cfg := tcommon.Config{
		AgentVersion:   agentVersion,
		StackVersion:   agentStackVersion,
		BuildDir:       agentBuildDir,
		GOVersion:      goVersion,
		RepoDir:        repoDir,
		DiagnosticsDir: diagDir,
		StateDir:       ".integration-cache",
		Platforms:      testPlatforms(),
		Packages:       testPackages(),
		Groups:         testGroups(),
		Matrix:         matrix,
		SingleTest:     singleTest,
		VerboseMode:    mg.Verbose(),
		Timestamp:      timestamp,
		TestFlags:      goTestFlags,
		ExtraEnv:       extraEnv,
		BinaryName:     binaryName,
	}

	r, err := runner.NewRunner(cfg, instanceProvisioner, stackProvisioner, batches...)
	if err != nil {
		return nil, fmt.Errorf("failed to create runner: %w", err)
	}
	return r, nil
}

func writeFile(name string, data []byte, perm os.FileMode) error {
	err := os.WriteFile(name, data, perm)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", name, err)
	}
	return nil
}

func getTestRunnerVersions() (string, string, error) {
	var err error
	agentStackVersion := os.Getenv("AGENT_STACK_VERSION")
	agentVersion := os.Getenv("AGENT_VERSION")
	if agentVersion == "" {
		agentVersion, err = mage.DefaultBeatBuildVariableSources.GetBeatVersion()
		if err != nil {
			return "", "", err
		}
		if agentStackVersion == "" {
			// always use snapshot for stack version
			agentStackVersion = fmt.Sprintf("%s-SNAPSHOT", agentVersion)
		}
		if hasSnapshotEnv() {
			// in the case that SNAPSHOT=true is set in the environment the
			// default version of the agent is used, but as a snapshot build
			agentVersion = fmt.Sprintf("%s-SNAPSHOT", agentVersion)
		}
	}

	if agentStackVersion == "" {
		agentStackVersion = agentVersion
	}

	return agentVersion, agentStackVersion, nil
}

func hasSnapshotEnv() bool {
	snapshot := os.Getenv(snapshotEnv)
	if snapshot == "" {
		return false
	}
	b, _ := strconv.ParseBool(snapshot)

	return b
}

func getGCEServiceTokenPath() (string, bool, error) {
	serviceTokenPath := os.Getenv("TEST_INTEG_AUTH_GCP_SERVICE_TOKEN_FILE")
	if serviceTokenPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", false, fmt.Errorf("unable to determine user's home directory: %w", err)
		}
		serviceTokenPath = filepath.Join(homeDir, ".config", "gcloud", "agent-testing-service-token.json")
	}
	_, err := os.Stat(serviceTokenPath)
	if os.IsNotExist(err) {
		return serviceTokenPath, false, nil
	} else if err != nil {
		return serviceTokenPath, false, fmt.Errorf("unable to check for service account key file at %s: %w", serviceTokenPath, err)
	}
	return serviceTokenPath, true, nil
}

func timestampEnabled() bool {
	timestamp := os.Getenv("TEST_INTEG_TIMESTAMP")
	if timestamp == "" {
		return false
	}
	b, _ := strconv.ParseBool(timestamp)
	return b
}

func testPlatforms() []string {
	platformsStr := os.Getenv("TEST_PLATFORMS")
	if platformsStr == "" {
		return nil
	}
	var platforms []string
	for _, p := range strings.Split(platformsStr, " ") {
		if p != "" {
			platforms = append(platforms, p)
		}
	}
	return platforms
}

func testPackages() []string {
	packagesStr, defined := os.LookupEnv("TEST_PACKAGES")
	if !defined {
		return nil
	}

	var packages []string
	for _, p := range strings.Split(packagesStr, ",") {
		if p == "tar.gz" {
			p = "targz"
		}
		packages = append(packages, p)
	}

	return packages
}

func testGroups() []string {
	groupsStr := os.Getenv("TEST_GROUPS")
	if groupsStr == "" {
		return nil
	}
	var groups []string
	for _, g := range strings.Split(groupsStr, " ") {
		if g != "" {
			groups = append(groups, g)
		}
	}
	return groups
}
