// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mbtest

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	// initialize the plugin system before libbeat does, so we can overwrite it properly
	_ "github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management/tests"
	"github.com/elastic/beats/v7/x-pack/metricbeat/cmd"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

var expectedMBStreams = &proto.UnitExpectedConfig{
	DataStream: &proto.DataStream{
		Namespace: "default",
	},
	Type:     "system/metrics",
	Id:       "system/metrics-system-default-system",
	Name:     "system-1",
	Revision: 1,
	Meta: &proto.Meta{
		Package: &proto.Package{
			Name:    "system",
			Version: "1.17.0",
		},
	},
}

func TestSingleMetricbeatMetricsetWithProcessors(t *testing.T) {
	tests.InitBeatsForTest(t, cmd.RootCmd)
	var mbStreams = []*proto.Stream{
		{
			Id: "system/metrics-system.cpu-default-system",
			DataStream: &proto.DataStream{
				Dataset: "system.cpu",
				Type:    "metrics",
			},
			Source: tests.RequireNewStruct(map[string]interface{}{
				"metricsets": []interface{}{"cpu"},
				"period":     "2s",
				"processors": []interface{}{
					map[string]interface{}{
						"add_fields": map[string]interface{}{
							"fields": map[string]interface{}{"testfield": true},
							"target": "@metadata",
						},
					},
				},
			}),
		},
		{
			Id: "system/metrics-system.memory-default-system",
			DataStream: &proto.DataStream{
				Dataset: "system.memory",
				Type:    "metrics",
			},
			Source: tests.RequireNewStruct(map[string]interface{}{
				"metricsets": []interface{}{"memory"},
				"period":     "2s",
			}),
		},
	}

	expectedMBStreams.Streams = mbStreams
	outPath, server := tests.SetupTestEnv(t, expectedMBStreams, time.Second*6)

	defer server.Srv.Stop()
	defer func() {
		err := os.RemoveAll(outPath)
		require.NoError(t, err)
	}()

	// After runfor seconds, this should shut down, allowing us to check the output
	t.Logf("Running beats...")
	err := cmd.RootCmd.Execute()
	require.NoError(t, err)

	t.Logf("Reading events...")
	events := tests.ReadEvents(t, outPath)
	t.Logf("Got %d events", len(events))

	// Look for processors
	expectedCPUMetaValues := map[string]interface{}{
		// Processors created by
		"@metadata.input_id":    "system/metrics-system-default-system",
		"@metadata.stream_id":   "system/metrics-system.cpu-default-system",
		"agent.id":              "test-agent",
		"data_stream.dataset":   "system.cpu",
		"data_stream.namespace": "default",
		"data_stream.type":      "metrics",
		// make sure the V2 shim isn't overwriting any custom processors
		"@metadata.testfield": true,
	}
	tests.ValuesExist(t, expectedCPUMetaValues, events, tests.ONCE)

	expectedMemoryMetaValues := map[string]interface{}{
		"@metadata.stream_id": "system/metrics-system.memory-default-system",
		"data_stream.dataset": "system.memory",
	}
	tests.ValuesExist(t, expectedMemoryMetaValues, events, tests.ONCE)

	// Look for proper CPU/memory config
	expectedCPU := map[string]interface{}{
		"system.cpu.cores":          nil,
		"system.cpu.total":          nil,
		"system.memory.actual.free": nil,
	}
	tests.ValuesExist(t, expectedCPU, events, tests.ONCE)

	// If there's a config issue, metricbeat will fallback to default metricsets. Make sure they don't exist.
	disabledMetricsets := []string{
		"system.process",
		"system.load",
		"system.process_summary",
	}
	tests.ValuesDoNotExist(t, disabledMetricsets, events)
}
