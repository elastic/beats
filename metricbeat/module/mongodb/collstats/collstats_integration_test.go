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

//go:build integration

package collstats

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetch(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	for _, event := range events {
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)
		metricsetFields := event.MetricSetFields

		// Check a few event Fields
		db, _ := metricsetFields["db"].(string)
		assert.NotEqual(t, db, "")

		collection, _ := metricsetFields["collection"].(string)
		assert.NotEqual(t, collection, "")
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("error trying to create data.json file:", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mongodb",
		"metricsets": []string{"collstats"},
		"hosts":      []string{host},
	}
}

func TestFetchStandaloneVersions(t *testing.T) {
	relStandaloneDir := filepath.Join("testing", "mongodb", "standalone")
	absStandaloneDir, err := filepath.Abs(relStandaloneDir)
	require.NoError(t, err, "resolve standalone directory")

	composeFile := filepath.Join(absStandaloneDir, "docker-compose.yml")

	if _, err := os.Stat(composeFile); errors.Is(err, os.ErrNotExist) {
		t.Skipf("standalone docker-compose file not found: %s", composeFile)
	}

	seedScript := filepath.Join(absStandaloneDir, "seed-standalone.sh")
	if _, err := os.Stat(seedScript); errors.Is(err, os.ErrNotExist) {
		t.Skipf("standalone seed script not found: %s", seedScript)
	}

	cmdName, prefixArgs := detectComposeCommand(t)

	cases := []struct {
		version             string
		port                string
		expectExtendedStats bool
	}{
		{version: "5.0", port: "27518", expectExtendedStats: false},
		{version: "7.0", port: "27519", expectExtendedStats: true},
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err, "resolve working directory")

	for _, tc := range cases {
		tc := tc
		t.Run("mongo_"+strings.ReplaceAll(tc.version, ".", "_"), func(t *testing.T) {
			t.Logf("Starting test for MongoDB %s on port %s", tc.version, tc.port)

			require.NoError(t, os.Chdir(absStandaloneDir))
			t.Cleanup(func() {
				if err := os.Chdir(originalWD); err != nil {
					t.Logf("failed to restore working dir: %v", err)
				}
			})

			projectName := fmt.Sprintf("mbcollstatsstandalone%s", strings.ReplaceAll(tc.version, ".", ""))
			t.Logf("Using project name: %s", projectName)
			t.Logf("Setting environment variables:")
			t.Logf("  DOCKER_COMPOSE_PROJECT_NAME=%s", projectName)
			t.Logf("  COMPOSE_PROJECT_NAME=%s", projectName)
			t.Logf("  MONGO_VERSION=%s", tc.version)
			t.Logf("  MONGO_PORT=%s", tc.port)

			t.Setenv("DOCKER_COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("MONGO_VERSION", tc.version)
			t.Setenv("MONGO_PORT", tc.port)

			cleanupEnv := buildComposeEnv(projectName, tc.version, tc.port)
			t.Logf("Compose environment: %v", cleanupEnv)

			t.Cleanup(func() {
				t.Logf("Cleaning up Docker Compose project %s", projectName)
				downArgs := []string{"-f", "docker-compose.yml", "down", "-v"}
				if err := runComposeCommand(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, downArgs...); err != nil {
					t.Logf("failed to tear down compose project %s: %v", projectName, err)
				}
			})

			t.Logf("Starting MongoDB container...")
			// Start the container using docker-compose
			upArgs := []string{"-f", "docker-compose.yml", "up", "-d"}
			if err := runComposeCommand(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, upArgs...); err != nil {
				t.Fatalf("failed to start compose project %s: %v", projectName, err)
			}

			// Wait for container to be healthy
			t.Logf("Waiting for MongoDB to be healthy...")
			var containerReady bool
			for i := 0; i < 30; i++ {
				healthCmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", projectName), "--filter", "health=healthy", "--format", "{{.Names}}")
				output, _ := healthCmd.CombinedOutput()
				if strings.TrimSpace(string(output)) != "" {
					containerReady = true
					break
				}
				time.Sleep(2 * time.Second)
			}

			if !containerReady {
				// Show container status for debugging
				statusCmd := exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", projectName), "--format", "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}")
				statusOutput, _ := statusCmd.CombinedOutput()
				t.Logf("Container status:\n%s", string(statusOutput))

				// Show logs
				logsCmd := append([]string{}, prefixArgs...)
				logsCmd = append(logsCmd, "-f", "docker-compose.yml", "logs", "--tail=50")
				composeLogs, _ := runComposeCommandOutput(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, logsCmd...)
				t.Logf("Docker Compose logs:\n%s", composeLogs)

				t.Fatalf("MongoDB container failed to become healthy")
			}

			// Construct the host string
			mongoHostStr := fmt.Sprintf("mongodb://localhost:%s", tc.port)
			t.Logf("MongoDB container started and healthy, host: %s", mongoHostStr)

			t.Logf("Running seed script from: %s", seedScript)
			t.Logf("Seed script working directory: %s", absStandaloneDir)
			seedStart := time.Now()
			if err := runSeedScript(seedScript, absStandaloneDir, cleanupEnv); err != nil {
				t.Logf("Seed script failed after %v", time.Since(seedStart))

				// List running containers for debugging
				listCmd := exec.Command("docker", "ps", "-a", "--format", "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}")
				listOutput, _ := listCmd.CombinedOutput()
				t.Logf("Current Docker containers:\n%s", string(listOutput))

				// Show compose logs for debugging
				logsCmd := append([]string{}, prefixArgs...)
				logsCmd = append(logsCmd, "-f", "docker-compose.yml", "logs", "--tail=50")
				composeLogs, _ := runComposeCommandOutput(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, logsCmd...)
				t.Logf("Docker Compose logs:\n%s", composeLogs)

				require.NoError(t, err, "seed standalone database")
			}
			t.Logf("Seed script completed successfully in %v", time.Since(seedStart))

			t.Logf("Creating metricset with config for host: %s", mongoHostStr)
			f := mbtest.NewReportingMetricSetV2Error(t, getConfig(mongoHostStr))

			t.Logf("Fetching collstats events...")
			events, errs := mbtest.ReportingFetchV2Error(f)
			if len(errs) > 0 {
				t.Logf("Fetch errors: %v", errs)
			}
			require.Empty(t, errs, "expected no fetch errors")
			require.NotEmpty(t, events, "expected collstats events")
			t.Logf("Fetched %d collstats events", len(events))

			verifyStandaloneEvents(t, events, tc.expectExtendedStats)
		})
	}
}

const (
	seededDatabaseName    = "mbtest"
	seededCollectionCount = 3
)

type collectionExpectation struct {
	expectedCount float64
	expectCapped  bool
}

var seededCollectionExpectations = map[string]collectionExpectation{
	"test_collection": {expectedCount: 10000},
	"test_indexed":    {expectedCount: 5000},
	"test_capped":     {expectedCount: 5000, expectCapped: true},
}

func verifyStandaloneEvents(t *testing.T, events []mb.Event, expectExtendedStats bool) {
	t.Helper()

	require.GreaterOrEqualf(t, len(events), seededCollectionCount, "expected at least %d events", seededCollectionCount)
	seenCollections := make(map[string]bool, len(seededCollectionExpectations))
	for name := range seededCollectionExpectations {
		seenCollections[name] = false
	}

	for _, event := range events {
		metricsetFields := event.MetricSetFields

		db, ok := metricsetFields["db"].(string)
		require.True(t, ok, "db field should be string")
		require.NotEmpty(t, db, "db field should not be empty")

		collection, ok := metricsetFields["collection"].(string)
		require.True(t, ok, "collection field should be string")
		require.NotEmpty(t, collection, "collection should not be empty")

		statsValue, ok := metricsetFields["stats"].(mapstr.M)
		require.True(t, ok, "stats should be map")

		totalValue, ok := metricsetFields["total"].(mapstr.M)
		require.True(t, ok, "total should be map")

		if expectation, exists := seededCollectionExpectations[collection]; exists {
			seenCollections[collection] = true
			require.Equal(t, seededDatabaseName, db, "unexpected database name for seeded collection")
			assertExactNumber(t, statsValue["count"], expectation.expectedCount, "stats.count should match seeded documents")
			assertPositiveNumber(t, statsValue["storageSize"], "stats.storageSize should be positive")
			assertPositiveNumber(t, statsValue["totalSize"], "stats.totalSize should be positive")

			assertNonNegativeNumber(t, totalValue["count"], "total.count should be non-negative")

			if expectation.expectCapped {
				assertBooleanField(t, statsValue, "capped", true, "stats.capped should be true for capped collection")
			}

			if expectExtendedStats {
				assertExtendedStatsFields(t, statsValue)
			}
		} else {
			assertNonNegativeNumber(t, statsValue["count"], "stats.count should be non-negative")
			assertNonNegativeNumber(t, statsValue["storageSize"], "stats.storageSize should be non-negative")
			assertNonNegativeNumber(t, statsValue["totalSize"], "stats.totalSize should be non-negative")
			assertNonNegativeNumber(t, totalValue["count"], "total.count should be non-negative")
		}
	}

	for coll, seen := range seenCollections {
		require.Truef(t, seen, "expected event for seeded collection %s", coll)
	}
}

func assertPositiveNumber(t *testing.T, value interface{}, msg string) {
	t.Helper()
	number, ok := normalizeNumeric(value)
	require.Truef(t, ok, "%s must be numeric (got %T)", msg, value)
	require.Greaterf(t, number, float64(0), "%s must be > 0 (got %v)", msg, value)
}

func assertNonNegativeNumber(t *testing.T, value interface{}, msg string) {
	t.Helper()
	number, ok := normalizeNumeric(value)
	require.Truef(t, ok, "%s must be numeric (got %T)", msg, value)
	require.GreaterOrEqualf(t, number, float64(0), "%s must be >= 0 (got %v)", msg, value)
}

func assertExactNumber(t *testing.T, value interface{}, expected float64, msg string) {
	t.Helper()
	number, ok := normalizeNumeric(value)
	require.Truef(t, ok, "%s must be numeric (got %T)", msg, value)
	require.InDeltaf(t, expected, number, 0.5, "%s expected %.0f (got %.2f)", msg, expected, number)
}

func assertBooleanField(t *testing.T, stats mapstr.M, key string, expected bool, msg string) {
	t.Helper()
	value, exists := stats[key]
	require.Truef(t, exists, "%s (field %s missing)", msg, key)
	actual, ok := value.(bool)
	require.Truef(t, ok, "%s must be boolean (got %T)", msg, value)
	require.Equalf(t, expected, actual, "%s expected %v (got %v)", msg, expected, actual)
}

func assertExtendedStatsFields(t *testing.T, stats mapstr.M) {
	t.Helper()
	for _, key := range []string{"freeStorageSize", "scaleFactor"} {
		value, exists := stats[key]
		require.Truef(t, exists, "expected extended stats field %s", key)
		assertNonNegativeNumber(t, value, fmt.Sprintf("stats.%s should be non-negative", key))
	}

	if value, exists := stats["numOrphanDocs"]; exists {
		assertNonNegativeNumber(t, value, "stats.numOrphanDocs should be non-negative")
	}
}

func normalizeNumeric(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

var (
	composeOnce    sync.Once
	composeCommand string
	composePrefix  []string
	composeErr     error
)

func detectComposeCommand(t *testing.T) (string, []string) {
	t.Helper()

	composeOnce.Do(func() {
		if _, err := exec.LookPath("docker"); err == nil {
			cmd := exec.Command("docker", "compose", "version")
			if err := cmd.Run(); err == nil {
				composeCommand = "docker"
				composePrefix = []string{"compose"}
				return
			}
		}

		if path, err := exec.LookPath("docker-compose"); err == nil {
			composeCommand = path
			composePrefix = nil
			return
		}

		composeErr = errors.New("docker compose plugin or docker-compose binary not available")
	})

	if composeErr != nil {
		t.Skipf("skipping replica set tests: %v", composeErr)
	}

	return composeCommand, composePrefix
}

func runComposeCommand(cmdName string, prefixArgs []string, dir string, env []string, args ...string) error {
	commandArgs := append([]string{}, prefixArgs...)
	commandArgs = append(commandArgs, args...)

	cmd := exec.Command(cmdName, commandArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compose command %v failed: %w (output: %s)", commandArgs, err, string(output))
	}

	return nil
}

func runComposeCommandOutput(cmdName string, prefixArgs []string, dir string, env []string, args ...string) (string, error) {
	commandArgs := append([]string{}, prefixArgs...)
	commandArgs = append(commandArgs, args...)

	cmd := exec.Command(cmdName, commandArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runSeedScript(scriptPath, dir string, env []string) error {
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	// Create pipes for stdout and stderr to see real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start seed script: %w", err)
	}

	// Read output
	var outputBuf, errorBuf strings.Builder
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				outputBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				errorBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for the command to finish with a timeout
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("seed script failed: %w (stdout: %s, stderr: %s)", err, outputBuf.String(), errorBuf.String())
		}
	case <-time.After(60 * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("seed script timed out after 60 seconds (stdout: %s, stderr: %s)", outputBuf.String(), errorBuf.String())
	}

	return nil
}

func buildComposeEnv(projectName, version, port string) []string {
	env := []string{
		fmt.Sprintf("DOCKER_COMPOSE_PROJECT_NAME=%s", projectName),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", projectName),
		fmt.Sprintf("MONGO_VERSION=%s", version),
	}

	if port != "" {
		env = append(env, fmt.Sprintf("MONGO_PORT=%s", port))
	}

	return env
}
