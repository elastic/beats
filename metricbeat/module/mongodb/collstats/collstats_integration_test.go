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
	"context"
	"errors"
	"fmt"
	"net"
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
		logIfVerbose(t, "%s/%s event: %+v", f.Module().Name(), f.Name(), event)
		metricsetFields := event.MetricSetFields

		// Check a few event Fields
		db, _ := metricsetFields["db"].(string)
		assert.NotEmpty(t, db)

		collection, _ := metricsetFields["collection"].(string)
		assert.NotEmpty(t, collection)
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

func getConfigWithScale(host string, scale int) map[string]interface{} {
	cfg := getConfig(host)
	cfg["scale"] = scale
	return cfg
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
		{version: "4.4", port: "27516", expectExtendedStats: false}, // oldest available; uses legacy collStats command
		{version: "5.0", port: "27518", expectExtendedStats: false},
		{version: "6.0", port: "27520", expectExtendedStats: false}, // 6.0 < 6.2.0: uses legacy collStats command
		{version: "7.0", port: "27519", expectExtendedStats: true},  // 7.0 > 6.2.0: uses $collStats aggregation
		{version: "8.0", port: "27521", expectExtendedStats: true},  // latest LTS; uses $collStats aggregation
	}

	originalWD, err := os.Getwd()
	require.NoError(t, err, "resolve working directory")

	for _, tc := range cases {
		tc := tc
		t.Run("mongo_"+strings.ReplaceAll(tc.version, ".", "_"), func(t *testing.T) {
			logIfVerbose(t, "Starting test for MongoDB %s on port %s", tc.version, tc.port)

			require.NoError(t, os.Chdir(absStandaloneDir))
			t.Cleanup(func() {
				if err := os.Chdir(originalWD); err != nil {
					t.Logf("failed to restore working dir: %v", err)
				}
			})

			projectName := fmt.Sprintf("mbcollstatsstandalone%s", strings.ReplaceAll(tc.version, ".", ""))
			logIfVerbose(t, "Using project name: %s", projectName)
			logIfVerbose(t, "Setting environment variables:")
			logIfVerbose(t, "  DOCKER_COMPOSE_PROJECT_NAME=%s", projectName)
			logIfVerbose(t, "  COMPOSE_PROJECT_NAME=%s", projectName)
			logIfVerbose(t, "  MONGO_VERSION=%s", tc.version)
			logIfVerbose(t, "  MONGO_PORT=%s", tc.port)

			t.Setenv("DOCKER_COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("MONGO_VERSION", tc.version)
			t.Setenv("MONGO_PORT", tc.port)

			cleanupEnv := buildComposeEnv(projectName, tc.version, tc.port)
			logIfVerbose(t, "Compose environment: %v", cleanupEnv)

			t.Cleanup(func() {
				logIfVerbose(t, "Cleaning up Docker Compose project %s", projectName)
				downArgs := []string{"-f", "docker-compose.yml", "down", "-v"}
				if err := runComposeCommand(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, downArgs...); err != nil {
					t.Logf("failed to tear down compose project %s: %v", projectName, err)
				}
			})

			logIfVerbose(t, "Starting MongoDB container...")
			// Start the container using docker-compose
			upArgs := []string{"-f", "docker-compose.yml", "up", "-d"}
			if err := runComposeCommand(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, upArgs...); err != nil {
				t.Fatalf("failed to start compose project %s: %v", projectName, err)
			}

			// Wait for container to be healthy
			logIfVerbose(t, "Waiting for MongoDB to be healthy...")
			var containerReady bool
			for i := 0; i < 30; i++ {
				healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
				healthCmd := exec.CommandContext(healthCtx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", projectName), "--filter", "health=healthy", "--format", "{{.Names}}") //nolint:gosec // safe integration test
				output, _ := healthCmd.CombinedOutput()
				healthCancel()
				if strings.TrimSpace(string(output)) != "" {
					containerReady = true
					break
				}
				time.Sleep(2 * time.Second)
			}

			if !containerReady {
				// Show container status for debugging
				statusCtx, statusCancel := context.WithTimeout(context.Background(), 10*time.Second)
				statusCmd := exec.CommandContext(statusCtx, "docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", projectName), "--format", "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}") //nolint:gosec // safe integration test
				statusOutput, _ := statusCmd.CombinedOutput()
				statusCancel()
				t.Logf("Container status:\n%s", string(statusOutput))

				// Show logs
				logsArgs := []string{"-f", "docker-compose.yml", "logs", "--tail=50"}
				composeLogs, _ := runComposeCommandOutput(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, logsArgs...)
				t.Logf("Docker Compose logs:\n%s", composeLogs)

				t.Fatalf("MongoDB container failed to become healthy")
			}

			// Construct the host string
			mongoHostStr := fmt.Sprintf("mongodb://localhost:%s", tc.port)
			logIfVerbose(t, "MongoDB container started and healthy, host: %s", mongoHostStr)

			logIfVerbose(t, "Running seed script from: %s", seedScript)
			logIfVerbose(t, "Seed script working directory: %s", absStandaloneDir)

			// Check if bash is available
			if _, err := exec.LookPath("bash"); err != nil {
				t.Logf("Warning: bash not found in PATH: %v", err)
			}

			// Check if seed script is executable
			if info, err := os.Stat(seedScript); err != nil {
				t.Logf("Warning: cannot stat seed script: %v", err)
			} else {
				logIfVerbose(t, "Seed script permissions: %v", info.Mode())
			}

			seedStart := time.Now()
			if err := runSeedScript(seedScript, absStandaloneDir, cleanupEnv); err != nil {
				t.Logf("Seed script failed after %v", time.Since(seedStart))
				t.Logf("Seed script error: %v", err)

				// List running containers for debugging
				listCtx, listCancel := context.WithTimeout(context.Background(), 10*time.Second)
				listCmd := exec.CommandContext(listCtx, "docker", "ps", "-a", "--format", "table {{.Names}}\\t{{.Status}}\\t{{.Ports}}")
				listOutput, _ := listCmd.CombinedOutput()
				listCancel()
				t.Logf("Current Docker containers:\n%s", string(listOutput))

				// Show compose logs for debugging
				logsArgs := []string{"-f", "docker-compose.yml", "logs", "--tail=50"}
				composeLogs, _ := runComposeCommandOutput(cmdName, prefixArgs, absStandaloneDir, cleanupEnv, logsArgs...)
				t.Logf("Docker Compose logs:\n%s", composeLogs)

				require.NoError(t, err, "seed standalone database")
			}
			logIfVerbose(t, "Seed script completed successfully in %v", time.Since(seedStart))

			logIfVerbose(t, "Creating metricset with config for host: %s", mongoHostStr)
			f := mbtest.NewReportingMetricSetV2Error(t, getConfig(mongoHostStr))

			logIfVerbose(t, "Fetching collstats events...")
			events, errs := mbtest.ReportingFetchV2Error(f)
			if len(errs) > 0 {
				t.Logf("Fetch errors: %v", errs)
			}
			require.Empty(t, errs, "expected no fetch errors")
			require.NotEmpty(t, events, "expected collstats events")
			logIfVerbose(t, "Fetched %d collstats events", len(events))

			verifyStandaloneEvents(t, events, tc.expectExtendedStats)

			// Regression guard: a non-default scale must not break collection on
			// any version. The legacy collStats command path (MongoDB < 6.2) must
			// build an ordered command document; using an unordered multi-key map
			// makes the driver reject every fetch ("multi-key map passed in for
			// ordered parameter cmd"), yielding zero events.
			const scale = 1024
			logIfVerbose(t, "Fetching collstats events with scale=%d...", scale)
			scaledFetcher := mbtest.NewReportingMetricSetV2Error(t, getConfigWithScale(mongoHostStr, scale))
			scaledEvents, scaledErrs := mbtest.ReportingFetchV2Error(scaledFetcher)
			if len(scaledErrs) > 0 {
				t.Logf("Scaled fetch errors: %v", scaledErrs)
			}
			require.Empty(t, scaledErrs, "expected no fetch errors with scale=%d", scale)
			require.NotEmpty(t, scaledEvents, "expected collstats events with scale=%d", scale)
			verifyScaledEvents(t, scaledEvents, scale)
		})
	}
}

// verifyScaledEvents asserts that, for the seeded collections, a configured scale
// is honored: scaleFactor (when reported by the server) reflects the requested
// scale. Storage sizes are scaled server-side, so we only assert the factor.
func verifyScaledEvents(t *testing.T, events []mb.Event, scale int) {
	t.Helper()

	for _, event := range events {
		collection, _ := event.MetricSetFields["collection"].(string)
		if _, seeded := seededCollectionExpectations[collection]; !seeded {
			continue
		}

		stats, ok := event.MetricSetFields["stats"].(mapstr.M)
		require.True(t, ok, "stats should be map")

		if sf, exists := stats["scaleFactor"]; exists {
			assertExactNumber(t, sf, float64(scale), "stats.scaleFactor should reflect configured scale")
		}
	}
}

func TestFetchShardedMongos(t *testing.T) {
	relShardedDir := filepath.Join("testing", "mongodb", "sharded")
	absShardedDir, err := filepath.Abs(relShardedDir)
	require.NoError(t, err, "resolve sharded directory")

	composeFile := filepath.Join(absShardedDir, "docker-compose.yml")
	if _, err := os.Stat(composeFile); errors.Is(err, os.ErrNotExist) {
		t.Skipf("sharded docker-compose file not found: %s", composeFile)
	}

	initScript := filepath.Join(absShardedDir, "init-sharded-cluster.sh")
	if _, err := os.Stat(initScript); errors.Is(err, os.ErrNotExist) {
		t.Skipf("sharded init script not found: %s", initScript)
	}

	cmdName, prefixArgs := detectComposeCommand(t)
	cases := []struct {
		version             string
		expectExtendedStats bool
	}{
		{version: "5.0", expectExtendedStats: false},
		{version: "7.0", expectExtendedStats: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("mongo_"+strings.ReplaceAll(tc.version, ".", "_"), func(t *testing.T) {
			projectName := fmt.Sprintf("mbcollstatssharded%s", strings.ReplaceAll(tc.version, ".", ""))
			cleanupEnv := buildComposeEnv(projectName, tc.version, "")

			t.Setenv("DOCKER_COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("COMPOSE_PROJECT_NAME", projectName)
			t.Setenv("MONGO_VERSION", tc.version)

			t.Cleanup(func() {
				downArgs := []string{"-f", "docker-compose.yml", "down", "-v"}
				if err := runComposeCommand(cmdName, prefixArgs, absShardedDir, cleanupEnv, downArgs...); err != nil {
					t.Logf("failed to tear down compose project %s: %v", projectName, err)
				}
			})

			// The sharded fixture binds dedicated host ports for the mongos
			// routers (27117 for mongos1, 27118 for mongos2) to avoid colliding
			// with the standalone mongodb fixture on 27017. If something else
			// already owns those ports, "compose up" fails deep inside Docker
			// with an opaque "port is already allocated" error. Surface that
			// clearly up front so the failure is actionable instead of flaky.
			requireHostPortsFree(t, "27117", "27118")

			upArgs := []string{"-f", "docker-compose.yml", "up", "-d"}
			require.NoError(t, runComposeCommand(cmdName, prefixArgs, absShardedDir, cleanupEnv, upArgs...), "start sharded compose project")
			require.NoError(t, waitForHealthyContainers(projectName, []string{"config1", "shard1-primary", "shard2-primary"}), "wait for sharded containers")
			require.NoError(t, runSeedScript(initScript, absShardedDir, cleanupEnv), "initialize sharded cluster")

			f := mbtest.NewReportingMetricSetV2Error(t, getConfig("mongodb://localhost:27117"))
			events, errs := mbtest.ReportingFetchV2Error(f)
			require.Empty(t, errs, "expected mongos fetch to fall back from top to listCollections")
			require.NotEmpty(t, events, "expected collstats events from mongos")

			assertShardedCollectionEvent(t, events, "coll_hash", 20000, tc.expectExtendedStats)
			assertShardedCollectionEvent(t, events, "coll_range", 20000, tc.expectExtendedStats)
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

		// Standalone deployments are never sharded: shardCount must be absent for
		// every event, on both the legacy command path and the $collStats
		// aggregation path. Regression guard for $collStats emitting a top-level
		// "host" that must not be mistaken for shard metadata (which previously
		// tagged standalone results with shardCount=1).
		_, hasShardCount := statsValue["shardCount"]
		require.Falsef(t, hasShardCount, "standalone event %s.%s must not report shardCount", db, collection)

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

func assertShardedCollectionEvent(t *testing.T, events []mb.Event, collectionName string, expectedCount float64, expectExtendedStats bool) {
	t.Helper()

	for _, event := range events {
		metricsetFields := event.MetricSetFields
		collection, _ := metricsetFields["collection"].(string)
		if collection != collectionName {
			continue
		}

		statsValue, ok := metricsetFields["stats"].(mapstr.M)
		require.True(t, ok, "stats should be map")
		assertExactNumber(t, statsValue["count"], expectedCount, "stats.count should match sharded documents")
		if expectExtendedStats {
			assertExactNumber(t, statsValue["shardCount"], 2, "stats.shardCount should report both shards")
			assertNonNegativeNumber(t, statsValue["freeStorageSize"], "stats.freeStorageSize should be non-negative")
		}
		return
	}

	t.Fatalf("expected event for sharded collection %s", collectionName)
}

func waitForHealthyContainers(projectName string, services []string) error {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", fmt.Sprintf("name=%s", projectName), "--filter", "health=healthy", "--format", "{{.Names}}") //nolint:gosec // safe integration test
		output, _ := cmd.CombinedOutput()
		cancel()

		healthyContainers := string(output)
		allHealthy := true
		for _, service := range services {
			expectedName := fmt.Sprintf("%s-%s-1", projectName, service)
			if !strings.Contains(healthyContainers, expectedName) {
				allHealthy = false
				break
			}
		}
		if allHealthy {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timed out waiting for healthy containers for project %s", projectName)
}

// requireHostPortsFree fails the test early with an actionable message if any of
// the given TCP host ports is already bound. The sharded fixture publishes fixed
// host ports, so a stray MongoDB (e.g. a leftover container from another module's
// compose project) would otherwise cause an opaque "port is already allocated"
// failure deep inside "compose up".
func requireHostPortsFree(t *testing.T, ports ...string) {
	t.Helper()
	var dialer net.Dialer
	for _, port := range ports {
		addr := net.JoinHostPort("127.0.0.1", port)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		cancel()
		if err == nil {
			conn.Close()
			t.Fatalf("host port %s is already in use; the sharded mongos fixture needs it free. "+
				"Stop any container/process bound to it (e.g. `docker ps --filter publish=%s`) and retry.", port, port)
		}
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
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "docker", "compose", "version")
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, commandArgs...)
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, commandArgs...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func runSeedScript(scriptPath, dir string, env []string) error {
	// Try to use bash, fall back to sh if not available
	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}

	seedTimeout := 5 * time.Minute

	// Create the command - use absolute path to be safe
	ctx, cancel := context.WithTimeout(context.Background(), seedTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, scriptPath)
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
		return fmt.Errorf("failed to start seed script with %s: %w", shell, err)
	}

	// Read output
	var outputBuf, errorBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
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
		defer wg.Done()
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

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		wg.Wait()
		if err != nil {
			return fmt.Errorf("seed script failed with %s: %w\nSTDOUT:\n%s\nSTDERR:\n%s", shell, err, outputBuf.String(), errorBuf.String())
		}
	case <-ctx.Done():
		wg.Wait()
		return fmt.Errorf("seed script timed out after %s with %s\nSTDOUT:\n%s\nSTDERR:\n%s", seedTimeout, shell, outputBuf.String(), errorBuf.String())
	}

	return nil
}

func logIfVerbose(t *testing.T, format string, args ...interface{}) {
	if !shouldLogVerbose() {
		return
	}

	t.Helper()
	t.Logf(format, args...)
}

func shouldLogVerbose() bool {
	if v := strings.ToLower(os.Getenv("METRICBEAT_COLLSTATS_LOGS")); v == "1" || v == "true" || v == "yes" {
		return true
	}

	if os.Getenv("CI") != "" {
		return false
	}

	return testing.Verbose()
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
