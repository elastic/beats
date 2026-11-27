// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
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
				Source: requireNewStruct(t, map[string]any{
					"metricsets": []any{"filesystem"},
					"period":     "1m",
				}),
			},
		},
	}

	cfgMap := buildConfigMap(t, &rawExpected, &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true, Tags: []string{"production", "team-payments"}})

	// Actual checks
	processorFields := map[string]any{
		"add_fields.fields.stream_id": "system/metrics-system.filesystem-default-system",
		"add_fields.fields.dataset":   "generic",
		"add_fields.fields.namespace": "default",
		"add_fields.fields.type":      "log",
		"add_fields.fields.input_id":  "system/metrics-system-default-system",
		"add_fields.fields.id":        "beat-ID",
		"add_tags.tags":               []any{"production", "team-payments"},
	}
	findFieldsInProcessors(t, processorFields, cfgMap)
}

func TestBareConfigForReceiver(t *testing.T) {
	// config with datastreams, metadata, etc, removed
	rawExpected := proto.UnitExpectedConfig{
		Id:   "system/metrics-system-default-system",
		Type: "system/metrics",
		Name: "system-1",
		Streams: []*proto.Stream{
			{
				Id: "system/metrics-system.filesystem-default-system",
				Source: requireNewStruct(t, map[string]any{
					"metricsets": []any{"filesystem"},
					"period":     "1m",
				}),
			},
		},
	}

	streams, err := CreateInputsFromStreamsForReceiver(&rawExpected, "log", &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true, Tags: []string{"production", "team-payments"}})
	require.NoError(t, err, "error in CreateInputsFromStreamsForReceiver")
	reloadCfg, err := CreateReloadConfigFromInputs(streams)
	require.NoError(t, err, "error in CreateReloadConfigFromInputs")
	cfgMap := mapstr.M{}
	err = reloadCfg[0].Config.Unpack(&cfgMap)
	require.NoError(t, err, "error in unpack for config %#v", reloadCfg[0].Config)

	// Actual checks
	processorFields := map[string]any{
		"add_agent_metadata.stream_id":             "system/metrics-system.filesystem-default-system",
		"add_agent_metadata.data_stream.dataset":   "generic",
		"add_agent_metadata.data_stream.namespace": "default",
		"add_agent_metadata.data_stream.type":      "log",
		"add_agent_metadata.input_id":              "system/metrics-system-default-system",
		"add_agent_metadata.elastic_agent.id":      "beat-ID",
		"add_agent_metadata.tags":                  []any{"production", "team-payments"},
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
				Source: requireNewStruct(t, map[string]any{
					"metricsets": []any{"filesystem"},
					"period":     "1m",
				}),
			},
		},
		Source: requireNewStruct(t, map[string]any{
			"processors": []any{
				map[string]any{
					"add_fields": map[string]any{
						"fields": map[string]any{
							"cluster": map[string]any{
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
	processorFields := map[string]any{
		"add_fields.fields.stream_id":    "system/metrics-system.filesystem-default-system", // make sure we're not overwiting anything
		"add_fields.fields.dataset":      "generic",
		"add_fields.fields.cluster.name": "kind", // actual test for the global processors
		"add_fields.target":              "orchestrator",
	}
	findFieldsInProcessors(t, processorFields, cfgMap)
}

func TestGlobalProcessorsInjectedOnceForSimpleConfig(t *testing.T) {
	addTags := map[string]any{"add_tags": map[string]any{"tags": []any{"global"}}}
	raw := &proto.UnitExpectedConfig{
		Id:   "filestream-1",
		Type: "filestream",
		Source: requireNewStruct(t, map[string]any{
			"type":       "filestream",
			"paths":      []any{"/tmp/*.log"},
			"processors": []any{addTags},
		}),
	}
	agentInfo := &client.AgentInfo{ID: "beat-ID", Version: "8.0.0"}

	cases := map[string]struct {
		create func() ([]map[string]any, error)
		want   []any
	}{
		"CreateInputsFromStreams": {
			create: func() ([]map[string]any, error) {
				return CreateInputsFromStreams(raw, "logs", agentInfo)
			},
			want: []any{
				mapstr.M{"add_fields": mapstr.M{"fields": mapstr.M{"input_id": "filestream-1"}, "target": "@metadata"}},
				mapstr.M{"add_fields": mapstr.M{"fields": mapstr.M{"dataset": "generic", "namespace": "default", "type": "logs"}, "target": "data_stream"}},
				mapstr.M{"add_fields": mapstr.M{"fields": mapstr.M{"dataset": "generic"}, "target": "event"}},
				mapstr.M{"add_fields": mapstr.M{"fields": mapstr.M{"id": "beat-ID", "snapshot": false, "version": "8.0.0"}, "target": "elastic_agent"}},
				mapstr.M{"add_fields": mapstr.M{"fields": mapstr.M{"id": "beat-ID"}, "target": "agent"}},
				addTags,
			},
		},
		"CreateInputsFromStreamsForReceiver": {
			create: func() ([]map[string]any, error) {
				return CreateInputsFromStreamsForReceiver(raw, "logs", agentInfo)
			},
			want: []any{
				mapstr.M{"add_agent_metadata": mapstr.M{
					"data_stream":   mapstr.M{"dataset": "generic", "namespace": "default", "type": "logs"},
					"elastic_agent": mapstr.M{"id": "beat-ID", "snapshot": false, "version": "8.0.0"},
					"input_id":      "filestream-1",
				}},
				addTags,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			inputs, err := tc.create()
			require.NoError(t, err)
			require.Len(t, inputs, 1)
			assert.Equal(t, tc.want, inputs[0]["processors"])
		})
	}
}

// TestSyntheticsStreamsForReceiver checks that synthetics inputs leave the
// data stream routing to Heartbeat: no fixed `index`, a `data_stream` setting
// and no processor that rewrites data_stream.* after Heartbeat routed the event.
func TestSyntheticsStreamsForReceiver(t *testing.T) {
	agentInfo := &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true, Tags: []string{"production"}}

	newUnit := func(t *testing.T, inputType, dataset string, source map[string]any) *proto.UnitExpectedConfig {
		return &proto.UnitExpectedConfig{
			Id:         inputType + "-my-monitor",
			Type:       inputType,
			Name:       "synthetics-1",
			DataStream: &proto.DataStream{Namespace: "custom-ns"},
			Streams: []*proto.Stream{{
				Id:         "stream-id",
				DataStream: &proto.DataStream{Type: "synthetics", Dataset: dataset},
				Source:     requireNewStruct(t, source),
			}},
		}
	}

	for _, tc := range []struct{ inputType, dataset string }{
		{"synthetics/http", "http"},
		{"synthetics/browser", "browser"},
	} {
		t.Run(tc.inputType, func(t *testing.T) {
			unit := newUnit(t, tc.inputType, tc.dataset, map[string]any{"id": "my-monitor", "schedule": "@every 5s"})
			inputs, err := CreateInputsFromStreamsForReceiver(unit, "logs", agentInfo)
			require.NoError(t, err, "creating inputs")
			require.Len(t, inputs, 1, "one input per stream")
			in := inputs[0]

			assert.NotContains(t, in, "index", "a fixed index would override Heartbeat's per-event routing")
			assert.Equal(t, map[string]any{"type": "synthetics", "dataset": tc.dataset, "namespace": "custom-ns"},
				in["data_stream"], "Heartbeat must receive the data_stream to build its own routing")

			procs, ok := in["processors"].([]any)
			require.True(t, ok, "processors must be a list, got %T", in["processors"])
			require.Len(t, procs, 1, "only add_agent_metadata expected")
			assert.Equal(t, mapstr.M{"add_agent_metadata": mapstr.M{
				"elastic_agent": mapstr.M{"id": "beat-ID", "snapshot": true, "version": "8.0.0"},
				"input_id":      tc.inputType + "-my-monitor",
				"stream_id":     "stream-id",
				"tags":          []string{"production"},
			}}, procs[0], "add_agent_metadata must not carry data_stream")
		})
	}

	t.Run("explicit index is preserved", func(t *testing.T) {
		unit := newUnit(t, "synthetics/http", "http", map[string]any{"id": "m", "index": "custom-index"})
		inputs, err := CreateInputsFromStreamsForReceiver(unit, "logs", agentInfo)
		require.NoError(t, err, "creating inputs")
		require.Len(t, inputs, 1, "one input per stream")
		assert.Equal(t, "custom-index", inputs[0]["index"], "explicit index wins")
		assert.NotContains(t, inputs[0], "data_stream", "data_stream is dropped when index is explicit")
	})

	t.Run("nil agent info does not overwrite the dataset", func(t *testing.T) {
		unit := newUnit(t, "synthetics/browser", "browser", map[string]any{"id": "m"})
		inputs, err := CreateInputsFromStreamsForReceiver(unit, "logs", nil)
		require.NoError(t, err, "creating inputs")
		require.Len(t, inputs, 1, "one input per stream")
		assert.NotContains(t, inputs[0], "index", "no fixed index")
		assert.Equal(t, map[string]any{"type": "synthetics", "dataset": "browser", "namespace": "custom-ns"},
			inputs[0]["data_stream"], "data_stream setting kept")
		procs, ok := inputs[0]["processors"].([]any)
		require.True(t, ok, "processors must be a []any")
		for _, p := range procs {
			pm, ok := p.(mapstr.M)
			require.True(t, ok, "processor must be a mapstr.M")
			af, ok := pm["add_fields"].(mapstr.M)
			if !ok {
				continue
			}
			assert.NotEqual(t, "data_stream", af["target"], "add_fields must not rewrite data_stream")
			assert.NotEqual(t, "event", af["target"], "add_fields must not rewrite event.dataset")
		}
	})

	t.Run("non-synthetics inputs keep index and data_stream processor", func(t *testing.T) {
		unit := &proto.UnitExpectedConfig{
			Id: "filestream-1", Type: "filestream",
			DataStream: &proto.DataStream{Namespace: "default"},
			Streams: []*proto.Stream{{
				Id:         "s",
				DataStream: &proto.DataStream{Type: "logs", Dataset: "foo"},
				Source:     requireNewStruct(t, map[string]any{"paths": []any{"/x"}}),
			}},
		}
		inputs, err := CreateInputsFromStreamsForReceiver(unit, "logs", agentInfo)
		require.NoError(t, err, "creating inputs")
		assert.Equal(t, "logs-foo-default", inputs[0]["index"], "index still injected")
		procs, ok := inputs[0]["processors"].([]any)
		require.True(t, ok, "processors must be a []any")
		require.NotEmpty(t, procs, "processors must not be empty")
		first, ok := procs[0].(mapstr.M)
		require.True(t, ok, "first processor must be a mapstr.M")
		proc, ok := first["add_agent_metadata"].(mapstr.M)
		require.True(t, ok, "first processor must be add_agent_metadata")
		assert.Equal(t, mapstr.M{"dataset": "foo", "namespace": "default", "type": "logs"}, proc["data_stream"],
			"data_stream still set on add_agent_metadata")
	})
}

func TestIndexPrecedence(t *testing.T) {

	withSource := func(source map[string]any) *proto.UnitExpectedConfig {
		return &proto.UnitExpectedConfig{
			DataStream: &proto.DataStream{
				Namespace: "default",
			},
			Id:       "filestream-filestream",
			Type:     "filestream",
			Name:     "filestream-1",
			Revision: 1,
			Meta: &proto.Meta{
				Package: &proto.Package{
					Name:    "filestream",
					Version: "1.3.0",
				},
			},
			Streams: []*proto.Stream{
				{
					Id: "filestream-filestream.generic",
					DataStream: &proto.DataStream{
						Dataset: "filestream.generic",
						Type:    "logs",
					},
					Source: requireNewStruct(t, source),
				},
			},
		}
	}

	t.Run("index takes precedence", func(t *testing.T) {
		source := map[string]any{
			"data_stream": map[string]any{
				"dataset": "filestream.generic",
			},
			"index": "logs",
			"paths": []any{"/logs/*"},
		}

		cfgMap := buildConfigMap(t, withSource(source), &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})
		require.Equal(t, "logs", cfgMap["index"])
		require.Nil(t, cfgMap["data_stream"])
	})

	t.Run("data_stream forms an index", func(t *testing.T) {
		source := map[string]any{
			"data_stream": map[string]any{
				"dataset": "filestream.generic",
			},
			"paths": []any{"/logs/*"},
		}

		cfgMap := buildConfigMap(t, withSource(source), &client.AgentInfo{ID: "beat-ID", Version: "8.0.0", Snapshot: true})
		require.Equal(t, "logs-filestream.generic-default", cfgMap["index"])
		require.NotNil(t, cfgMap["data_stream"])
	})
}

func TestMBGenerate(t *testing.T) {
	sourceStream := requireNewStruct(t, map[string]any{
		"metricsets": []any{"filesystem"},
		"period":     "1m",
		"processors": []any{
			map[string]any{
				"drop_event.when.regexp": map[string]any{
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
	configFields := map[string]any{
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
		Source: requireNewStruct(t, map[string]any{
			"hosts":    []any{"localhost:9200"},
			"username": "elastic",
			"password": "changeme",
		}),
	}

	cfg, err := GroupByOutputs(&testExpected)
	require.NoError(t, err)
	testStruct := mapstr.M{}
	err = cfg.Config.Unpack(&testStruct)
	require.NoError(t, err)
	innerCfg, exists := testStruct["elasticsearch"]
	assert.True(t, exists, "elasticsearch key does not exist")
	innerCfgMap, ok := innerCfg.(map[string]any)
	require.True(t, ok, "must be a map[string]any")
	_, pwExists := innerCfgMap["password"]
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
	inStream := map[string]any{}
	outStream := injectIndexStream(dataStreamType, unit, stream, inStream)
	require.Equal(t, "synthetics-icmp-example", outStream["index"])

	//test Defaults
	emptyStream := &proto.Stream{DataStream: &proto.DataStream{}}
	emptyUnit := &proto.UnitExpectedConfig{DataStream: &proto.DataStream{}}
	outDefaultIndex := injectIndexStream(dataStreamType, emptyUnit, emptyStream, inStream)
	require.Equal(t, "unused-generic-default", outDefaultIndex["index"])

}

func requireNewStruct(t testing.TB, v map[string]any) *structpb.Struct {
	str, err := structpb.NewStruct(v)
	if err != nil {
		require.NoError(t, err)
	}
	return str
}

func findFieldsInProcessors(t *testing.T, configFields map[string]any, cfgMap mapstr.M) {
	for key, val := range configFields {
		gotKey := false
		gotVal := false
		var got any
		procList, ok := cfgMap["processors"].([]any)
		require.True(t, ok, "must be a list of objects")
		for _, proc := range procList {
			p, ok := proc.(map[string]any)
			require.True(t, ok, "must be a map[string]any")
			processor := mapstr.M(p)
			found, err := processor.GetValue(key)
			if err == nil {
				gotKey = true
				if val == nil {
					gotVal = true
				} else {
					if assert.ObjectsAreEqual(val, found) {
						gotVal = true
					} else {
						got = found
					}
				}
			}
		}
		assert.True(t, gotKey, "did not find key for %s", key)
		assert.True(t, gotVal, "got incorrect value for %s, expected %v, got %v", key, val, got)
	}
}

func TestMetadataFromDatastreamValues(t *testing.T) {
	cases := map[string]struct {
		defaultDataStreamType string
		expected              *proto.UnitExpectedConfig
		streamExpected        *proto.Stream

		expectedStreamType string
		expectedDataset    string
		expectedNamespace  string
	}{
		// defaults test case
		"default_type": {
			defaultDataStreamType: "logs",

			expectedStreamType: "logs",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  DefaultNamespaceName,
		},

		// type test cases
		"type_from_expected": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Type: "expected-metrics",
				},
			},

			expectedStreamType: "expected-metrics",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  DefaultNamespaceName,
		},
		"type_from_stream": {
			defaultDataStreamType: "logs",
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Type: "stream-metrics",
				},
			},

			expectedStreamType: "stream-metrics",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  DefaultNamespaceName,
		},
		"type_from_both": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Type: "expected-metrics",
				},
			},
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Type: "stream-metrics",
				},
			},

			expectedStreamType: "expected-metrics",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  DefaultNamespaceName,
		},

		// dataset test cases
		"dataset_from_expected": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Dataset: "expected-dataset",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    "expected-dataset",
			expectedNamespace:  DefaultNamespaceName,
		},
		"dataset_from_stream": {
			defaultDataStreamType: "logs",
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Dataset: "stream-dataset",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    "stream-dataset",
			expectedNamespace:  DefaultNamespaceName,
		},
		"dataset_from_both": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Dataset: "expected-dataset",
				},
			},
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Dataset: "stream-dataset",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    "expected-dataset",
			expectedNamespace:  DefaultNamespaceName,
		},

		// namespace test cases
		"namespace_from_expected": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Namespace: "expected-namespace",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  "expected-namespace",
		},
		"namespace_from_stream": {
			defaultDataStreamType: "logs",
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Namespace: "stream-namespace",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  "stream-namespace",
		},
		"namespace_from_both": {
			defaultDataStreamType: "logs",
			expected: &proto.UnitExpectedConfig{
				DataStream: &proto.DataStream{
					Namespace: "expected-namespace",
				},
			},
			streamExpected: &proto.Stream{
				DataStream: &proto.DataStream{
					Namespace: "stream-namespace",
				},
			},

			expectedStreamType: "logs",
			expectedDataset:    DefaultDatasetName,
			expectedNamespace:  "expected-namespace",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			streamType, dataset, namespace := metadataFromDatastreamValues(tc.defaultDataStreamType, tc.expected, tc.streamExpected)
			require.Equal(t, tc.expectedStreamType, streamType)
			require.Equal(t, tc.expectedDataset, dataset)
			require.Equal(t, tc.expectedNamespace, namespace)
		})
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

// BenchmarkInjectGlobalProcessors injects the global processors into every
// stream of a unit, the cost per stream must not grow with the stream count.
func BenchmarkInjectGlobalProcessors(b *testing.B) {
	for _, streams := range []int{500, 2000, 8000} {
		b.Run(strconv.Itoa(streams), func(b *testing.B) {
			protoStreams := make([]*proto.Stream, streams)
			for i := range protoStreams {
				protoStreams[i] = &proto.Stream{
					Id:     "stream-" + strconv.Itoa(i),
					Source: requireNewStruct(b, map[string]any{"period": "1m"}),
				}
			}
			// The source of the unit holds every stream.
			streamList := make([]any, streams)
			for i := range streamList {
				streamList[i] = map[string]any{"id": "stream-" + strconv.Itoa(i), "period": "1m"}
			}
			expected := &proto.UnitExpectedConfig{
				Streams: protoStreams,
				Source: requireNewStruct(b, map[string]any{
					"processors": []any{map[string]any{"add_fields": map[string]any{"target": "orchestrator"}}},
					"streams":    streamList,
				}),
			}

			for b.Loop() {
				for range streams {
					injectGlobalProcesssors(expected, map[string]any{"period": "1m"})
				}
			}
		})
	}
}
