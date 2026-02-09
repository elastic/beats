// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package cel

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/otel"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func TestOTELCELMetrics(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	client := testServer.Client()

	// Set up the otelCELMetrics
	log := logp.NewLogger("cel_metrics_test")
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("test-service"),
	)

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		// Copy the pipe's reader to our buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			t.Errorf("Error copying pipe to buffer: %v", err)
		}
		close(done)
	}()

	metricExporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithTemporalitySelector(otel.DeltaSelector),
		stdoutmetric.WithEncoder(&otel.ConcurrentEncoder{Encoder: json.NewEncoder(w)}))
	if err != nil {
		t.Fatalf("failed to create exporter: %v", err)
	}

	otelCELMetrics, transport, err := newOTELCELMetrics(log, *resource, client.Transport, metricExporter)
	if err != nil {
		t.Fatalf("failed to create otelCELMetrics: %v", err)
	}
	ctx := context.Background()
	defer otelCELMetrics.Shutdown(ctx)

	reg := monitoring.NewRegistry()

	inputMetrics, _ := newInputMetrics(reg, log)

	mRecorder, err := newMetricsRecorder(inputMetrics, otelCELMetrics)
	if err != nil {
		t.Fatalf("failed to create metrics recorder: %v", err)
	}
	// Create an HTTP client using the otelCELMetrics transport
	client.Transport = transport

	var totalCelDuration time.Duration
	var totalPublishDuration time.Duration
	var totalPeriodicRunDuration time.Duration
	// mock a cel periodic run
	mRecorder.StartPeriodic(ctx)
	for index := range 5 {
		startProgram := time.Now()
		mRecorder.AddProgramExecution(ctx)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		resp.Body.Close()
		celDuration := time.Since(startProgram)
		totalCelDuration = totalCelDuration + celDuration
		mRecorder.AddCELDuration(ctx, celDuration)
		mRecorder.AddProgramSuccessExecution(ctx)
		mRecorder.AddReceivedEvents(ctx, uint(index+1)) //nolint:gosec // disable G115
		mRecorder.AddReceivedBatch(ctx, 1)
		startPublish := time.Now()
		time.Sleep(100 * time.Millisecond)
		mRecorder.AddPublishedBatch(ctx, 1)
		mRecorder.AddPublishedEvents(ctx, uint(index+1)) //nolint:gosec // disable G115
		publishDuration := time.Since(startPublish)
		totalPublishDuration = totalPublishDuration + publishDuration
		mRecorder.AddPublishDuration(ctx, publishDuration)
		totalDuration := time.Since(startProgram)
		totalPeriodicRunDuration = totalPeriodicRunDuration + totalDuration
		mRecorder.AddProgramRunDuration(ctx, totalDuration)
	}
	mRecorder.EndPeriodic(ctx)

	time.Sleep(500 * time.Millisecond)
	w.Close()
	<-done

	// Check for presence of expected OTEL metrics
	expectedMetricNames := []string{
		"input.cel.periodic.run",
		"input.cel.periodic.program.run.started",
		"input.cel.periodic.program.run.success",
		"input.cel.periodic.batch.received",
		"input.cel.periodic.batch.published",
		"input.cel.periodic.event.received",
		"input.cel.periodic.event.published",
		"input.cel.periodic.run.duration",
		"input.cel.periodic.cel.duration",
		"input.cel.periodic.event.publish.duration",
		"input.cel.program.batch.received",
		"input.cel.program.batch.published",
		"input.cel.program.event.received",
		"input.cel.program.event.published",
		"input.cel.program.run.duration",
		"input.cel.program.cel.duration",
		"input.cel.program.publish.duration",
		"http.client.request.body.size",
		"http.client.request.duration",
	}

	// if the metric is empty, it does not export.
	output := buf.String()
	notFound := []string{}
	for _, metricName := range expectedMetricNames {
		if !strings.Contains(output, metricName) {
			notFound = append(notFound, metricName)
		}
	}

	if len(notFound) != 0 {
		t.Errorf("expected all metrics to be found, but missing: %v", notFound)
	}

	// check that inputMetrics are incremented
	if inputMetrics.executions.Get() != uint64(5) {
		t.Errorf("executions = %v, want %v", inputMetrics.executions.Get(), uint64(5))
	}
	if inputMetrics.eventsReceived.Get() != uint64(15) {
		t.Errorf("eventsReceived = %v, want %v", inputMetrics.eventsReceived.Get(), uint64(15))
	}
	if inputMetrics.batchesReceived.Get() != uint64(5) {
		t.Errorf("batchesReceived = %v, want %v", inputMetrics.batchesReceived.Get(), uint64(5))
	}
	if inputMetrics.eventsPublished.Get() != uint64(15) {
		t.Errorf("eventsPublished = %v, want %v", inputMetrics.eventsPublished.Get(), uint64(15))
	}
	if inputMetrics.batchesPublished.Get() != uint64(5) {
		t.Errorf("batchesPublished = %v, want %v", inputMetrics.batchesPublished.Get(), uint64(5))
	}
	if inputMetrics.celProcessingTime.Count() != int64(5) {
		t.Errorf("celProcessingTime.Count() = %v, want %v", inputMetrics.celProcessingTime.Count(), int64(5))
	}
	if inputMetrics.celProcessingTime.Sum() != totalCelDuration.Nanoseconds() {
		t.Errorf("celProcessingTime.Sum() = %v, want %v", inputMetrics.celProcessingTime.Sum(), totalCelDuration.Nanoseconds())
	}
}

// inMemoryExporter is a metrics exporter that stores metrics in memory for testing.
type inMemoryExporter struct {
	mu      sync.Mutex
	metrics []metricdata.ResourceMetrics
}

var _ sdkmetric.Exporter = (*inMemoryExporter)(nil)

func (e *inMemoryExporter) Export(_ context.Context, metrics *metricdata.ResourceMetrics) error {
	e.mu.Lock()
	e.metrics = append(e.metrics, *metrics)
	e.mu.Unlock()
	return nil
}

func (e *inMemoryExporter) Shutdown(context.Context) error     { return nil }
func (e *inMemoryExporter) ForceFlush(_ context.Context) error { return nil }

func (e *inMemoryExporter) Temporality(_ sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

func (e *inMemoryExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}

func (e *inMemoryExporter) getMetrics() []metricdata.ResourceMetrics {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.metrics
}

// testPublisher is a publisher that signals when events are published.
type testPublisher struct {
	mu        sync.Mutex
	published []beat.Event
	cursors   []map[string]any
	eventCh   chan struct{} // signals when an event is published
}

func (p *testPublisher) Publish(e beat.Event, cursor any) error {
	p.mu.Lock()
	p.published = append(p.published, e)
	if cursor != nil {
		if c, ok := cursor.(map[string]any); ok {
			p.cursors = append(p.cursors, c)
		}
	}
	p.mu.Unlock()

	// Signal that an event was published (non-blocking).
	select {
	case p.eventCh <- struct{}{}:
	default:
	}
	return nil
}

// TestDegradedRunDoesNotCountAsSuccess verifies that when a CEL program returns
// an error object (causing a degraded state), the success metric is not incremented.
// This is a regression test for https://github.com/elastic/beats/issues/48714.
func TestDegradedRunDoesNotCountAsSuccess(t *testing.T) {
	// Setup InMemory Exporter
	exporter := &inMemoryExporter{}
	otel.GetGlobalMetricsExporterFactory().SetGlobalMetricsExporter(exporter)
	defer otel.GetGlobalMetricsExporterFactory().SetGlobalMetricsExporter(nil)

	// Configure input with a program that returns a single error object in "events",
	// which causes isDegraded=true but flows to publication as an event.
	configMap := map[string]any{
		"interval": "1h", // Long interval since we only need one run
		"program":  `{"events": {"error": "simulated failure"}}`,
		"resource": map[string]any{
			"url": "http://localhost",
		},
	}
	cfg := conf.MustNewConfigFrom(configMap)
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		t.Fatalf("failed to unpack config: %v", err)
	}

	src := &source{config}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	v2Ctx := v2.Context{
		Logger:          logp.NewLogger("cel_test"),
		ID:              "test_degraded_run",
		IDWithoutName:   "test_degraded_run",
		Cancelation:     ctx,
		MetricsRegistry: monitoring.NewRegistry(),
	}

	// Create publisher with a channel to signal when events are published.
	eventPublished := make(chan struct{}, 1)
	pub := &testPublisher{
		eventCh: eventPublished,
	}

	// Run input in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- input{}.run(v2Ctx, src, nil, pub, &v2Ctx)
	}()

	// Wait for at least one event to be published, or timeout.
	select {
	case <-eventPublished:
		// Event published, the program ran and we can check metrics.
	case <-time.After(30 * time.Second):
		t.Fatal("Timed out waiting for event to be published")
	}

	// Cancel context to stop the input.
	cancel()
	<-errCh

	// Check that success count is 0 (degraded runs should not count as success).
	var successCount int64
	for _, resourceMetrics := range exporter.getMetrics() {
		for _, scopeMetrics := range resourceMetrics.ScopeMetrics {
			for _, metric := range scopeMetrics.Metrics {
				if metric.Name == "input.cel.periodic.program.run.success" {
					if data, ok := metric.Data.(metricdata.Sum[int64]); ok {
						for _, dp := range data.DataPoints {
							successCount += dp.Value
						}
					}
				}
			}
		}
	}

	if successCount != 0 {
		t.Errorf("Expected 0 successes for degraded program run, got %d", successCount)
	}
}
