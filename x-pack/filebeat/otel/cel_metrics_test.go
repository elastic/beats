// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package otel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/stretchr/testify/assert"
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

	// Set up the OTELCELMetrics
	log := logp.NewLogger("cel_metrics_test")
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("test-service"),
	)

	var buf bytes.Buffer
	metricExporter, err := stdoutmetric.New(stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithTemporalitySelector(DeltaSelector),
		stdoutmetric.WithEncoder(NewConcurentEncoder(json.NewEncoder(&buf))))

	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}

	otelCELMetrics, transport, err := NewOTELCELMetrics(log, testServer.URL, *resource, client.Transport, metricExporter)
	if err != nil {
		t.Fatalf("Failed to create OTELCELMetrics: %v", err)
	}

	// Create an HTTP client using the OTELCELMetrics transport
	client.Transport = transport

	// mock a cel periodic run
	ctx := context.Background()
	otelCELMetrics.AddPeriodicRun(ctx, 1)
	otelCELMetrics.StartPeriodic()
	for index := range 5 {
		startProgram := time.Now()
		otelCELMetrics.AddProgramExecution(ctx, 1)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, testServer.URL, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()
		otelCELMetrics.AddCELDuration(ctx, time.Since(startProgram))
		otelCELMetrics.AddProgramSuccessExecution(ctx, 1)
		otelCELMetrics.AddEvents(ctx, int64(index))
		otelCELMetrics.AddGeneratedBatch(ctx, 1)
		startPublish := time.Now()
		time.Sleep(100 * time.Millisecond)
		otelCELMetrics.AddPublishedBatch(ctx, 1)
		otelCELMetrics.AddPublishedEvents(ctx, int64(index))
		otelCELMetrics.AddPublishDuration(ctx, time.Since(startPublish))
		otelCELMetrics.AddTotalDuration(ctx, time.Since(startProgram))
	}
	otelCELMetrics.EndPeriodic(ctx)

	// exporter is async, wait until the byte buffer size is > 0 and stable in size
	checkStart := time.Now()
	for buf.Len() == 0 {
		if time.Since(checkStart) > 5*time.Second {
			t.Fatalf("output was not generated correctly buf: %s", buf.String())
		}
		time.Sleep(1 * time.Millisecond)
	}
	len1 := buf.Len()
	len2 := 0

	for len1 != len2 && time.Since(checkStart) < 5*time.Second {
		if time.Since(checkStart) > 5*time.Second {
			t.Fatal("output generated but not stable")
		}
		len2 = len1
		len1 = buf.Len()
	}

	// Check for presence of expected OTEL metrics
	expectedMetricNames := []string{
		"input.cel.periodic.run",
		"input.cel.periodic.program.run.started",
		"input.cel.periodic.program.run.success",
		"input.cel.periodic.batch.generated",
		"input.cel.periodic.batch.published",
		"input.cel.periodic.event.generated",
		"input.cel.periodic.event.published",
		"input.cel.periodic.run.duration",
		"input.cel.periodic.cel.duration",
		"input.cel.periodic.event.publish.duration",
		"input.cel.program.batch.processed",
		"input.cel.program.batch.published",
		"input.cel.program.event.generated",
		"http.client.request.body.size",
		"http.client.request.duration",
	}

	output := buf.String()
	notFound := []string{}
	for _, metricName := range expectedMetricNames {
		if !strings.Contains(output, metricName) {
			notFound = append(notFound, metricName)
		}
	}

	assert.Equal(t, 0, len(notFound), notFound)
}
