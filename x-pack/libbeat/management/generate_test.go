// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestMBGenerate(t *testing.T) {
	// following examples are built from this
	// 	raw := `
	//     id: system/metrics-system-default-system
	//     name: system-1
	//     revision: 1
	//     type: system/metrics
	//     use_output: default
	//     meta:
	//       package:
	//         name: system
	//         version: 1.17.0
	//     data_stream:
	//       namespace: default
	//     streams:
	//       - id: system/metrics-system.cpu-default-system
	//         data_stream:
	//           dataset: system.cpu
	//           type: metrics
	//         metricsets:
	//           - cpu
	//         cpu.metrics:
	//           - percentages
	//           - normalized_percentages
	//         period: 10s
	//       - id: system/metrics-system.diskio-default-system
	//         data_stream:
	//           dataset: system.diskio
	//           type: metrics
	//         metricsets:
	//           - diskio
	//         diskio.include_devices: null
	//         period: 10s
	//       - id: system/metrics-system.filesystem-default-system
	//         data_stream:
	//           dataset: system.filesystem
	//           type: metrics
	//         metricsets:
	//           - filesystem
	//         period: 1m
	//         processors:
	//           - drop_event.when.regexp:
	//               system.filesystem.mount_point: ^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)
	//       - id: system/metrics-system.fsstat-default-system
	//         data_stream:
	//           dataset: system.fsstat
	//           type: metrics
	//         metricsets:
	//           - fsstat
	//         period: 1m
	//         processors:
	//           - drop_event.when.regexp:
	//               system.fsstat.mount_point: ^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)
	//       - id: system/metrics-system.load-default-system
	//         data_stream:
	//           dataset: system.load
	//           type: metrics
	//         metricsets:
	//           - load
	//         period: 10s
	//       - id: system/metrics-system.memory-default-system
	//         data_stream:
	//           dataset: system.memory
	//           type: metrics
	//         metricsets:
	//           - memory
	//         period: 10s
	//       - id: system/metrics-system.network-default-system
	//         data_stream:
	//           dataset: system.network
	//           type: metrics
	//         metricsets:
	//           - network
	//         period: 10s
	//         network.interfaces: null
	//       - id: system/metrics-system.process-default-system
	//         data_stream:
	//           dataset: system.process
	//           type: metrics
	//         metricsets:
	//           - process
	//         period: 10s
	//         process.include_top_n.by_cpu: 5
	//         process.include_top_n.by_memory: 5
	//         process.cmdline.cache.enabled: true
	//         process.cgroups.enabled: false
	//         process.include_cpu_ticks: false
	//         processes:
	//           - .*
	//       - id: system/metrics-system.process.summary-default-system
	//         data_stream:
	//           dataset: system.process.summary
	//           type: metrics
	//         metricsets:
	//           - process_summary
	//         period: 10s
	//       - id: system/metrics-system.socket_summary-default-system
	//         data_stream:
	//           dataset: system.socket_summary
	//           type: metrics
	//         metricsets:
	//           - socket_summary
	//         period: 10s
	//       - id: system/metrics-system.uptime-default-system
	//         data_stream:
	//           dataset: system.uptime
	//           type: metrics
	//         metricsets:
	//           - uptime
	//         period: 10s
	// `

	sourceStream := requireNewStruct(t, map[string]interface{}{
		"metricsets": []interface{}{"filesystem"},
		"period":     "1m",
		"processors": []interface{}{
			map[string]interface{}{
				"drop_event.when.regexp": map[string]interface{}{
					"system.filesystem.mount_point": "^/(sys|cgroup|proc|dev|etc|host|lib|snap)($|/)",
				},
			},
		},
	})

	rawExpected := proto.UnitExpectedConfig{
		DataStream: &proto.DataStream{
			Namespace: "default",
		},
		Id:       "system/metrics-system-default-system",
		Type:     "system/metrics",
		Name:     "system-1",
		Revision: 1,
		Meta: &proto.Meta{
			Package: &proto.Package{
				Name:    "system",
				Version: "1.17.0",
			},
		},
		Streams: []*proto.Stream{
			{
				Id: "system/metrics-system.filesystem-default-system",
				DataStream: &proto.DataStream{
					Dataset: "system.filesystem",
					Type:    "metrics",
				},
				Source: sourceStream,
			},
		},
	}

	reloadCfg, err := generateBeatConfig(&rawExpected, &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})
	require.NoError(t, err, "error in generateBeatConfig")
	//unpack, again, so we can read it
	for _, stream := range reloadCfg {
		cfgMap := mapstr.M{}
		err = stream.Config.Unpack(&cfgMap)
		require.NoError(t, err, "error in unpack for config %#v", stream.Config)
		t.Logf("Config: %s", cfgMap.StringToPrint())
	}

}

func TestOutputGen(t *testing.T) {
	testExpected := proto.UnitExpectedConfig{
		Type: "elasticsearch",
		Source: requireNewStruct(t, map[string]interface{}{
			"hosts":    []interface{}{"localhost:9200"},
			"username": "elastic",
			"password": "changeme",
		}),
	}

	cfg, err := groupByOutputs(&testExpected)
	require.NoError(t, err)
	testStruct := mapstr.M{}
	err = cfg.Config.Unpack(&testStruct)
	require.Error(t, err)
	innerCfg, exists := testStruct["elasticsearch"]
	assert.True(t, exists, "elasticsearch key does not exist")
	_, pwExists := innerCfg.(map[string]interface{})["password"]
	assert.True(t, pwExists, "password config not found")

}

func requireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err)
	}
	return str
}
