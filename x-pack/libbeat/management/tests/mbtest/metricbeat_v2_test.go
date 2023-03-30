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
				"period":     "1s",
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
				"period":     "1s",
			}),
		},
	}

	expectedMBStreams.Streams = mbStreams
	outPath, server := tests.SetupTestEnv(t, expectedMBStreams, time.Second*40)

	defer server.Srv.Stop()
	defer func() {
		err := os.RemoveAll(outPath)
		require.NoError(t, err)
	}()

	go func() {
		t.Logf("Running beats...")
		err := cmd.RootCmd.Execute()
		require.NoError(t, err)
	}()

	found := false
	for {
		if found {
			server.Client.Stop()
			server.Srv.Stop()
			break
		}
		time.Sleep(time.Second)
		// check to see we have at least one event from both metricsets
		events := tests.ReadEvents(t, outPath)
		memoryEvt := false
		cpuEvt := false
		for _, evt := range events {
			if found, _ := evt.GetValue("data_stream.dataset"); found != nil {
				dataset, ok := found.(string)
				if ok {
					if dataset == "system.cpu" {
						cpuEvt = true
					}
					if dataset == "system.memory" {
						memoryEvt = true
					}
				}
			}
			if cpuEvt && memoryEvt {
				t.Logf("found memory and CPU events")
				found = true
			}
		}
	}

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
	tests.ValuesExist(t, expectedCPUMetaValues, events, tests.ONCE, "expectedCPUMetaValues")

	expectedMemoryMetaValues := map[string]interface{}{
		"@metadata.stream_id": "system/metrics-system.memory-default-system",
		"data_stream.dataset": "system.memory",
	}
	tests.ValuesExist(t, expectedMemoryMetaValues, events, tests.ONCE, "expectedMemoryMetaValues")

	// Look for proper CPU/memory config
	expectedCPU := map[string]interface{}{
		"system.cpu.cores":          nil,
		"system.cpu.total":          nil,
		"system.memory.actual.free": nil,
	}
	tests.ValuesExist(t, expectedCPU, events, tests.ONCE, "expectedCPU")

	// If there's a config issue, metricbeat will fallback to default metricsets. Make sure they don't exist.
	disabledMetricsets := []string{
		"system.process",
		"system.load",
		"system.process_summary",
	}
	tests.ValuesDoNotExist(t, disabledMetricsets, events, "disabledMetricsets")
}
