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
	"testing"
	"time"

	"github.com/elastic/beats/v7/x-pack/filebeat/otel"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"

	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
		defer resp.Body.Close()
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
