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

package processing

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProcessorsConfigs(t *testing.T) {
	defaultInfo := beat.Info{
		Beat:        "test",
		EphemeralID: uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000")),
		Hostname:    "test.host.name",
		ID:          uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440001")),
		Name:        "test.host.name",
		Version:     "0.1",
	}

	ecsFields := mapstr.M{"version": ecs.Version}

	cases := map[string]struct {
		factory  SupportFactory
		global   string
		local    beat.ProcessingConfig
		drop     bool
		event    string
		want     mapstr.M
		wantMeta mapstr.M
		infoMod  func(beat.Info) beat.Info
	}{
		"user global fields and tags": {
			global: "{fields: {global: 1}, fields_under_root: true, tags: [tag]}",
			event:  `{"value": "abc"}`,
			want: mapstr.M{
				"value":  "abc",
				"global": uint64(1),
				"tags":   []string{"tag"},
			},
		},
		"beat local fields": {
			global: "",
			local: beat.ProcessingConfig{
				Fields: mapstr.M{"local": 1},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value": "abc",
				"local": 1,
			},
		},
		"beat local and user global fields": {
			global: "{fields: {global: 1}, fields_under_root: true, tags: [tag]}",
			local: beat.ProcessingConfig{
				Fields: mapstr.M{"local": 1},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value":  "abc",
				"global": uint64(1),
				"local":  1,
				"tags":   []string{"tag"},
			},
		},
		"user global fields overwrite beat local fields": {
			global: "{fields: {global: a, shared: global}, fields_under_root: true}",
			local: beat.ProcessingConfig{
				Fields: mapstr.M{"local": "b", "shared": "local"},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value":  "abc",
				"local":  "b",
				"global": "a",
				"shared": "global",
			},
		},
		"user local fields and tags": {
			local: beat.ProcessingConfig{
				EventMetadata: mapstr.EventMetadata{
					Fields: mapstr.M{"local": "a"},
					Tags:   []string{"tag"},
				},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value": "abc",
				"fields": mapstr.M{
					"local": "a",
				},
				"tags": []string{"tag"},
			},
		},
		"user local fields (under root) and tags": {
			local: beat.ProcessingConfig{
				EventMetadata: mapstr.EventMetadata{
					Fields:          mapstr.M{"local": "a"},
					FieldsUnderRoot: true,
					Tags:            []string{"tag"},
				},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value": "abc",
				"local": "a",
				"tags":  []string{"tag"},
			},
		},
		"user local fields overwrite user global fields": {
			global: `{fields: {global: a, shared: global}, fields_under_root: true, tags: [global]}`,
			local: beat.ProcessingConfig{
				EventMetadata: mapstr.EventMetadata{
					Fields: mapstr.M{
						"local":  "a",
						"shared": "local",
					},
					FieldsUnderRoot: true,
					Tags:            []string{"local"},
				},
			},
			event: `{"value": "abc"}`,
			want: mapstr.M{
				"value":  "abc",
				"global": "a",
				"local":  "a",
				"shared": "local",
				"tags":   []string{"global", "local"},
			},
		},
		"with client metadata": {
			local: beat.ProcessingConfig{
				Meta: mapstr.M{"index": "test"},
			},
			event:    `{"value": "abc"}`,
			want:     mapstr.M{"value": "abc"},
			wantMeta: mapstr.M{"index": "test"},
		},
		"with client processor": {
			local: beat.ProcessingConfig{
				Processor: func() beat.ProcessorList {
					g := newGroup("test", logp.L())
					g.add(actions.NewAddFields(mapstr.M{"custom": "value"}, true, true))
					return g
				}(),
			},
			event: `{"value": "abc"}`,
			want:  mapstr.M{"value": "abc", "custom": "value"},
		},
		"with beat default fields": {
			factory: MakeDefaultBeatSupport(true),
			global:  `{fields: {global: a, agent.foo: bar}, fields_under_root: true, tags: [tag]}`,
			event:   `{"value": "abc"}`,
			want: mapstr.M{
				"ecs": ecsFields,
				"host": mapstr.M{
					"name": "test.host.name",
				},
				"agent": mapstr.M{
					"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
					"name":         "test.host.name",
					"id":           "123e4567-e89b-12d3-a456-426655440001",
					"type":         "test",
					"version":      "0.1",
					"foo":          "bar",
				},
				"value":  "abc",
				"global": "a",
				"tags":   []string{"tag"},
			},
		},
		"with beat default fields and custom name": {
			factory: MakeDefaultBeatSupport(true),
			global:  `{fields: {global: a, agent.foo: bar}, fields_under_root: true, tags: [tag]}`,
			event:   `{"value": "abc"}`,
			infoMod: func(info beat.Info) beat.Info {
				info.Name = "other.test.host.name"
				return info
			},
			want: mapstr.M{
				"ecs": ecsFields,
				"host": mapstr.M{
					"name": "other.test.host.name",
				},
				"agent": mapstr.M{
					"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
					"name":         "other.test.host.name",
					"id":           "123e4567-e89b-12d3-a456-426655440001",
					"type":         "test",
					"version":      "0.1",
					"foo":          "bar",
				},
				"value":  "abc",
				"global": "a",
				"tags":   []string{"tag"},
			},
		},
		"with observer default fields": {
			factory: MakeDefaultObserverSupport(false),
			global:  `{fields: {global: a, observer.foo: bar}, fields_under_root: true, tags: [tag]}`,
			event:   `{"value": "abc"}`,
			want: mapstr.M{
				"ecs": ecsFields,
				"observer": mapstr.M{
					"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
					"hostname":     "test.host.name",
					"id":           "123e4567-e89b-12d3-a456-426655440001",
					"type":         "test",
					"version":      "0.1",
					"foo":          "bar",
				},
				"value":  "abc",
				"global": "a",
				"tags":   []string{"tag"},
			},
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := common.NewConfigWithYAML([]byte(test.global), "test")
			require.NoError(t, err)

			info := defaultInfo
			if test.infoMod != nil {
				info = test.infoMod(info)
			}

			factory := test.factory
			if factory == nil {
				factory = MakeDefaultSupport(true)
			}

			support, err := factory(info, logp.L(), cfg)
			require.NoError(t, err)

			prog, err := support.Create(test.local, test.drop)
			require.NoError(t, err)

			actual, err := prog.Run(&beat.Event{
				Timestamp: time.Now(),
				Fields:    fromJSON(test.event),
			})
			require.NoError(t, err)

			// validate
			assert.Equal(t, test.want, actual.Fields)
			assert.Equal(t, test.wantMeta, actual.Meta)
		})
	}
}

func TestNormalization(t *testing.T) {
	cases := map[string]struct {
		normalize bool
		in        mapstr.M
		mod       mapstr.M
		want      mapstr.M
	}{
		"no sharing if normalized": {
			normalize: true,
			in:        mapstr.M{"a": "b"},
			mod:       mapstr.M{"change": "x"},
			want:      mapstr.M{"a": "b"},
		},
		"data sharing if not normalized": {
			normalize: false,
			in:        mapstr.M{"a": "b"},
			mod:       mapstr.M{"change": "x"},
			want:      mapstr.M{"a": "b", "change": "x"},
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s, err := MakeDefaultSupport(test.normalize)(beat.Info{}, logp.L(), common.NewConfig())
			require.NoError(t, err)

			prog, err := s.Create(beat.ProcessingConfig{}, false)
			require.NoError(t, err)

			fields := test.in.Clone()
			actual, err := prog.Run(&beat.Event{Fields: fields})
			require.NoError(t, err)
			require.NotNil(t, actual)

			fields.DeepUpdate(test.mod)
			assert.Equal(t, test.want, actual.Fields)

			err = s.Close()
			require.NoError(t, err)
		})
	}
}

func BenchmarkNormalization(b *testing.B) {
	s, err := MakeDefaultSupport(true)(beat.Info{}, logp.L(), common.NewConfig())
	require.NoError(b, err)

	prog, err := s.Create(beat.ProcessingConfig{}, false)
	require.NoError(b, err)

	fields := mapstr.M{"a": "b"}
	for i := 0; i < b.N; i++ {
		f := fields.Clone()
		_, _ = prog.Run(&beat.Event{Fields: f})
	}
}

func TestAlwaysDrop(t *testing.T) {
	s, err := MakeDefaultSupport(true)(beat.Info{}, logp.L(), common.NewConfig())
	require.NoError(t, err)

	prog, err := s.Create(beat.ProcessingConfig{}, true)
	require.NoError(t, err)

	actual, err := prog.Run(&beat.Event{})
	require.NoError(t, err)
	assert.Nil(t, actual)

	err = s.Close()
	require.NoError(t, err)
}

func TestDynamicFields(t *testing.T) {
	factory, err := MakeDefaultSupport(true)(beat.Info{}, logp.L(), common.NewConfig())
	require.NoError(t, err)

	dynFields := mapstr.NewPointer(mapstr.M{})
	prog, err := factory.Create(beat.ProcessingConfig{
		DynamicFields: &dynFields,
	}, false)
	require.NoError(t, err)

	actual, err := prog.Run(&beat.Event{Fields: mapstr.M{"hello": "world"}})
	require.NoError(t, err)
	assert.Equal(t, mapstr.M{"hello": "world"}, actual.Fields)

	dynFields.Set(mapstr.M{"dyn": "field"})
	actual, err = prog.Run(&beat.Event{Fields: mapstr.M{"hello": "world"}})
	require.NoError(t, err)
	assert.Equal(t, mapstr.M{"hello": "world", "dyn": "field"}, actual.Fields)

	err = factory.Close()
	require.NoError(t, err)
}

func TestProcessingClose(t *testing.T) {
	factory, err := MakeDefaultSupport(true)(beat.Info{}, logp.L(), common.NewConfig())
	require.NoError(t, err)

	// Inject a processor in the builder that we can check if has been closed.
	factoryProcessor := &processorWithClose{}
	b := factory.(*builder)
	if b.processors == nil {
		b.processors = newGroup("global", logp.L())
	}
	b.processors.add(factoryProcessor)

	clientProcessor := &processorWithClose{}
	g := newGroup("test", logp.L())
	g.add(clientProcessor)

	prog, err := factory.Create(beat.ProcessingConfig{
		Processor: g,
	}, false)
	require.NoError(t, err)

	// Check that both processors are called
	assert.False(t, factoryProcessor.called)
	assert.False(t, clientProcessor.called)
	_, err = prog.Run(&beat.Event{Fields: mapstr.M{"hello": "world"}})
	require.NoError(t, err)
	assert.True(t, factoryProcessor.called)
	assert.True(t, clientProcessor.called)

	// Check that closing the client processing pipeline doesn't close the global pipeline
	assert.False(t, factoryProcessor.closed)
	assert.False(t, clientProcessor.closed)
	err = processors.Close(prog)
	require.NoError(t, err)
	assert.False(t, factoryProcessor.closed)
	assert.True(t, clientProcessor.closed)

	// Check that closing the factory closes the processor in the global pipeline
	err = factory.Close()
	require.NoError(t, err)
	assert.True(t, factoryProcessor.closed)
}

func fromJSON(in string) mapstr.M {
	var tmp mapstr.M
	err := json.Unmarshal([]byte(in), &tmp)
	if err != nil {
		panic(err)
	}
	return tmp
}

type processorWithClose struct {
	closed bool
	called bool
}

func (p *processorWithClose) Run(e *beat.Event) (*beat.Event, error) {
	p.called = true
	return e, nil
}

func (p *processorWithClose) Close() error {
	p.closed = true
	return nil
}

func (p *processorWithClose) String() string {
	return "processorWithClose"
}
