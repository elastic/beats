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

func TestBareConfig(t *testing.T) {
	// config with datastreams, metadata, etc, removed
	rawExpected := proto.UnitExpectedConfig{
		Id:   "system/metrics-system-default-system",
		Type: "system/metrics",
		Name: "system-1",
		Streams: []*proto.Stream{
			{
				Id: "system/metrics-system.filesystem-default-system",
				Source: requireNewStruct(t, map[string]interface{}{
					"metricsets": []interface{}{"filesystem"},
					"period":     "1m",
				}),
			},
		},
	}

	cfgMap := buildConfigMap(t, &rawExpected, &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})

	// Actual checks
	processorFields := map[string]interface{}{
		"add_fields.fields.stream_id": "system/metrics-system.filesystem-default-system",
		"add_fields.fields.dataset":   "generic",
		"add_fields.fields.namespace": "default",
		"add_fields.fields.type":      "log",
		"add_fields.fields.input_id":  "system/metrics-system-default-system",
		"add_fields.fields.id":        "beat-ID",
	}
	findFieldsInProcessors(t, processorFields, cfgMap)
}

func TestGlobalProcessInject(t *testing.T) {
	// config with datastreams, metadata, etc, removed
	rawExpected := proto.UnitExpectedConfig{
		Id:   "system/metrics-system-default-system",
		Type: "system/metrics",
		Name: "system-1",
		Streams: []*proto.Stream{
			{
				Id: "system/metrics-system.filesystem-default-system",
				Source: requireNewStruct(t, map[string]interface{}{
					"metricsets": []interface{}{"filesystem"},
					"period":     "1m",
				}),
			},
		},
		Source: requireNewStruct(t, map[string]interface{}{
			"processors": []interface{}{
				map[string]interface{}{
					"add_fields": map[string]interface{}{
						"fields": map[string]interface{}{
							"cluster": map[string]interface{}{
								"name": "kind",
								"url":  "kind-control-plane:6443",
							},
						},
						"target": "orchestrator",
					},
				},
			},
		}),
	}

	cfgMap := buildConfigMap(t, &rawExpected, &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})
	processorFields := map[string]interface{}{
		"add_fields.fields.stream_id":    "system/metrics-system.filesystem-default-system", // make sure we're not overwiting anything
		"add_fields.fields.dataset":      "generic",
		"add_fields.fields.cluster.name": "kind", // actual test for the global processors
		"add_fields.target":              "orchestrator",
	}
	findFieldsInProcessors(t, processorFields, cfgMap)
}

func TestMBGenerate(t *testing.T) {
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

	cfgMap := buildConfigMap(t, &rawExpected, &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})
	configFields := map[string]interface{}{
		"drop_event":                  nil,
		"add_fields.fields.stream_id": "system/metrics-system.filesystem-default-system",
		"add_fields.fields.dataset":   "system.filesystem",
		"add_fields.fields.input_id":  "system/metrics-system-default-system",
		"add_fields.fields.id":        "beat-ID",
	}
	findFieldsInProcessors(t, configFields, cfgMap)

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
	require.NoError(t, err)
	innerCfg, exists := testStruct["elasticsearch"]
	assert.True(t, exists, "elasticsearch key does not exist")
	_, pwExists := innerCfg.(map[string]interface{})["password"]
	assert.True(t, pwExists, "password config not found")

}

func TestOutputIndex(t *testing.T) {
	dataStreamType := "unused"
	stream := &proto.Stream{
		DataStream: &proto.DataStream{
			Type:      "synthetics",
			Dataset:   "icmp",
			Namespace: "example", // This should *not* get applied.
		},
	}
	unit := &proto.UnitExpectedConfig{
		DataStream: &proto.DataStream{
			Namespace: "default",
		},
	}
	inStream := map[string]interface{}{}
	outStream := injectIndexStream(dataStreamType, unit, stream, inStream)
	require.Equal(t, "synthetics-icmp-default", outStream["index"])

	//test Defaults
	emptyStream := &proto.Stream{DataStream: &proto.DataStream{}}
	emptyUnit := &proto.UnitExpectedConfig{DataStream: &proto.DataStream{}}
	outDefaultIndex := injectIndexStream(dataStreamType, emptyUnit, emptyStream, inStream)
	require.Equal(t, "unused-generic-default", outDefaultIndex["index"])

}

func requireNewStruct(t *testing.T, v map[string]interface{}) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err)
	}
	return str
}

func findFieldsInProcessors(t *testing.T, configFields map[string]interface{}, cfgMap mapstr.M) {
	for key, val := range configFields {
		gotKey := false
		gotVal := false
		errStr := ""
		for _, proc := range cfgMap["processors"].([]interface{}) {
			processor := mapstr.M(proc.(map[string]interface{}))
			found, ok := processor.GetValue(key)
			if ok == nil {
				gotKey = true
				if val == nil {
					gotVal = true
				} else {
					if val == found {
						gotVal = true
					} else {
						errStr = found.(string)
					}
				}
			}
		}
		assert.True(t, gotKey, "did not find key for %s", key)
		assert.True(t, gotVal, "got incorrect key for %s, expected %s, got %s", key, val, errStr)
	}
}

func buildConfigMap(t *testing.T, unitRaw *proto.UnitExpectedConfig, agentInfo *client.AgentInfo) mapstr.M {
	reloadCfg, err := generateBeatConfig(unitRaw, agentInfo)
	require.NoError(t, err, "error in generateBeatConfig")
	cfgMap := mapstr.M{}
	err = reloadCfg[0].Config.Unpack(&cfgMap)
	require.NoError(t, err, "error in unpack for config %#v", reloadCfg[0].Config)
	return cfgMap
}
