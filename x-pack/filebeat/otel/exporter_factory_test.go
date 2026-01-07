// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"os"
	"sync"
	"testing"
)

func TestExporterFactoryRace(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	factory := NewMetricsExporterFactory(GetDefaultMetricExporterOptions())

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			factory.GetExporter(context.Background(), true) // global=true
		}()
	}
	wg.Wait()
}

func TestGetGlobalExporterFactory(t *testing.T) {
	// set global to false
	factory1 := GetGlobalMetricsExporterFactory()
	if factory1 == nil {
		t.Errorf("Expected non-nil factory, got nil")
	}

	factory2 := GetGlobalMetricsExporterFactory()
	if factory1 != factory2 {
		t.Errorf("Expected same factory instance, got different instances")
	}
}

func TestExporterFactory(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	options := GetDefaultMetricExporterOptions()
	factory := NewMetricsExporterFactory(options)
	if factory == nil {
		t.Errorf("Expected non-nil factory")
	}
	exporter1, etype1, err := factory.GetExporter(context.Background(), false)
	if err != nil {
		t.Errorf("GetExporter returned error: %v", err)
	}
	if exporter1 == nil {
		t.Errorf("exporter1 is nil")
	}
	if etype1 == "" {
		t.Errorf("etype1 is empty")
	}
	exporter2, etype2, err := factory.GetExporter(context.Background(), true)
	if err != nil {
		t.Errorf("GetExporter returned error: %v", err)
	}
	if exporter2 == nil {
		t.Errorf("exporter2 is nil")
	}
	if etype2 == "" {
		t.Errorf("etype2 is empty")
	}
	exporter3, etype3, err := factory.GetExporter(context.Background(), true)
	if err != nil {
		t.Errorf("GetExporter returned error: %v", err)
	}
	if exporter3 == nil {
		t.Errorf("exporter3 is nil")
	}
	if etype3 == "" {
		t.Errorf("etype3 is empty")
	}
	if etype1 != etype2 {
		t.Errorf("etype1 = %v, want %v", etype1, etype2)
	}
	if exporter1 == exporter2 {
		t.Errorf("exporter1 and exporter2 should be different")
	}
	if exporter2 != exporter3 {
		t.Errorf("exporter2 = %v, want %v", exporter2, exporter3)
	}
}

func TestExporterFactoryNoMetricsEnvironment(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	options := GetDefaultMetricExporterOptions()
	factory := NewMetricsExporterFactory(options)
	if factory == nil {
		t.Errorf("Expected non-nil factory")
	}
	exporter, etype, err := factory.GetExporter(context.Background(), false)
	if err != nil {
		t.Errorf("GetExporter returned error: %v", err)
	}
	if exporter == nil {
		t.Errorf("exporter is nil")
	}
	if etype == "" {
		t.Errorf("etype is empty")
	}
	if etype != ExporterType("grpc") {
		t.Errorf("etype = %v, want %v", etype, ExporterType("grpc"))
	}
}

func TestGetGlobalExporterNoneType(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	factory := GetGlobalMetricsExporterFactory()
	exporter, _, err := factory.GetExporter(context.Background(), true)
	if err != nil {
		t.Errorf("GetExporter returned an error: %v", err)
	}
	if exporter == nil {
		t.Errorf("Exporter should not be nil")
	}
	if exporter != factory.globalMetricsExporter {
		t.Errorf("exporter = %v, want %v", exporter, factory.globalMetricsExporter)
	}
}

func TestGetGlobalExporterGRPCType(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	factory := GetGlobalMetricsExporterFactory()
	exporter, _, err := factory.GetExporter(context.Background(), true)
	if err != nil {
		t.Errorf("GetExporter returned an error: %v", err)
	}
	if exporter == nil {
		t.Errorf("Exporter should not be nil")
	}
	if exporter != factory.globalMetricsExporter {
		t.Errorf("exporter = %v, want %v", exporter, factory.globalMetricsExporter)
	}
}

func TestGetExporterFromEnvironment(t *testing.T) {
	exporterFactory = nil
	factory := NewMetricsExporterFactory(GetDefaultMetricExporterOptions())

	tests := []struct {
		name            string
		metricsExporter string
		protocol        string
		endpoint        string
		eType           string
		isNil           bool
	}{
		{name: "console override endpoint and protocol being set", metricsExporter: "console", eType: "console", protocol: "set", endpoint: "set"},
		{name: "console override endpoint being set", metricsExporter: "console", eType: "console", protocol: "", endpoint: "set"},
		{name: "console, protocol being set", metricsExporter: "console", eType: "console", protocol: "set", endpoint: ""},
		{name: "console", metricsExporter: "console", eType: "console", protocol: "", endpoint: ""},
		{name: "prometheus is none", metricsExporter: "prometheus", eType: "none", protocol: "set", endpoint: "set", isNil: true},
		{name: "logging is none", metricsExporter: "logging", eType: "none", protocol: "set", endpoint: "set", isNil: true},
		{name: "none", metricsExporter: "none", eType: "none", protocol: "set", endpoint: "set", isNil: true},
		{name: "none nothing set", metricsExporter: "", eType: "none", protocol: "", endpoint: "", isNil: true},
		{name: "none empty exporter", metricsExporter: "", eType: "none", protocol: "", endpoint: "", isNil: true},
		{name: "none otlp but no other environment set", metricsExporter: "otlp", eType: "none", protocol: "", endpoint: "", isNil: true},
		{name: "otlp with endpoint set", metricsExporter: "otlp", eType: "grpc", protocol: "", endpoint: "set"},
		{name: "metrics exporter falls through to grpc", metricsExporter: "anything", eType: "grpc", protocol: "", endpoint: "set"},
		{name: "grpc default", metricsExporter: "", eType: "grpc", protocol: "", endpoint: "set"},
		{name: "grpc explicit", metricsExporter: "", eType: "grpc", protocol: "OTLP/gRPC", endpoint: "set"},
		{name: "http/protobuf explicit", metricsExporter: "", eType: "http", protocol: "http/protobuf", endpoint: "set"},
		{name: "http/json not allowed", metricsExporter: "", eType: "none", protocol: "http/json", endpoint: "set", isNil: true},
		{name: "invalid protocol", metricsExporter: "", eType: "none", protocol: "invalid", endpoint: "set", isNil: true},
	}

	for _, tc := range tests {
		if tc.metricsExporter == "" {
			os.Unsetenv("OTEL_METRICS_EXPORTER")
		} else {
			os.Setenv("OTEL_METRICS_EXPORTER", tc.metricsExporter)
		}
		if tc.endpoint == "" {
			os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		} else {
			os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tc.endpoint)
		}
		if tc.protocol == "" {
			os.Unsetenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL")
		} else {
			os.Setenv("OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", tc.protocol)
		}
		exporter, etype, err := factory.GetExporter(context.Background(), false)
		if err != nil {
			t.Errorf("%s: GetExporter returned error: %v", tc.name, err)
		}
		if (exporter == nil) != tc.isNil {
			t.Errorf("%s, exporter unexpected: got nil=%v, want nil=%v", tc.name, exporter == nil, tc.isNil)
		}
		if etype != ExporterType(tc.eType) {
			t.Errorf("%s, exporter type unexpected: got %v, want %v", tc.name, etype, ExporterType(tc.eType))
		}

	}
}
