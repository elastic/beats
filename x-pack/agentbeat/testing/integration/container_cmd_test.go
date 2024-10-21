// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/kibana"
	"github.com/elastic/elastic-agent/pkg/core/process"
	atesting "github.com/elastic/elastic-agent/pkg/testing"
	"github.com/elastic/elastic-agent/pkg/testing/define"
	"github.com/elastic/elastic-agent/pkg/testing/tools/fleettools"
)

func createPolicy(
	t *testing.T,
	ctx context.Context,
	agentFixture *atesting.Fixture,
	info *define.Info,
	policyName string,
	dataOutputID string) (string, string) {

	createPolicyReq := kibana.AgentPolicy{
		Name:        policyName,
		Namespace:   info.Namespace,
		Description: "test policy for agent enrollment",
		MonitoringEnabled: []kibana.MonitoringEnabledOption{
			kibana.MonitoringEnabledLogs,
			kibana.MonitoringEnabledMetrics,
		},
		AgentFeatures: []map[string]interface{}{
			{
				"name":    "test_enroll",
				"enabled": true,
			},
		},
	}

	if dataOutputID != "" {
		createPolicyReq.DataOutputID = dataOutputID
	}

	// Create policy
	policy, err := info.KibanaClient.CreatePolicy(ctx, createPolicyReq)
	if err != nil {
		t.Fatalf("could not create Agent Policy: %s", err)
	}

	// Create enrollment API key
	createEnrollmentAPIKeyReq := kibana.CreateEnrollmentAPIKeyRequest{
		PolicyID: policy.ID,
	}

	t.Logf("Creating enrollment API key...")
	enrollmentToken, err := info.KibanaClient.CreateEnrollmentAPIKey(ctx, createEnrollmentAPIKeyReq)
	if err != nil {
		t.Fatalf("unable to create enrolment API key: %s", err)
	}

	return policy.ID, enrollmentToken.APIKey
}

func prepareAgentCMD(
	t *testing.T,
	ctx context.Context,
	agentFixture *atesting.Fixture,
	args []string,
	env []string) (*exec.Cmd, *strings.Builder) {

	cmd, err := agentFixture.PrepareAgentCommand(ctx, args)
	if err != nil {
		t.Fatalf("could not prepare agent command: %s", err)
	}

	t.Cleanup(func() {
		if cmd.Process != nil {
			t.Log(">> cleaning up: killing the Elastic-Agent process")
			if err := cmd.Process.Kill(); err != nil {
				t.Fatalf("could not kill Elastic-Agent process: %s", err)
			}

			// Kill does not wait for the process to finish, so we wait here
			state, err := cmd.Process.Wait()
			if err != nil {
				t.Errorf("Elastic-Agent exited with error after kill signal: %s", err)
				t.Errorf("Elastic-Agent exited with status %d", state.ExitCode())
				out, err := cmd.CombinedOutput()
				if err == nil {
					t.Log(string(out))
				}
			}

			return
		}
		t.Log(">> cleaning up: no process to kill")
	})

	agentOutput := strings.Builder{}
	cmd.Stderr = &agentOutput
	cmd.Stdout = &agentOutput
	cmd.Env = append(os.Environ(), env...)
	return cmd, &agentOutput
}

func TestContainerCMD(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Group: "container",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	// prepare must be called otherwise `agentFixture.WorkDir()` will be empty
	// and it must be set so the `STATE_PATH` below gets a valid path.
	err = agentFixture.Prepare(ctx)
	require.NoError(t, err)

	fleetURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	if err != nil {
		t.Fatalf("could not get Fleet URL: %s", err)
	}

	_, enrollmentToken := createPolicy(
		t,
		ctx,
		agentFixture,
		info,
		fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String()),
		"")
	env := []string{
		"FLEET_ENROLL=1",
		"FLEET_URL=" + fleetURL,
		"FLEET_ENROLLMENT_TOKEN=" + enrollmentToken,
		// As the agent isn't built for a container, it's upgradable, triggering
		// the start of the upgrade watcher. If `STATE_PATH` isn't set, the
		// upgrade watcher will commence from a different path within the
		// container, distinct from the current execution path.
		"STATE_PATH=" + agentFixture.WorkDir(),
	}

	cmd, agentOutput := prepareAgentCMD(t, ctx, agentFixture, []string{"container"}, env)
	t.Logf(">> running binary with: %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		t.Fatalf("error running container cmd: %s", err)
	}

	require.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err = agentFixture.IsHealthy(ctx, withEnv(env))
		return err == nil
	},
		5*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\", Agent logs\n%s",
		err, agentOutput,
	)
}

func TestContainerCMDWithAVeryLongStatePath(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Group: "container",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	fleetURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	if err != nil {
		t.Fatalf("could not get Fleet URL: %s", err)
	}

	testCases := map[string]struct {
		statePath          string
		expectedStatePath  string
		expectedSocketPath string
		expectError        bool
	}{
		"small path": { // Use the set path
			statePath:          filepath.Join(os.TempDir(), "foo", "bar"),
			expectedStatePath:  filepath.Join(os.TempDir(), "foo", "bar"),
			expectedSocketPath: "/tmp/foo/bar/data/smp7BzlzcwgrLK4PUxpu7G1O5UwV4adr.sock",
		},
		"no path set": { // Use the default path
			statePath:          "",
			expectedStatePath:  "/usr/share/elastic-agent/state",
			expectedSocketPath: "/usr/share/elastic-agent/state/data/Td8I7R-Zby36_zF_IOd9QVNlFblNEro3.sock",
		},
		"long path": { // Path too long to create a unix socket, it will use /tmp/elastic-agent
			statePath:          "/tmp/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedStatePath:  "/tmp/ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			expectedSocketPath: "/tmp/elastic-agent/Xegnlbb8QDcqNLPzyf2l8PhVHjWvlQgZ.sock",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
			require.NoError(t, err)

			_, enrollmentToken := createPolicy(
				t,
				ctx,
				agentFixture,
				info,
				fmt.Sprintf("test-policy-enroll-%s", uuid.Must(uuid.NewV4()).String()),
				"")

			env := []string{
				"FLEET_ENROLL=1",
				"FLEET_URL=" + fleetURL,
				"FLEET_ENROLLMENT_TOKEN=" + enrollmentToken,
				"STATE_PATH=" + tc.statePath,
			}

			cmd, agentOutput := prepareAgentCMD(t, ctx, agentFixture, []string{"container"}, env)
			t.Logf(">> running binary with: %v", cmd.Args)
			if err := cmd.Start(); err != nil {
				t.Fatalf("error running container cmd: %s", err)
			}

			require.Eventuallyf(t, func() bool {
				// This will return errors until it connects to the agent,
				// they're mostly noise because until the agent starts running
				// we will get connection errors. If the test fails
				// the agent logs will be present in the error message
				// which should help to explain why the agent was not
				// healthy.
				err = agentFixture.IsHealthy(ctx, withEnv(env))
				return err == nil
			},
				1*time.Minute, time.Second,
				"Elastic-Agent did not report healthy. Agent status error: \"%v\", Agent logs\n%s",
				err, agentOutput,
			)

			t.Cleanup(func() {
				_ = os.RemoveAll(tc.expectedStatePath)
			})

			// Now that the Elastic-Agent is healthy, check that the control socket path
			// is the expected one
			if _, err := os.Stat(tc.expectedStatePath); err != nil {
				t.Errorf("cannot stat expected state path ('%s'): %s", tc.expectedStatePath, err)
			}
			if _, err := os.Stat(tc.expectedSocketPath); err != nil {
				t.Errorf("cannot stat expected socket path ('%s'): %s", tc.expectedSocketPath, err)
			}
			containerPaths := filepath.Join(tc.expectedStatePath, "container-paths.yml")
			if _, err := os.Stat(tc.expectedSocketPath); err != nil {
				t.Errorf("cannot stat expected container-paths.yml path ('%s'): %s", containerPaths, err)
			}

			if t.Failed() {
				containerPathsContent, err := os.ReadFile(containerPaths)
				if err != nil {
					t.Fatalf("could not read container-paths.yml: %s", err)
				}

				t.Log("contents of 'container-paths-yml'")
				t.Log(string(containerPathsContent))
			}
		})
	}
}

func withEnv(env []string) process.CmdOption {
	return func(c *exec.Cmd) error {
		c.Env = append(os.Environ(), env...)
		return nil
	}
}

func TestContainerCMDEventToStderr(t *testing.T) {
	info := define.Require(t, define.Requirements{
		Stack: &define.Stack{},
		Local: false,
		Sudo:  true,
		OS: []define.OS{
			{Type: define.Linux},
		},
		Group: "container",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	agentFixture, err := define.NewFixtureFromLocalBuild(t, define.Version())
	require.NoError(t, err)

	// We call agentFixture.Prepare to set the workdir
	require.NoError(t, agentFixture.Prepare(ctx), "failed preparing agent fixture")

	_, outputID := createMockESOutput(t, info)
	policyID, enrollmentAPIKey := createPolicy(
		t,
		ctx,
		agentFixture,
		info,
		fmt.Sprintf("%s-%s", t.Name(), uuid.Must(uuid.NewV4()).String()),
		outputID)

	fleetURL, err := fleettools.DefaultURL(ctx, info.KibanaClient)
	if err != nil {
		t.Fatalf("could not get Fleet URL: %s", err)
	}

	env := []string{
		"FLEET_ENROLL=1",
		"FLEET_URL=" + fleetURL,
		"FLEET_ENROLLMENT_TOKEN=" + enrollmentAPIKey,
		"STATE_PATH=" + agentFixture.WorkDir(),
		// That is what we're interested in testing
		"EVENTS_TO_STDERR=true",
	}

	cmd, agentOutput := prepareAgentCMD(t, ctx, agentFixture, []string{"container"}, env)
	addLogIntegration(t, info, policyID, "/tmp/flog.log")
	generateLogFile(t, "/tmp/flog.log", time.Second/2, 100)

	t.Logf(">> running binary with: %v", cmd.Args)
	if err := cmd.Start(); err != nil {
		t.Fatalf("error running container cmd: %s", err)
	}

	assert.Eventuallyf(t, func() bool {
		// This will return errors until it connects to the agent,
		// they're mostly noise because until the agent starts running
		// we will get connection errors. If the test fails
		// the agent logs will be present in the error message
		// which should help to explain why the agent was not
		// healthy.
		err := agentFixture.IsHealthy(ctx, withEnv(env))
		return err == nil
	},
		2*time.Minute, time.Second,
		"Elastic-Agent did not report healthy. Agent status error: \"%v\", Agent logs\n%s",
		err, agentOutput,
	)

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

func createMockESOutput(t *testing.T, info *define.Info) (string, string) {
	mockesURL := startMockES(t)
	createOutputBody := `
{
  "id": "mock-es-%[1]s",
  "name": "mock-es-%[1]s",
  "type": "elasticsearch",
  "is_default": false,
  "hosts": [
    "%s"
  ],
  "preset": "latency"
} 
`
	// The API will return an error if the output ID/name contains an
	// UUID substring, so we replace the '-' by '_' to keep the API happy.
	outputUUID := strings.Replace(uuid.Must(uuid.NewV4()).String(), "-", "_", -1)
	bodyStr := fmt.Sprintf(createOutputBody, outputUUID, mockesURL)
	bodyReader := strings.NewReader(bodyStr)
	// THE URL IS MISSING
	status, result, err := info.KibanaClient.Request(http.MethodPost, "/api/fleet/outputs", nil, nil, bodyReader)
	if err != nil {
		t.Fatalf("could execute request to create output: %#v, status: %d, result:\n%s\nBody:\n%s", err, status, string(result), bodyStr)
	}
	if status != http.StatusOK {
		t.Fatalf("creating output failed. Status code %d, response\n:%s", status, string(result))
	}

	outputResp := struct {
		Item struct {
			ID                  string   `json:"id"`
			Name                string   `json:"name"`
			Type                string   `json:"type"`
			IsDefault           bool     `json:"is_default"`
			Hosts               []string `json:"hosts"`
			Preset              string   `json:"preset"`
			IsDefaultMonitoring bool     `json:"is_default_monitoring"`
		} `json:"item"`
	}{}

	if err := json.Unmarshal(result, &outputResp); err != nil {
		t.Errorf("could not decode create output response: %s", err)
		t.Logf("Response:\n%s", string(result))
	}

	return mockesURL, outputResp.Item.ID
}

func addLogIntegration(t *testing.T, info *define.Info, policyID, logFilePath string) {
	agentPolicyBuilder := strings.Builder{}
	tmpl, err := template.New(t.Name() + "custom-log-policy").Parse(policyJSON)
	if err != nil {
		t.Fatalf("cannot parse template: %s", err)
	}

	err = tmpl.Execute(&agentPolicyBuilder, policyVars{
		Name:        "Log-Input-" + t.Name() + "-" + time.Now().Format(time.RFC3339),
		PolicyID:    policyID,
		LogFilePath: logFilePath,
		Dataset:     "logs",
		Namespace:   "default",
	})
	if err != nil {
		t.Fatalf("could not render template: %s", err)
	}
	// We keep a copy of the policy for debugging prurposes
	agentPolicy := agentPolicyBuilder.String()

	// Call Kibana to create the policy.
	// Docs: https://www.elastic.co/guide/en/fleet/current/fleet-api-docs.html#create-integration-policy-api
	resp, err := info.KibanaClient.Connection.Send(
		http.MethodPost,
		"/api/fleet/package_policies",
		nil,
		nil,
		bytes.NewBufferString(agentPolicy))
	if err != nil {
		t.Fatalf("could not execute request to Kibana/Fleet: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		// On error dump the whole request response so we can easily spot
		// what went wrong.
		t.Errorf("received a non 200-OK when adding package to policy. "+
			"Status code: %d", resp.StatusCode)
		respDump, err := httputil.DumpResponse(resp, true)
		if err != nil {
			t.Fatalf("could not dump error response from Kibana: %s", err)
		}
		// Make debugging as easy as possible
		t.Log("================================================================================")
		t.Log("Kibana error response:")
		t.Log(string(respDump))
		t.Log("================================================================================")
		t.Log("Rendered policy:")
		t.Log(agentPolicy)
		t.Log("================================================================================")
		t.FailNow()
	}
}
