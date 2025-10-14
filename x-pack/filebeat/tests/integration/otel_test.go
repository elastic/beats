// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && !agentbeat

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/gofrs/uuid/v5"

	libbeattesting "github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/mock-es/pkg/api"
)

func writeEventsToLogFile(t *testing.T, filename string, numEvents int) {
	t.Helper()
	logFile, err := os.Create(filename)
	if err != nil {
		t.Fatalf("could not create file '%s': %s", filename, err)
	}
	// write events to log file
	for i := 0; i < numEvents; i++ {
		msg := fmt.Sprintf("Line %d", i)
		_, err = logFile.Write([]byte(msg + "\n"))
		require.NoErrorf(t, err, "failed to write line %d to temp file", i)
	}

	if err := logFile.Sync(); err != nil {
		t.Fatalf("could not sync log file '%s': %s", filename, err)
	}
	if err := logFile.Close(); err != nil {
		t.Fatalf("could not close log file '%s': %s", filename, err)
	}
}

func assertMapsEqual(t *testing.T, m1, m2 mapstr.M, ignoredFields []string, msg string) {
	t.Helper()

	flatM1 := m1.Flatten()
	flatM2 := m2.Flatten()
	for _, f := range ignoredFields {
		hasKeyM1, _ := flatM1.HasKey(f)
		hasKeyM2, _ := flatM2.HasKey(f)

		if !hasKeyM1 && !hasKeyM2 {
			assert.Failf(t, msg, "ignored field %q does not exist in either map, please remove it from the ignored fields", f)
		}

		flatM1.Delete(f)
		flatM2.Delete(f)
	}
	require.Equal(t, "", cmp.Diff(flatM1, flatM2), "expected maps to be equal")
}

func assertMonitoring(t *testing.T, port int) {
	address := fmt.Sprintf("http://localhost:%d", port)
	r, err := http.Get(address) //nolint:noctx,bodyclose,gosec // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, r.StatusCode, "incorrect status code")

	r, err = http.Get(address + "/not-exist") //nolint:noctx,bodyclose // fine for tests
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, r.StatusCode, "incorrect status code")
}

func TestFilebeatOTelDocumentLevelRetries(t *testing.T) {
	tests := []struct {
		name                     string
		maxRetries               int
		failuresPerEvent         int
		bulkErrorCode            string
		eventIDsToFail           []int
		expectedIngestedEventIDs []int
	}{
		{
			name:                     "bulk 429 with retries",
			maxRetries:               3,
			failuresPerEvent:         2,     // Fail 2 times, succeed on 3rd attempt
			bulkErrorCode:            "429", // retryable error
			eventIDsToFail:           []int{1, 3, 5, 7},
			expectedIngestedEventIDs: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, // All events should eventually be ingested
		},
		{
			name:                     "bulk exhausts retries",
			maxRetries:               3,
			failuresPerEvent:         5, // Fail more than max_retries
			bulkErrorCode:            "429",
			eventIDsToFail:           []int{2, 4, 6, 8},
			expectedIngestedEventIDs: []int{0, 1, 3, 5, 7, 9}, // Only non-failing events should be ingested
		},
		{
			name:                     "bulk with permanent mapping errors",
			maxRetries:               3,
			failuresPerEvent:         0,                          // always fail
			bulkErrorCode:            "400",                      // never retried
			eventIDsToFail:           []int{1, 4, 8},             // Only specific events fail
			expectedIngestedEventIDs: []int{0, 2, 3, 5, 6, 7, 9}, // Only non-failing events should be ingested
		},
	}

	const numTestEvents = 10
	reEventLine := regexp.MustCompile(`"message":"Line (\d+)"`)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ingestedTestEvents []string
			var mu sync.Mutex
			eventFailureCounts := make(map[string]int)

			deterministicHandler := func(action api.Action, event []byte) int {
				// Handle non-bulk requests
				if action.Action != "create" {
					return http.StatusOK
				}

				// Extract event ID from the event data
				if matches := reEventLine.FindSubmatch(event); len(matches) > 1 {
					eventIDStr := string(matches[1])
					eventID, err := strconv.Atoi(eventIDStr)
					if err != nil {
						return http.StatusInternalServerError
					}

					eventKey := "Line " + eventIDStr

					mu.Lock()
					defer mu.Unlock()

					isFailingEvent := slices.Contains(tt.eventIDsToFail, eventID)

					var shouldFail bool
					if isFailingEvent {
						// This event is configured to fail
						failureCount := eventFailureCounts[eventKey]

						switch tt.bulkErrorCode {
						case "400":
							// Permanent errors always fail
							shouldFail = true
						case "429":
							// Temporary errors fail until failuresPerEvent threshold
							shouldFail = failureCount < tt.failuresPerEvent
						}
					} else {
						// Events not in the fail list always succeed
						shouldFail = false
					}

					if shouldFail {
						eventFailureCounts[eventKey] = eventFailureCounts[eventKey] + 1
						if tt.bulkErrorCode == "429" {
							return http.StatusTooManyRequests
						} else {
							return http.StatusBadRequest
						}
					}

					// track ingested event
					found := false
					for _, existing := range ingestedTestEvents {
						if existing == eventKey {
							found = true
							break
						}
					}
					if !found {
						ingestedTestEvents = append(ingestedTestEvents, eventKey)
					}
					return http.StatusOK
				}

				return http.StatusOK
			}

			reader := metric.NewManualReader()
			provider := metric.NewMeterProvider(metric.WithReader(reader))

			mux := http.NewServeMux()
			mux.Handle("/", api.NewDeterministicAPIHandler(
				uuid.Must(uuid.NewV4()),
				"",
				provider,
				time.Now().Add(24*time.Hour),
				0,
				0,
				deterministicHandler,
			))

			server := httptest.NewServer(mux)
			defer server.Close()

			filebeatOTel := integration.NewBeat(
				t,
				"filebeat-otel",
				"../../filebeat.test",
				"otel",
			)

			namespace := strings.ReplaceAll(uuid.Must(uuid.NewV4()).String(), "-", "")
			index := "logs-integration-" + namespace

			beatsConfig := struct {
				Index          string
				InputFile      string
				ESEndpoint     string
				MaxRetries     int
				MonitoringPort int
			}{
				Index:          index,
				InputFile:      filepath.Join(filebeatOTel.TempDir(), "log.log"),
				ESEndpoint:     server.URL,
				MaxRetries:     tt.maxRetries,
				MonitoringPort: int(libbeattesting.MustAvailableTCP4Port(t)),
			}

			cfg := `
filebeat.inputs:
  - type: filestream
    id: filestream-input-id
    enabled: true
    file_identity.native: ~
    prospector.scanner.fingerprint.enabled: false
    paths:
      - {{.InputFile}}
output:
  elasticsearch:
    hosts:
      - {{.ESEndpoint}}
    username: admin
    password: testing
    index: {{.Index}}
    compression_level: 0
    max_retries: {{.MaxRetries}}
logging.level: debug
queue.mem.flush.timeout: 0s
setup.template.enabled: false
http.enabled: true
http.host: localhost
http.port: {{.MonitoringPort}}
`
			var configBuffer bytes.Buffer
			require.NoError(t,
				template.Must(template.New("config").Parse(cfg)).Execute(&configBuffer, beatsConfig))

			filebeatOTel.WriteConfigFile(configBuffer.String())
			writeEventsToLogFile(t, beatsConfig.InputFile, numTestEvents)
			filebeatOTel.Start()
			defer filebeatOTel.Stop()

			// Wait for file input to be fully read
			filebeatOTel.WaitStdErrContains(fmt.Sprintf("End of file reached: %s; Backoff now.", beatsConfig.InputFile), 30*time.Second)

			// Wait for expected events to be ingested
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				mu.Lock()
				defer mu.Unlock()

				// collect mock-es metrics
				rm := metricdata.ResourceMetrics{}
				err := reader.Collect(context.Background(), &rm)
				assert.NoError(ct, err, "failed to collect metrics from mock-es")
				metrics := make(map[string]int64)
				for _, sm := range rm.ScopeMetrics {
					for _, m := range sm.Metrics {
						if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
							var total int64
							for _, dp := range sum.DataPoints {
								total += dp.Value
							}
							metrics[m.Name] = total
						}
					}
				}
				assert.Equal(ct, int64(len(tt.expectedIngestedEventIDs)), metrics["bulk.create.ok"], "expected bulk.create.ok metric to match ingested events")

				// If we have the right count, validate the specific events
				// Verify we have the correct events ingested
				for _, expectedID := range tt.expectedIngestedEventIDs {
					expectedEventKey := fmt.Sprintf("Line %d", expectedID)
					found := false
					for _, ingested := range ingestedTestEvents {
						if ingested == expectedEventKey {
							found = true
							break
						}
					}
					assert.True(ct, found, "expected _bulk event %s to be ingested", expectedEventKey)
				}

				// Verify we have valid line content for all ingested events
				for _, ingested := range ingestedTestEvents {
					assert.Regexp(ct, `^Line \d+$`, ingested, "unexpected ingested event format: %s", ingested)
				}
			}, 30*time.Second, 1*time.Second, "timed out waiting for expected event processing")

			// Confirm filebeat agreed with our accounting of ingested events
			require.EventuallyWithT(t, func(ct *assert.CollectT) {
				address := fmt.Sprintf("http://localhost:%d", beatsConfig.MonitoringPort)
				r, err := http.Get(address + "/stats") //nolint:noctx,bodyclose // fine for tests
				assert.NoError(ct, err)
				assert.Equal(ct, http.StatusOK, r.StatusCode, "incorrect status code")
				var m mapstr.M
				err = json.NewDecoder(r.Body).Decode(&m)
				assert.NoError(ct, err)

				m = m.Flatten()

				// Currently, otelconsumer either ACKs or fails the entire batch and has no visibility into individual event failures within the exporter.
				// From otelconsumer's perspective, the whole batch is considered successful as long as ConsumeLogs returns no error.
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.total"], "expected total events sent to output to match")
				assert.Equal(ct, float64(numTestEvents), m["libbeat.output.events.acked"], "expected total events acked to match")
				assert.Equal(ct, float64(0), m["libbeat.output.events.dropped"], "expected total events dropped to match")
			}, 10*time.Second, 100*time.Millisecond, "expected output stats to be available in monitoring endpoint")
		})
	}
}
