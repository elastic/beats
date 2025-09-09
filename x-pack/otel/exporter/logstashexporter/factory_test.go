// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package logstashexporter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter/exportertest"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
)

func TestCreateExporter(t *testing.T) {
	factory := NewFactory()
	cfg := logstashOutputConfig{
		Config: logstash.DefaultConfig(),
		HostWorkerCfg: outputs.HostWorkerCfg{
			Hosts: []string{"localhost:5044"},
		},
	}
	params := exportertest.NewNopSettings(component.MustNewType(Name))
	exporter, err := factory.CreateLogs(context.Background(), params, cfg)
	require.NoError(t, err)
	require.NotNil(t, exporter)

	require.NoError(t, exporter.Shutdown(context.Background()))
}
