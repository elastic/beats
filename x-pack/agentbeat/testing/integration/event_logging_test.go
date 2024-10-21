// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
	"github.com/elastic/elastic-agent/pkg/testing/tools/testcontext"
)

var eventLogConfig = `
outputs:
  default:
    type: elasticsearch
    hosts:
      - %s
    protocol: http
    preset: balanced

inputs:
  - type: filestream
    id: your-input-id
    streams:
      - id: your-filestream-stream-id
        data_stream:
          dataset: generic
        paths:
          - %s

# Disable monitoring so there are less Beats running and less logs being generated.
agent.monitoring:
  enabled: false
  logs: false
  metrics: false
  pprof.enabled: false
  use_output: default

# Needed if you already have an Elastic-Agent running on your machine
# That's very helpful for running the tests locally
agent.monitoring:
  http:
    enabled: false
    port: 7002
agent.grpc:
  address: localhost
  port: 7001
`

func TestEventLogFile(t *testing.T) {
	_ = define.Require(t, define.Requirements{
		Group: Default,
		Stack: &define.Stack{},
		Local: true,
		Sudo:  false,
	})
	ctx, cancel := testcontext.WithDeadline(
		t,
		context.Background(),
		time.Now().Add(10*time.Minute))
	defer cancel()

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	esURL := startMockES(t)

	logFilepath := path.Join(t.TempDir(), t.Name())
	generateLogFile(t, logFilepath, time.Millisecond*100, 1)

	cfg := fmt.Sprintf(eventLogConfig, esURL, logFilepath)

	if err := agentFixture.Prepare(ctx); err != nil {
		t.Fatalf("cannot prepare Elastic-Agent fixture: %s", err)
	}

	if err := agentFixture.Configure(ctx, []byte(cfg)); err != nil {
		t.Fatalf("cannot configure Elastic-Agent fixture: %s", err)
	}

	cmd, err := agentFixture.PrepareAgentCommand(ctx, nil)
	if err != nil {
		t.Fatalf("cannot prepare Elastic-Agent command: %s", err)
	}

	output := strings.Builder{}
	cmd.Stderr = &output
	cmd.Stdout = &output

	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start Elastic-Agent: %s", err)
	}

	// Make sure the Elastic-Agent process is not running before
	// exiting the test
	t.Cleanup(func() {
		// Ignore the error because we cancelled the context,
		// and that always returns an error
		_ = cmd.Wait()
		if t.Failed() {
			t.Log("Elastic-Agent output:")
			t.Log(output.String())
		}
	})

	// Now the Elastic-Agent is running, so validate the Event log file.
	requireEventLogFileExistsWithData(t, agentFixture)

	// The diagnostics command is already tested by another test,
	// here we just want to validate the events log behaviour
	// extract the zip file into a temp folder
	expectedLogFiles, expectedEventLogFiles := getLogFilenames(
		t,
		filepath.Join(agentFixture.WorkDir(),
			"data",
			"elastic-agent-*",
			"logs"))

	collectDiagnosticsAndVeriflyLogs(
		t,
		ctx,
		agentFixture,
		[]string{"diagnostics", "collect"},
		append(expectedLogFiles, expectedEventLogFiles...))

	collectDiagnosticsAndVeriflyLogs(
		t,
		ctx,
		agentFixture,
		[]string{"diagnostics", "collect", "--exclude-events"},
		expectedLogFiles)
}

func TestEventLogOutputConfiguredViaFleet(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Group: "container",
	})
	t.Skip("Flaky test: https://github.com/elastic/elastic-agent/issues/5159")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	_, outputID := createMockESOutput(t, info)
	policyName := fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String())
	policyID, enrollmentAPIKey := createPolicy(
		t,
		ctx,
		agentFixture,
		info,
		policyName,
		outputID)

	fleetURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	if err != nil {
		t.Fatalf("could not get Fleet URL: %s", err)
	}

	enrollArgs := []string{
		"enroll",
		"--force",
		"--skip-daemon-reload",
		"--url",
		fleetURL,
		"--enrollment-token",
		enrollmentAPIKey,
	}

	addLogIntegration(t, info, policyID, "/tmp/flog.log")
	generateLogFile(t, "/tmp/flog.log", time.Second/2, 100)

	enrollCmd, err := agentFixture.PrepareAgentCommand(ctx, enrollArgs)
	if err != nil {
		t.Fatalf("could not prepare enroll command: %s", err)
	}
	if out, err := enrollCmd.CombinedOutput(); err != nil {
		t.Fatalf("error enrolling Elastic-Agent: %s\nOutput:\n%s", err, string(out))
	}

	runAgentCMD, agentOutput := prepareAgentCMD(t, ctx, agentFixture, nil, nil)
	if err := runAgentCMD.Start(); err != nil {
		t.Fatalf("could not start Elastic-Agent: %s", err)
	}

	assert.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err := agentFixture.IsHealthy(ctx)
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\", Agent logs\n%s",
		err, agentOutput,
	)

	// The default behaviour is to log events to the events log file
	// so ensure this is happening
	requireEventLogFileExistsWithData(t, agentFixture)

	// Add a policy overwrite to change the events output to stderr
	addOverwriteToPolicy(t, info, policyName, policyID)

	// Ensure Elastic-Agent is healthy after the policy change
	assert.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err := agentFixture.IsHealthy(ctx)
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy after policy change. Agent status error: \"%v\", Agent logs\n%s",
		err, agentOutput,
	)

	// Ensure the events logs are going to stderr
	assert.Eventually(t, func() bool {
		agentOutputStr := agentOutput.String()
		scanner := bufio.NewScanner(strings.NewReader(agentOutputStr))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "Cannot index event") {
				return true
			}
		}

		return false
	}, 3*time.Minute, 10*time.Second, "cannot find events on stderr")

}

func addOverwriteToPolicy(t *testing.T, info *define.Info, policyName, policyID string) {
	addLoggingOverwriteBody := fmt.Sprintf(`
{
  "name": "%s",
  "namespace": "default",
  "overrides": {
    "agent": {
      "logging": {
        "event_data": {
          "to_stderr": true,
          "to_files": false
        }
      }
    }
  }
}
`, policyName)
	resp, err := info.KibanaClient.Send(
		http.MethodPut,
		fmt.Sprintf("/api/fleet/agent_policies/%s", policyID),
		nil,
		nil,
		bytes.NewBufferString(addLoggingOverwriteBody),
	)
	if err != nil {
		t.Fatalf("could not execute request to Kibana/Fleet: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		// On error dump the whole request response so we can easily spot
		// what went wrong.
		t.Errorf("received a non 200-OK when adding overwrite to policy. "+
			"Status code: %d", resp.StatusCode)
		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			t.Fatalf("could not dump error response from Kibana: %s", err)
		}
		// Make debugging as easy as possible
		t.Log("================================================================================")
		t.Log("Kibana error response:")
		t.Log(string(respDump))
		t.FailNow()
	}
}

func requireEventLogFileExistsWithData(t *testing.T, agentFixture *atesting.Fixture) {
	// Now the Elastic-Agent is running, so validate the Event log file.
	// Because the path changes based on the Elastic-Agent version, we
	// use glob to find the file
	var logFileName string
	require.Eventually(t, func() bool {
		// We ignore this error because the folder might not be there.
		// Once the folder and file are there, then this call should succeed
		// and we can read the file.
		glob := filepath.Join(
			agentFixture.WorkDir(),
			"data", "elastic-agent-*", "logs", "events", "*")
		files, err := filepath.Glob(glob)
		if err != nil {
			t.Fatalf("could not scan for the events log file: %s", err)
		}

		if len(files) == 1 {
			logFileName = files[0]
			return true
		}

		return false

	}, time.Minute, time.Second, "could not find event log file")

	logEntryBytes, err := os.ReadFile(logFileName)
	if err != nil {
		t.Fatalf("cannot read file '%s': %s", logFileName, err)
	}

	logEntry := string(logEntryBytes)
	expectedStr := "Cannot index event"
	if !strings.Contains(logEntry, expectedStr) {
		t.Errorf(
			"did not find the expected log entry ('%s') in the events log file",
			expectedStr)
		t.Log("Event log file contents:")
		t.Log(logEntry)
	}
}

func collectDiagnosticsAndVeriflyLogs(
	t *testing.T,
	ctx context.Context,
	agentFixture *atesting.Fixture,
	cmd,
	expectedFiles []string) {

	diagPath, err := agentFixture.ExecDiagnostics(ctx, cmd...)
	if err != nil {
		t.Fatalf("could not execute diagnostics excluding events log: %s", err)
	}

	extractionDir := t.TempDir()
	extractZipArchive(t, diagPath, extractionDir)
	diagLogFiles, diagEventLogFiles := getLogFilenames(
		t,
		filepath.Join(extractionDir, "logs", "elastic-agent*"))
	allLogs := append(diagLogFiles, diagEventLogFiles...)

	require.ElementsMatch(
		t,
		expectedFiles,
		allLogs,
		"expected: 'listA', got: 'listB'")
}

func getLogFilenames(
	t *testing.T,
	basepath string,
) (logFiles, eventLogFiles []string) {

	logFilesGlob := filepath.Join(basepath, "*.ndjson")
	logFilesPath, err := filepath.Glob(logFilesGlob)
	if err != nil {
		t.Fatalf("could not get log file names:%s", err)
	}

	for _, f := range logFilesPath {
		logFiles = append(logFiles, filepath.Base(f))
	}

	eventLogFilesGlob := filepath.Join(basepath, "events", "*.ndjson")
	eventLogFilesPath, err := filepath.Glob(eventLogFilesGlob)
	if err != nil {
		t.Fatalf("could not get log file names:%s", err)
	}

	for _, f := range eventLogFilesPath {
		eventLogFiles = append(eventLogFiles, filepath.Base(f))
	}

	return logFiles, eventLogFiles
}
