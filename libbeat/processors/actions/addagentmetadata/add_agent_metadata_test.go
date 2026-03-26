// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package addagentmetadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var testCfg = Config{
	InputID:  "unique-system-metrics-input",
	StreamID: "stream-abc-123",
	DataStream: &DataStreamConfig{
		Dataset:   "system.cpu",
		Namespace: "default",
		Type:      "metrics",
	},
	ElasticAgent: &ElasticAgentConfig{
		ID:       "db87c002-3ed1-4929-9edd-98cb6f76b2b1",
		Snapshot: true,
		Version:  "9.3.0",
	},
}

func TestAddAgentMetadata(t *testing.T) {
	p := New(testCfg)

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields:    mapstr.M{"existing_field": "keep_me"},
	}

	result, err := p.Run(event)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify @metadata
	assert.Equal(t, "unique-system-metrics-input", result.Meta["input_id"])
	assert.Equal(t, "stream-abc-123", result.Meta["stream_id"])

	// Verify data_stream
	ds, ok := result.Fields["data_stream"].(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "system.cpu", ds["dataset"])
	assert.Equal(t, "default", ds["namespace"])
	assert.Equal(t, "metrics", ds["type"])

	// Verify event.dataset
	ev, ok := result.Fields["event"].(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "system.cpu", ev["dataset"])

	// Verify elastic_agent
	ea, ok := result.Fields["elastic_agent"].(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "db87c002-3ed1-4929-9edd-98cb6f76b2b1", ea["id"])
	assert.Equal(t, true, ea["snapshot"])
	assert.Equal(t, "9.3.0", ea["version"])

	// Verify agent.id mirrors elastic_agent.id
	ag, ok := result.Fields["agent"].(mapstr.M)
	require.True(t, ok)
	assert.Equal(t, "db87c002-3ed1-4929-9edd-98cb6f76b2b1", ag["id"])

	// Verify pre-existing fields are preserved
	assert.Equal(t, "keep_me", result.Fields["existing_field"])
}

func TestAddAgentMetadata_NilEvent(t *testing.T) {
	p := New(testCfg)
	result, err := p.Run(nil)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestAddAgentMetadata_NilFieldsAndMeta(t *testing.T) {
	p := New(testCfg)
	event := &beat.Event{Timestamp: time.Now()}

	result, err := p.Run(event)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotNil(t, result.Meta)
	assert.NotNil(t, result.Fields)

	assert.Equal(t, "unique-system-metrics-input", result.Meta["input_id"])
	ds := result.Fields["data_stream"].(mapstr.M) //nolint:errcheck //it's a test
	assert.Equal(t, "system.cpu", ds["dataset"])
}

func TestAddAgentMetadata_OptionalFields(t *testing.T) {
	t.Run("no elastic_agent", func(t *testing.T) {
		cfg := Config{
			DataStream: &DataStreamConfig{
				Dataset:   "system.cpu",
				Namespace: "default",
				Type:      "metrics",
			},
		}
		p := New(cfg)
		event := &beat.Event{Timestamp: time.Now(), Fields: mapstr.M{}}

		result, err := p.Run(event)
		require.NoError(t, err)

		// input_id and stream_id should not be set when empty
		_, hasInputID := result.Meta["input_id"]
		assert.False(t, hasInputID)
		_, hasStreamID := result.Meta["stream_id"]
		assert.False(t, hasStreamID)
		_, hasElasticAgent := result.Fields["elastic_agent"]
		assert.False(t, hasElasticAgent)
		_, hasDS := result.Fields["data_stream"]
		assert.True(t, hasDS)
	})

	t.Run("no data_stream", func(t *testing.T) {
		cfg := Config{
			ElasticAgent: &ElasticAgentConfig{
				ID:      "agent-id",
				Version: "9.3.0",
			},
		}
		p := New(cfg)
		event := &beat.Event{Timestamp: time.Now(), Fields: mapstr.M{}}

		result, err := p.Run(event)
		require.NoError(t, err)

		// input_id and stream_id should not be set when empty
		_, hasInputID := result.Meta["input_id"]
		assert.False(t, hasInputID)
		_, hasStreamID := result.Meta["stream_id"]
		assert.False(t, hasStreamID)
		_, hasDS := result.Fields["data_stream"]
		assert.False(t, hasDS)
		_, hasElasticAgent := result.Fields["elastic_agent"]
		assert.True(t, hasElasticAgent)
	})
}

func TestAddAgentMetadata_PreservesExistingSubMaps(t *testing.T) {
	p := New(testCfg)
	event := &beat.Event{
		Timestamp: time.Now(),
		Meta:      mapstr.M{"existing_meta": "preserved"},
		Fields: mapstr.M{
			"agent": mapstr.M{"name": "my-host"},
			"event": mapstr.M{"module": "system"},
		},
	}

	result, err := p.Run(event)
	require.NoError(t, err)

	// Existing meta preserved
	assert.Equal(t, "preserved", result.Meta["existing_meta"])

	// Existing agent fields preserved alongside new ones
	ag := result.Fields["agent"].(mapstr.M) //nolint:errcheck //it's a test
	assert.Equal(t, "my-host", ag["name"])
	assert.Equal(t, testCfg.ElasticAgent.ID, ag["id"])

	// Existing event fields preserved alongside new ones
	ev := result.Fields["event"].(mapstr.M) //nolint:errcheck //it's a test
	assert.Equal(t, "system", ev["module"])
	assert.Equal(t, testCfg.DataStream.Dataset, ev["dataset"])
}

func TestAddAgentMetadata_FromConfig(t *testing.T) {
	c, err := conf.NewConfigFrom(map[string]interface{}{
		"input_id":  "test-input",
		"stream_id": "test-stream",
		"data_stream": map[string]interface{}{
			"dataset":   "system.cpu",
			"namespace": "default",
			"type":      "metrics",
		},
		"elastic_agent": map[string]interface{}{
			"id":       "agent-123",
			"snapshot": false,
			"version":  "9.3.0",
		},
	})
	require.NoError(t, err)

	p, err := CreateAddAgentMetadata(c, nil)
	require.NoError(t, err)

	event := &beat.Event{Timestamp: time.Now(), Fields: mapstr.M{}}
	result, err := p.Run(event)
	require.NoError(t, err)

	assert.Equal(t, "test-input", result.Meta["input_id"])
	ds := result.Fields["data_stream"].(mapstr.M) //nolint:errcheck //it's a test
	assert.Equal(t, "system.cpu", ds["dataset"])
}

func TestAddAgentMetadata_String(t *testing.T) {
	p := New(testCfg)
	s := p.String()
	assert.Contains(t, s, "add_agent_metadata")
	assert.Contains(t, s, testCfg.InputID)
	assert.Contains(t, s, testCfg.ElasticAgent.ID)
}

// equivalentAddFieldsProcessors returns the list of individual add_fields
// processors that produce the same result as a single add_agent_metadata
// processor. This is the baseline for benchmarking.
func equivalentAddFieldsProcessors(cfg Config) *processors.Processors {
	logger := logptest.NewTestingLogger(&testing.T{}, "")
	procs := processors.NewList(logger)

	if cfg.InputID != "" {
		procs.List = append(procs.List,
			addfields.MakeFieldsProcessor("@metadata", mapstr.M{"input_id": cfg.InputID}, true))
	}

	procs.List = append(procs.List,
		addfields.MakeFieldsProcessor("data_stream", mapstr.M{
			"dataset":   cfg.DataStream.Dataset,
			"namespace": cfg.DataStream.Namespace,
			"type":      cfg.DataStream.Type,
		}, true))

	procs.List = append(procs.List,
		addfields.MakeFieldsProcessor("event", mapstr.M{
			"dataset": cfg.DataStream.Dataset,
		}, true))

	if cfg.StreamID != "" {
		procs.List = append(procs.List,
			addfields.MakeFieldsProcessor("@metadata", mapstr.M{"stream_id": cfg.StreamID}, true))
	}

	procs.List = append(procs.List,
		addfields.MakeFieldsProcessor("elastic_agent", mapstr.M{
			"id":       cfg.ElasticAgent.ID,
			"snapshot": cfg.ElasticAgent.Snapshot,
			"version":  cfg.ElasticAgent.Version,
		}, true))

	procs.List = append(procs.List,
		addfields.MakeFieldsProcessor("agent", mapstr.M{
			"id": cfg.ElasticAgent.ID,
		}, true))

	return procs
}

// TestEquivalence verifies that the combined processor produces the exact same
// output as the chain of individual add_fields processors.
func TestEquivalence(t *testing.T) {
	combined := New(testCfg)
	separate := equivalentAddFieldsProcessors(testCfg)

	makeEvent := func() *beat.Event {
		return &beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"host": mapstr.M{"name": "test-host"},
			},
		}
	}

	eventCombined := makeEvent()
	resultCombined, err := combined.Run(eventCombined)
	require.NoError(t, err)

	eventSeparate := makeEvent()
	resultSeparate, err := separate.Run(eventSeparate)
	require.NoError(t, err)

	assert.Equal(t, resultSeparate.Meta, resultCombined.Meta, "Meta should be equivalent")
	assert.Equal(t, resultSeparate.Fields, resultCombined.Fields, "Fields should be equivalent")
}
