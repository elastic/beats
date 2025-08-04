// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package etw

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	regenerateGolden = flag.Bool("regenerate-golden", false, "regenerate the golden file for comparison tests")
)

// createTestSessionConfig creates a standard session configuration for tests
func createTestSessionConfig(sessionName string) Config {
	return Config{
		Providers: []ProviderConfig{
			{
				GUID:            testProviderGUID,
				Name:            testProviderName,
				TraceLevel:      "verbose",
				MatchAnyKeyword: 0xFFFFFFFFFFFFFFFF,
			},
		},
		SessionName:    sessionName,
		BufferSize:     1024,
		MinimumBuffers: 10,
		MaximumBuffers: 10,
	}
}

// createETLSessionConfig creates a session configuration for ETL file reading
func createETLSessionConfig(sessionName string) Config {
	return Config{
		Logfile:     "./testdata/sample-test-events.etl",
		SessionName: sessionName,
	}
}

// ETW Callback Measurer for benchmarks

// ETWCallbackMeasurer counts ETW callback invocations safely
type ETWCallbackMeasurer struct {
	count int64
}

// InjectableCallback returns a callback function that measures event processing performance
func (m *ETWCallbackMeasurer) InjectableCallback(session *Session) func(record *EventRecord) uintptr {
	return func(record *EventRecord) uintptr {
		_, err := session.RenderEvent(record)
		if err == nil || errors.Is(err, ErrUnprocessableEvent) {
			atomic.AddInt64(&m.count, 1)
		}
		return 0
	}
}

// Reset resets the counter for a new measurement
func (m *ETWCallbackMeasurer) Reset() {
	atomic.StoreInt64(&m.count, 0)
}

// GetCount returns the current count safely
func (m *ETWCallbackMeasurer) GetCount() int64 {
	return atomic.LoadInt64(&m.count)
}

// Test Infrastructure

// isRunningAsAdmin checks if the current process has administrator privileges
func isRunningAsAdmin() bool {
	cmd := exec.CommandContext(context.Background(), "net", "session")
	return cmd.Run() == nil
}

// TestMain ensures proper setup and cleanup for benchmarks
func TestMain(m *testing.M) {
	if runtime.GOOS == "windows" && !isRunningAsAdmin() {
		log.Printf("Warning: ETW benchmarks require administrator privileges")
		log.Printf("Please run as administrator for full benchmark functionality")
	}

	code := m.Run()

	os.Exit(code)
}

// Benchmark Tests

// BenchmarkETWCallbackRate tests the ETW callback rate performance
func BenchmarkETWCallbackRate(b *testing.B) {
	setupProviderManager(b)

	// Configure event generation
	eventsPerBatch := 300
	batchInterval := 10 * time.Millisecond

	// Set up callback measurer
	measurer := &ETWCallbackMeasurer{}

	// Create session and start event generation
	session, generator, consumerDone := setupBenchmarkSession(b, measurer.InjectableCallback)

	// Create a separate channel for stopping event generation
	stopEventGeneration := make(chan struct{})
	eventGenDone := startEventGeneration(generator, eventsPerBatch, batchInterval, stopEventGeneration)

	b.Cleanup(func() {
		close(stopEventGeneration)
		cleanupBenchmarkSession(b, session, generator, consumerDone, eventGenDone)
	})

	b.ResetTimer()

	// Run benchmark iterations
	for j := 0; j < b.N; j++ {
		runBenchmarkIteration(b, measurer, eventsPerBatch, batchInterval)
	}
}

// setupBenchmarkSession creates and configures an ETW session for benchmarking
func setupBenchmarkSession(b *testing.B, callbackFactory func(session *Session) func(record *EventRecord) uintptr) (*Session, *ETWEventGenerator, chan struct{}) {
	sessionConfig := createTestSessionConfig(uniqueSessionName("BenchmarkETW"))

	session, err := NewSession(sessionConfig)
	if err != nil {
		b.Fatalf("Failed to create ETW session: %v", err)
	}

	// Set up the callback using the factory
	session.Callback = callbackFactory(session)

	// Create the real-time ETW session
	if err := session.CreateRealtimeSession(); err != nil {
		b.Fatalf("Failed to create realtime ETW session: %v", err)
	}

	// Create event generator
	generator, err := NewETWEventGenerator()
	if err != nil {
		b.Fatalf("Failed to create event generator: %v", err)
	}

	// Start session consumer
	consumerDone := make(chan struct{})
	consumerError := make(chan error, 1)
	go func() {
		defer close(consumerDone)
		if err := session.StartConsumer(); err != nil {
			consumerError <- err
		}
	}()

	// Wait for consumer to be ready and check for errors
	time.Sleep(5 * time.Second)
	select {
	case err := <-consumerError:
		b.Fatalf("Consumer failed to start: %v", err)
	default:
	}

	return session, generator, consumerDone
}

// startEventGeneration starts the event generator in a separate goroutine and returns the done channel
func startEventGeneration(generator *ETWEventGenerator, eventsPerBatch int, batchInterval time.Duration, stopEventGeneration chan struct{}) chan struct{} {
	eventGenDone := make(chan struct{})
	go generator.StartGenerating(eventsPerBatch, batchInterval, stopEventGeneration, eventGenDone)
	return eventGenDone
}

// runBenchmarkIteration runs a single benchmark iteration with performance measurements
func runBenchmarkIteration(b *testing.B, measurer *ETWCallbackMeasurer, eventsPerBatch int, batchInterval time.Duration) {
	measurer.Reset()

	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	b.StartTimer()
	startTime := time.Now()

	time.Sleep(1 * time.Second) // Process events for 1 second

	endTime := time.Now()
	b.StopTimer()

	time.Sleep(200 * time.Millisecond) // Allow ETW processing to catch up

	// Calculate and report metrics
	callsInThisIteration := measurer.GetCount()
	actualDuration := endTime.Sub(startTime)

	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	reportBenchmarkMetrics(b, callsInThisIteration, actualDuration, memBefore, memAfter, eventsPerBatch, batchInterval)
}

// reportBenchmarkMetrics reports performance metrics for the benchmark
func reportBenchmarkMetrics(b *testing.B, callsInThisIteration int64, actualDuration time.Duration, memBefore, memAfter runtime.MemStats, eventsPerBatch int, batchInterval time.Duration) {
	callsPerSecond := float64(callsInThisIteration) / actualDuration.Seconds()
	expectedEventsPerSecond := float64(eventsPerBatch) / batchInterval.Seconds()

	b.ReportMetric(callsPerSecond, "events/s")
	b.ReportMetric(float64(callsInThisIteration), "total_events_iter")
	b.ReportMetric(expectedEventsPerSecond, "target_events/s")
	b.ReportMetric(actualDuration.Seconds(), "duration_s")

	// Memory metrics
	bytesAllocated := memAfter.TotalAlloc - memBefore.TotalAlloc
	allocations := memAfter.Mallocs - memBefore.Mallocs

	if callsInThisIteration > 0 {
		b.ReportMetric(float64(bytesAllocated)/float64(callsInThisIteration), "bytes/event")
		b.ReportMetric(float64(allocations)/float64(callsInThisIteration), "allocs/event")
	}

	b.ReportMetric(float64(bytesAllocated), "total_bytes_iter")
	b.ReportMetric(float64(allocations), "total_allocs_iter")
}

// cleanupBenchmarkSession properly cleans up benchmark session resources
func cleanupBenchmarkSession(b *testing.B, session *Session, generator *ETWEventGenerator, consumerDone chan struct{}, eventGenDone chan struct{}) {
	// Wait for event generator to finish
	if eventGenDone != nil {
		select {
		case <-eventGenDone:
		case <-time.After(10 * time.Second):
			b.Logf("Timeout waiting for event generator to exit.")
		}
	}

	if err := generator.close(); err != nil {
		b.Logf("Failed to close event generator: %v", err)
	}

	if err := session.StopSession(); err != nil {
		b.Logf("Failed to stop ETW session: %v", err)
	}

	// Wait for consumer goroutine to finish
	select {
	case <-consumerDone:
	case <-time.After(time.Minute):
		b.Logf("Warning: Session didn't stop within timeout")
	}
}

// Golden File Tests

// TestETLGoldenFile validates ETL parsing data quality and compares results against a golden file
func TestETLGoldenFile(t *testing.T) {
	setupProviderManager(t)

	// Create and configure session for ETL file reading
	sessionConfig := createETLSessionConfig(uniqueSessionName("GoldenTestETW"))
	session, err := NewSession(sessionConfig)
	if err != nil {
		t.Fatalf("Failed to create ETW session: %v", err)
	}

	// Collect events from ETL file
	events := collectEventsFromETL(t, session)

	// Validate event data quality
	validateEventDataQuality(t, events)

	// Compare with golden file
	compareWithGoldenFile(t, events)
}

// collectEventsFromETL processes an ETL file and returns all rendered events
func collectEventsFromETL(t *testing.T, session *Session) []RenderedEtwEvent {
	var events []RenderedEtwEvent
	var eventCount int

	session.Callback = func(record *EventRecord) uintptr {
		event, err := session.RenderEvent(record)
		if err != nil {
			if errors.Is(err, ErrUnprocessableEvent) {
				return 0
			}
			t.Errorf("Failed to render event: %v", err)
			return 1
		}
		events = append(events, event)
		eventCount++
		return 0
	}

	// Start consumer
	consumerDone := make(chan struct{})
	consumerError := make(chan error, 1)
	go func() {
		defer close(consumerDone)
		if err := session.StartConsumer(); err != nil {
			consumerError <- err
		}
	}()

	time.Sleep(time.Second)

	select {
	case err := <-consumerError:
		t.Fatalf("Consumer failed to start: %v", err)
	default:
	}

	t.Cleanup(func() {
		if err := session.StopSession(); err != nil {
			t.Logf("Failed to stop ETW session: %v", err)
		}
	})

	// Wait for events to be processed
	for eventCount == 0 {
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(2 * time.Second)

	if len(events) == 0 {
		t.Fatal("No events were processed from the ETL file")
	}

	t.Logf("Processed %d events from ETL file for golden file comparison", len(events))
	return events
}

// validateEventDataQuality checks for data quality issues in events
func validateEventDataQuality(t *testing.T, events []RenderedEtwEvent) {
	var totalBadProperties int

	for eventIdx, event := range events {
		for propIdx, prop := range event.Properties {
			if str, ok := prop.Value.(string); ok && strings.Contains(str, "Complex/Unsupported property") {
				t.Errorf("Event %d Property %d (%s) should be parsed correctly, but got: %s",
					eventIdx, propIdx, prop.Name, str)
				totalBadProperties++
			}

			if str, ok := prop.Value.(string); ok {
				for _, r := range str {
					if r > 127 && (r < 0x20 || (r >= 0x2000 && r <= 0x206F)) {
						t.Errorf("Event %d Property %d (%s) contains invalid Unicode character U+%04X: %s",
							eventIdx, propIdx, prop.Name, r, str)
						totalBadProperties++
						break
					}
				}
			}

			totalBadProperties += checkForBadDataRecursive(prop.Value, eventIdx, propIdx, prop.Name, t)
		}
	}

	if totalBadProperties > 0 {
		t.Errorf("Found %d properties with bad data across %d events", totalBadProperties, len(events))
	}
}

// compareWithGoldenFile compares events with the golden file
func compareWithGoldenFile(t *testing.T, events []RenderedEtwEvent) {
	normalizedEvents := normalizeEventsForComparison(events)
	goldenPath := filepath.Join("testdata", "golden_events.json")

	if *regenerateGolden {
		regenerateGoldenFile(t, normalizedEvents, goldenPath)
		return
	}

	expectedEvents := loadGoldenFile(t, goldenPath)
	compareEvents(t, normalizedEvents, expectedEvents)
}

// regenerateGoldenFile creates a new golden file with the provided events
func regenerateGoldenFile(t *testing.T, events []RenderedEtwEvent, goldenPath string) {
	goldenData, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal events to JSON: %v", err)
	}

	if err := os.WriteFile(goldenPath, goldenData, 0644); err != nil {
		t.Fatalf("Failed to write golden file: %v", err)
	}

	t.Logf("Golden file regenerated at: %s", goldenPath)
	t.Logf("Golden file contains %d normalized events", len(events))
}

// loadGoldenFile loads and parses the golden file
func loadGoldenFile(t *testing.T, goldenPath string) []RenderedEtwEvent {
	goldenData, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Failed to read golden file %s: %v\nRun with -regenerate-golden to create it", goldenPath, err)
	}

	var expectedEvents []RenderedEtwEvent
	if err := json.Unmarshal(goldenData, &expectedEvents); err != nil {
		t.Fatalf("Failed to unmarshal golden file: %v", err)
	}

	return expectedEvents
}

// compareEvents compares actual and expected events
func compareEvents(t *testing.T, actualEvents, expectedEvents []RenderedEtwEvent) {
	if len(actualEvents) != len(expectedEvents) {
		t.Fatalf("Event count mismatch: got %d, expected %d", len(actualEvents), len(expectedEvents))
	}

	var differences []string
	for i, actualEvent := range actualEvents {
		expectedEvent := expectedEvents[i]
		if diffs := compareRenderedEvents(actualEvent, expectedEvent, i); len(diffs) > 0 {
			differences = append(differences, diffs...)
		}
	}

	if len(differences) > 0 {
		t.Errorf("Found %d differences between actual and expected events:", len(differences))
		for _, diff := range differences {
			t.Errorf("  %s", diff)
		}
		t.Errorf("If the differences are expected, regenerate the golden file with -regenerate-golden")
	} else {
		t.Logf("All %d events match the golden file", len(actualEvents))
	}
}

// Event Normalization and Comparison Helpers

// normalizeEventsForComparison removes time-sensitive and non-deterministic fields
func normalizeEventsForComparison(events []RenderedEtwEvent) []RenderedEtwEvent {
	var normalized []RenderedEtwEvent

	for _, event := range events {
		normalizedEvent := event

		// Clear time-sensitive fields
		normalizedEvent.Timestamp = time.Time{}

		// Sort keywords for consistent comparison
		sort.Strings(normalizedEvent.Keywords)

		normalized = append(normalized, normalizedEvent)
	}

	return normalized
}

// compareRenderedEvents compares two RenderedEtwEvent structs and returns differences
func compareRenderedEvents(actual, expected RenderedEtwEvent, eventIndex int) []string {
	var differences []string

	// Compare basic event fields
	if actual.EventID != expected.EventID {
		differences = append(differences, fmt.Sprintf("Event %d: EventID mismatch: got %d, expected %d", eventIndex, actual.EventID, expected.EventID))
	}
	if actual.Level != expected.Level {
		differences = append(differences, fmt.Sprintf("Event %d: Level mismatch: got %s, expected %s", eventIndex, actual.Level, expected.Level))
	}
	if actual.Task != expected.Task {
		differences = append(differences, fmt.Sprintf("Event %d: Task mismatch: got %s, expected %s", eventIndex, actual.Task, expected.Task))
	}
	if actual.Opcode != expected.Opcode {
		differences = append(differences, fmt.Sprintf("Event %d: Opcode mismatch: got %s, expected %s", eventIndex, actual.Opcode, expected.Opcode))
	}

	// Compare keywords
	if len(actual.Keywords) != len(expected.Keywords) {
		differences = append(differences, fmt.Sprintf("Event %d: Keywords count mismatch: got %d, expected %d", eventIndex, len(actual.Keywords), len(expected.Keywords)))
	} else {
		for i, keyword := range actual.Keywords {
			if keyword != expected.Keywords[i] {
				differences = append(differences, fmt.Sprintf("Event %d: Keyword %d mismatch: got %s, expected %s", eventIndex, i, keyword, expected.Keywords[i]))
			}
		}
	}

	// Compare properties
	if len(actual.Properties) != len(expected.Properties) {
		differences = append(differences, fmt.Sprintf("Event %d: Properties count mismatch: got %d, expected %d", eventIndex, len(actual.Properties), len(expected.Properties)))
	} else {
		for i, prop := range actual.Properties {
			expectedProp := expected.Properties[i]
			if prop.Name != expectedProp.Name {
				differences = append(differences, fmt.Sprintf("Event %d Property %d: Name mismatch: got %s, expected %s", eventIndex, i, prop.Name, expectedProp.Name))
			}
			if !comparePropertyValues(prop.Value, expectedProp.Value) {
				differences = append(differences, fmt.Sprintf("Event %d Property %d (%s): Value mismatch: got %v, expected %v", eventIndex, i, prop.Name, prop.Value, expectedProp.Value))
			}
		}
	}

	return differences
}

// comparePropertyValues compares two property values recursively
func comparePropertyValues(actual, expected interface{}) bool {
	actualJSON, err1 := json.Marshal(actual)
	expectedJSON, err2 := json.Marshal(expected)
	return err1 == nil && err2 == nil && string(actualJSON) == string(expectedJSON)
}

// checkForBadDataRecursive recursively checks for bad data in nested structures
func checkForBadDataRecursive(value any, eventIdx, propIdx int, propName string, t *testing.T) int {
	badCount := 0
	switch v := value.(type) {
	case string:
		if strings.Contains(v, "Complex/Unsupported property") {
			t.Errorf("Event %d Property %d (%s) nested value contains Complex/Unsupported property: %s", eventIdx, propIdx, propName, v)
			badCount++
		}
		for _, r := range v {
			if r > 127 && (r < 0x20 || (r >= 0x2000 && r <= 0x206F)) {
				t.Errorf("Event %d Property %d (%s) nested value contains invalid Unicode character U+%04X: %s", eventIdx, propIdx, propName, r, v)
				badCount++
				break
			}
		}
	case []any:
		for i, item := range v {
			badCount += checkForBadDataRecursive(item, eventIdx, propIdx, fmt.Sprintf("%s[%d]", propName, i), t)
		}
	case map[string]any:
		for key, val := range v {
			badCount += checkForBadDataRecursive(val, eventIdx, propIdx, fmt.Sprintf("%s.%s", propName, key), t)
		}
	}
	return badCount
}
