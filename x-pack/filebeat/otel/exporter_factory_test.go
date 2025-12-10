// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package otel

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGlobalExporterFactory(t *testing.T) {
	// set global to false
	factory1 := GetGlobalMetricsExporterFactory()
	assert.NotNil(t, factory1, "Expected non-nil factory, got nil")

	factory2 := GetGlobalMetricsExporterFactory()
	assert.Equal(t, factory1, factory2, "Expected same factory instance, got different instances")
}

func TestExporterFactory(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	options := GetDefaultMetricExporterOptions()
	factory := NewMetricsExporterFactory(options)
	assert.NotNil(t, factory, "Expected non-nil factory")
	exporter1, etype1, err := factory.GetExporter(context.Background(), false)
	assert.Nil(t, err)
	assert.NotNil(t, exporter1)
	assert.NotNil(t, etype1)
	exporter2, etype2, err := factory.GetExporter(context.Background(), true)
	assert.Nil(t, err)
	assert.NotNil(t, exporter2)
	assert.NotNil(t, etype2)
	exporter3, etype3, err := factory.GetExporter(context.Background(), true)
	assert.Nil(t, err)
	assert.NotNil(t, exporter3)
	assert.NotNil(t, etype3)
	assert.Equal(t, etype1, etype2)
	assert.NotEqual(t, exporter1, exporter2)
	assert.Equal(t, exporter2, exporter3)
}

func TestExporterFactoryNoMetricsEnvironment(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	options := GetDefaultMetricExporterOptions()
	factory := NewMetricsExporterFactory(options)
	assert.NotNil(t, factory, "Expected non-nil factory")
	exporter, etype, err := factory.GetExporter(context.Background(), false)
	assert.Nil(t, err)
	assert.NotNil(t, exporter)
	assert.NotNil(t, etype)
	assert.Equal(t, ExporterType("grpc"), etype)
}

func TestGetGlobalExporterNoneType(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	factory := GetGlobalMetricsExporterFactory()
	exporter, _, err := factory.GetExporter(context.Background(), true)
	assert.Nil(t, err, "GetExporter returned an error ")
	assert.NotNil(t, exporter, "Exporter should be nil")
	assert.Equal(t, exporter, factory.globalMetricsExporter)
}

func TestGetGlobalExporterGRPCType(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "set")
	factory := GetGlobalMetricsExporterFactory()
	exporter, _, err := factory.GetExporter(context.Background(), true)
	assert.Nil(t, err, "GetExporter returned an error ")
	assert.NotNil(t, exporter, "Exporter should not be nil")
	assert.Equal(t, exporter, factory.globalMetricsExporter)
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
		assert.Nil(t, err)
		assert.Equal(t, tc.isNil, exporter == nil, tc.name+", exporter unexpected")
		assert.Equal(t, ExporterType(tc.eType), etype, tc.name+", exporter type unexpected")

	}
}
