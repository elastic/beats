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
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/ecs"
	"github.com/elastic/beats/v7/libbeat/management"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/actions/addfields"
	"github.com/elastic/beats/v7/pkg/config"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/logp/logptest"
	"github.com/elastic/beats/v7/pkg/mapstr"
	"github.com/elastic/beats/v7/pkg/paths"

	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_docker_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_kubernetes_metadata"
)

func TestGenerateProcessorList(t *testing.T) {
	inputCfg := []mapstr.M{
		{"add_host_metadata": nil},
		{"add_cloud_metadata": nil},
		{"add_docker_metadata": nil},
		{"add_kubernetes_metadata": nil},
	}

	plugins, err := processors.NewPluginConfigFromList(inputCfg)
	require.NoError(t, err)

	processors, err := processors.New(plugins, logptest.NewTestingLogger(t, ""))
	require.NoError(t, err)
	// make sure the processor init got the config formatted in a way it expected
	require.Len(t, processors.List, 4)

	require.NoError(t, processors.Close())
}

func TestProcessorsConfigs(t *testing.T) {
	defaultInfo := beat.Info{
		Beat:        "test",
		EphemeralID: uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440000")),
		Hostname:    "test.host.name",
		ID:          uuid.Must(uuid.FromString("123e4567-e89b-12d3-a456-426655440001")),
		Name:        "test.host.name",
		Version:     "0.1",
		Paths:       tmpPaths(t),
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
					g.add(addfields.NewAddFields(mapstr.M{"custom": "value"}, true, true))
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
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cfg, err := config.NewConfigWithYAML([]byte(test.global), "test")
			require.NoError(t, err)

			info := defaultInfo
			if test.infoMod != nil {
				info = test.infoMod(info)
			}

			factory := test.factory
			if factory == nil {
				factory = MakeDefaultSupport(true, nil)
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

// TestEventNormalizationOverride verifies that the EventNormalization option
// in beat.ProcessingConfig overrides the "skipNormalize" setting that is
// specified in the builder (this is the default value set by the Beat).
func TestEventNormalizationOverride(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	testCases := []struct {
		skipNormalize          bool
		normalizeOverride      *bool
		hasGeneralizeProcessor bool
	}{
		{skipNormalize: false, normalizeOverride: nil, hasGeneralizeProcessor: true},
		{skipNormalize: false, normalizeOverride: boolPtr(false), hasGeneralizeProcessor: false},
		{skipNormalize: false, normalizeOverride: boolPtr(true), hasGeneralizeProcessor: true},
		{skipNormalize: true, normalizeOverride: nil, hasGeneralizeProcessor: false},
		{skipNormalize: true, normalizeOverride: boolPtr(false), hasGeneralizeProcessor: false},
		{skipNormalize: true, normalizeOverride: boolPtr(true), hasGeneralizeProcessor: true},
	}

	for _, tc := range testCases {
		builder, err := newBuilder(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), nil, mapstr.EventMetadata{}, nil, tc.skipNormalize, false)
		require.NoError(t, err)

		processor, err := builder.Create(beat.ProcessingConfig{EventNormalization: tc.normalizeOverride}, false)
		require.NoError(t, err)
		group, ok := processor.(*group)
		require.True(t, ok)

		if tc.hasGeneralizeProcessor {
			if assert.NotEmpty(t, group.list) {
				assert.Equal(t, "generalizeEvent", group.list[0].String())
			}
		} else {
			assert.Empty(t, group.list)
		}
	}
}

func TestNormalization(t *testing.T) {
	cases := map[string]struct {
		normalize bool
		inPlace   bool
		in        mapstr.M
		mod       mapstr.M
		want      mapstr.M
	}{
		"no sharing if normalized": {
			normalize: true,
			in:        mapstr.M{"a": "b", "nested": mapstr.M{"n": "1"}},
			mod:       mapstr.M{"change": "x", "nested": mapstr.M{"deep": "y"}},
			want:      mapstr.M{"a": "b", "nested": mapstr.M{"n": "1"}},
		},
		"data sharing if not normalized": {
			normalize: false,
			in:        mapstr.M{"a": "b"},
			mod:       mapstr.M{"change": "x"},
			want:      mapstr.M{"a": "b", "change": "x"},
		},
		"data sharing if normalized in place": {
			normalize: true,
			inPlace:   true,
			in:        mapstr.M{"a": "b", "drop": nil, "nested": mapstr.M{"n": "1", "drop": nil}},
			mod:       mapstr.M{"change": "x", "nested": mapstr.M{"deep": "y"}},
			want:      mapstr.M{"a": "b", "change": "x", "nested": mapstr.M{"n": "1", "deep": "y"}},
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s, err := MakeDefaultSupport(test.normalize, nil)(beat.Info{Paths: tmpPaths(t)}, logp.L(), config.NewConfig())
			require.NoError(t, err)

			prog, err := s.Create(beat.ProcessingConfig{NormalizeInPlace: test.inPlace}, false)
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
	s, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(b)}, logp.L(), config.NewConfig())
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
	s, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.L(), config.NewConfig())
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
	factory, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.L(), config.NewConfig())
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
	factory, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.L(), config.NewConfig())
	require.NoError(t, err)

	// Inject a processor in the builder that we can check if has been closed.
	factoryProcessor := &processorWithClose{}
	b, ok := factory.(*builder)
	require.True(t, ok)
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

func TestCreateSharesGeneralize(t *testing.T) {
	support, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	clientProcessors := newGroup("test", logp.NewNopLogger())
	clientProcessors.add(&processorWithClose{})

	cases := map[string]struct {
		a, b beat.ProcessingConfig
		same bool
	}{
		"default config": {
			same: true,
		},
		"keep null and in place": {
			a:    beat.ProcessingConfig{KeepNull: true, NormalizeInPlace: true},
			b:    beat.ProcessingConfig{KeepNull: true, NormalizeInPlace: true},
			same: true,
		},
		"one with client processors": {
			b:    beat.ProcessingConfig{Processor: clientProcessors},
			same: true,
		},
		"different keep null": {
			b: beat.ProcessingConfig{KeepNull: true},
		},
		"different in place": {
			b: beat.ProcessingConfig{NormalizeInPlace: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			a, err := support.Create(tc.a, false)
			require.NoError(t, err)
			b, err := support.Create(tc.b, false)
			require.NoError(t, err)

			same := chainProcessor(t, a, "generalizeEvent") == chainProcessor(t, b, "generalizeEvent")
			assert.Equal(t, tc.same, same, "chains share the generalize processor")
		})
	}
}

func TestCreateSelectsGeneralizeVariant(t *testing.T) {
	support, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	cases := map[string]struct {
		cfg         beat.ProcessingConfig
		want        mapstr.M
		wantInPlace bool
	}{
		"default": {
			want: mapstr.M{"a": "b"},
		},
		"keep null": {
			cfg:  beat.ProcessingConfig{KeepNull: true},
			want: mapstr.M{"a": "b", "null": nil},
		},
		"in place": {
			cfg:         beat.ProcessingConfig{NormalizeInPlace: true},
			want:        mapstr.M{"a": "b"},
			wantInPlace: true,
		},
		"keep null and in place": {
			cfg:         beat.ProcessingConfig{KeepNull: true, NormalizeInPlace: true},
			want:        mapstr.M{"a": "b", "null": nil},
			wantInPlace: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			chain, err := support.Create(tc.cfg, false)
			require.NoError(t, err)

			in := mapstr.M{"a": "b", "null": nil}
			out, err := chain.Run(&beat.Event{Timestamp: time.Now(), Fields: in})
			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, tc.want, out.Fields)

			in["added"] = "later"
			_, sharesInput := out.Fields["added"]
			assert.Equal(t, tc.wantInPlace, sharesInput, "output fields alias the input map")
		})
	}
}

// TestCreateSharedProcessorsConcurrentUse runs one chain per goroutine, as
// pipeline clients do, so the processors the chains share run concurrently.
func TestCreateSharedProcessorsConcurrentUse(t *testing.T) {
	info, builtin := testBuiltinFields(t)
	support, err := MakeDefaultBeatSupport(true)(info, logp.NewNopLogger(), config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	const goroutines, events = 16, 200
	var wg sync.WaitGroup
	for g := range goroutines {
		cfg := beat.ProcessingConfig{KeepNull: g%2 == 1, NormalizeInPlace: g%8 >= 4}
		if g%4 >= 2 {
			clientProcessors := newGroup("test", logp.NewNopLogger())
			clientProcessors.add(addfields.NewAddFields(mapstr.M{"client": g}, true, true))
			cfg.Processor = clientProcessors
		}
		chain, err := support.Create(cfg, false)
		require.NoError(t, err)

		wg.Go(func() {
			for i := range events {
				id := fmt.Sprintf("%d-%d", g, i)
				fields := mapstr.M{"id": id, "nested": mapstr.M{"n": i, "null": nil}}
				out, err := chain.Run(&beat.Event{Timestamp: time.Now(), Fields: fields})
				if !assert.NoError(t, err) || !assert.NotNil(t, out) {
					return
				}

				want := builtin.Clone()
				want["id"] = id
				want["nested"] = mapstr.M{"n": i}
				if cfg.KeepNull {
					want["nested"] = mapstr.M{"n": i, "null": nil}
				}
				if cfg.Processor != nil {
					want["client"] = g
				}
				assert.Equal(t, want, out.Fields)
			}
		})
	}
	wg.Wait()
}

func TestCreateCloseKeepsSharedProcessors(t *testing.T) {
	info, builtin := testBuiltinFields(t)
	support, err := MakeDefaultBeatSupport(true)(info, logp.NewNopLogger(), config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	b, ok := support.(*builder)
	require.Truef(t, ok, "support is %T, want *builder", support)
	shared := append(slices.Collect(maps.Values(b.generalize)), b.debugPrint)
	for _, p := range shared {
		assert.NotImplementsf(t, (*processors.Closer)(nil), p, "shared processor %s", p)
	}

	closing := &processorWithClose{}
	clientProcessors := newGroup("test", logp.NewNopLogger())
	clientProcessors.add(closing)
	closed, err := support.Create(beat.ProcessingConfig{Processor: clientProcessors}, false)
	require.NoError(t, err)
	open, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)

	require.NoError(t, processors.Close(closed))
	assert.True(t, closing.closed, "client processor of the closed chain")

	out, err := open.Run(&beat.Event{Timestamp: time.Now(), Fields: mapstr.M{"a": "b"}})
	require.NoError(t, err)
	require.NotNil(t, out)
	want := builtin.Clone()
	want["a"] = "b"
	assert.Equal(t, want, out.Fields)
}

// TestCreateReadOnlyFields changes the client's fields after Create to observe
// whether the chain kept that map or a copy of it.
func TestCreateReadOnlyFields(t *testing.T) {
	const late = "added after Create"
	dynamic := mapstr.NewPointer(mapstr.M{"dyn": "d"})
	clientProcessors := processors.NewList(logp.NewNopLogger())
	clientProcessors.AddProcessor(addfields.NewAddFields(mapstr.M{"proc": "p"}, true, true))

	cases := map[string]struct {
		globalFields mapstr.M
		cfg          beat.ProcessingConfig
		wantKept     bool
		wantAliased  bool
		wantExtra    mapstr.M
	}{
		"read only": {
			cfg:         beat.ProcessingConfig{FieldsReadOnly: true},
			wantKept:    true,
			wantAliased: true,
		},
		"not read only": {},
		"read only with global fields": {
			globalFields: mapstr.M{"g": "global"},
			cfg:          beat.ProcessingConfig{FieldsReadOnly: true},
			wantExtra:    mapstr.M{"fields": mapstr.M{"g": "global"}},
		},
		"read only with client event metadata fields": {
			cfg: beat.ProcessingConfig{
				FieldsReadOnly: true,
				EventMetadata:  mapstr.EventMetadata{Fields: mapstr.M{"c": "client"}},
			},
			wantExtra: mapstr.M{"fields": mapstr.M{"c": "client"}},
		},
		"read only with client processors": {
			cfg:       beat.ProcessingConfig{FieldsReadOnly: true, Processor: clientProcessors},
			wantKept:  true,
			wantExtra: mapstr.M{"proc": "p"},
		},
		"read only with dynamic fields": {
			cfg:       beat.ProcessingConfig{FieldsReadOnly: true, DynamicFields: &dynamic},
			wantKept:  true,
			wantExtra: mapstr.M{"dyn": "d"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			beatCfg := config.NewConfig()
			if tc.globalFields != nil {
				beatCfg = config.MustNewConfigFrom(mapstr.M{"fields": tc.globalFields})
			}
			support, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), beatCfg)
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, support.Close()) })

			input := mapstr.M{"type": "filestream"}
			cfg := tc.cfg
			cfg.Fields = mapstr.M{"input": input}
			chain, err := support.Create(cfg, false)
			require.NoError(t, err)
			input["late"] = late

			out, err := chain.Run(&beat.Event{Timestamp: time.Now(), Fields: mapstr.M{"message": "m"}})
			require.NoError(t, err)
			require.NotNil(t, out)

			wantInput := mapstr.M{"type": "filestream"}
			if tc.wantKept {
				wantInput["late"] = late
			}
			want := mapstr.M{"message": "m", "input": wantInput}
			want.DeepUpdate(tc.wantExtra)
			assert.Equal(t, want, out.Fields)
			assert.Equal(t, mapstr.M{"input": mapstr.M{"type": "filestream", "late": late}}, cfg.Fields,
				"client fields must not be modified by Create or Run")
			outInput, _ := out.Fields["input"].(mapstr.M)
			assert.Equal(t, tc.wantAliased, sameMap(input, outInput), "event references the client's input map")
		})
	}
}

// TestCreateReadOnlyFieldsConcurrentUse runs one chain per goroutine on one
// read-only fields map, so a write to that map by any chain shows as a race.
func TestCreateReadOnlyFieldsConcurrentUse(t *testing.T) {
	eventTime := time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	fieldsTime := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	cases := map[string]struct {
		fields      mapstr.M
		wantMeta    mapstr.M
		wantTime    time.Time
		wantAliased bool
	}{
		"input fields": {
			fields:      mapstr.M{"input": mapstr.M{"type": "filestream"}},
			wantTime:    eventTime,
			wantAliased: true,
		},
		"metadata key": {
			fields:   mapstr.M{"input": mapstr.M{"type": "filestream"}, beat.MetadataFieldKey: mapstr.M{"pipeline": "p"}},
			wantMeta: mapstr.M{"pipeline": "p"},
			wantTime: eventTime,
		},
		"timestamp key": {
			fields:   mapstr.M{"input": mapstr.M{"type": "filestream"}, beat.TimestampFieldKey: fieldsTime},
			wantTime: fieldsTime,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			support, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), config.NewConfig())
			require.NoError(t, err)
			t.Cleanup(func() { assert.NoError(t, support.Close()) })

			shared := tc.fields.Clone()
			sharedInput, _ := shared["input"].(mapstr.M)
			const goroutines, events = 8, 200
			var wg sync.WaitGroup
			for g := range goroutines {
				chain, err := support.Create(beat.ProcessingConfig{Fields: shared, FieldsReadOnly: true}, false)
				require.NoError(t, err)

				wg.Go(func() {
					for i := range events {
						id := fmt.Sprintf("%d-%d", g, i)
						out, err := chain.Run(&beat.Event{Timestamp: eventTime, Fields: mapstr.M{"id": id}})
						if !assert.NoError(t, err) || !assert.NotNil(t, out) {
							return
						}
						assert.Equal(t, mapstr.M{"id": id, "input": mapstr.M{"type": "filestream"}}, out.Fields)
						assert.Equal(t, tc.wantMeta, out.Meta)
						assert.Equal(t, tc.wantTime, out.Timestamp)
						outInput, _ := out.Fields["input"].(mapstr.M)
						assert.Equal(t, tc.wantAliased, sameMap(sharedInput, outInput), "event references the shared input map")
					}
				})
			}
			wg.Wait()
			assert.Equal(t, tc.fields, shared, "shared fields map after all events")
		})
	}
}

func TestDebugPrintOutput(t *testing.T) {
	log, observed := logptest.NewTestingLoggerWithObserver(t, "")
	support, err := MakeDefaultSupport(true, nil)(beat.Info{Beat: "test", Version: "0.1", Paths: tmpPaths(t)}, log, config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	chain, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)
	_, err = chain.Run(&beat.Event{
		Timestamp: time.Date(2026, 11, 7, 12, 30, 0, 123000000, time.UTC),
		Meta:      mapstr.M{"id": "abc"},
		// One key per map keeps the encoded key order deterministic.
		Fields: mapstr.M{"nested": mapstr.M{"list": []any{"<a & b>", 1, 1.5}}},
	})
	require.NoError(t, err)

	const want = `Publish event: {
  "@timestamp": "2026-11-07T12:30:00.123Z",
  "@metadata": {
    "beat": "test",
    "type": "_doc",
    "version": "0.1",
    "id": "abc"
  },
  "nested": {
    "list": [
      "<a & b>",
      1,
      1.5
    ]
  }
}`
	entries := observed.FilterMessageSnippet("Publish event").All()
	require.Len(t, entries, 1, "debug event log entries")
	assert.Equal(t, want, entries[0].Message)
	assert.Equal(t, map[string]any{logp.TypeKey: logp.EventType}, entries[0].ContextMap())
}

func TestCreateSharesDebugPrint(t *testing.T) {
	underAgent := management.UnderAgent()
	t.Cleanup(func() { management.SetUnderAgent(underAgent) })

	support, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.NewNopLogger(), config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	management.SetUnderAgent(false)
	standalone, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)
	assert.NotContains(t, standalone.String(), "debugPrint", "chain without debug logging or Agent")

	clientProcessors := newGroup("test", logp.NewNopLogger())
	clientProcessors.add(&processorWithClose{})
	management.SetUnderAgent(true)
	a, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)
	b, err := support.Create(beat.ProcessingConfig{KeepNull: true, Processor: clientProcessors}, false)
	require.NoError(t, err)
	assert.Same(t, chainProcessor(t, a, "debugPrint"), chainProcessor(t, b, "debugPrint"), "debugPrint processor under Agent")
}

// TestDebugPrintAfterEncoderPanic checks that an event that makes the shared
// debugPrint processor's encoder panic does not stop other clients' events
// from being logged.
func TestDebugPrintAfterEncoderPanic(t *testing.T) {
	log, observed := logptest.NewTestingLoggerWithObserver(t, "")
	support, err := MakeDefaultSupport(true, nil)(beat.Info{Beat: "test", Version: "0.1", Paths: tmpPaths(t)}, log, config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	panicking, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)
	healthy, err := support.Create(beat.ProcessingConfig{}, false)
	require.NoError(t, err)

	// Meta is not normalized, and the encoder panics on the typed nil
	// *time.Time after it has started the array.
	assert.Panics(t, func() {
		_, _ = panicking.Run(&beat.Event{
			Timestamp: time.Now(),
			Meta:      mapstr.M{"ts": []any{"first", (*time.Time)(nil)}},
			Fields:    mapstr.M{"a": "b"},
		})
	}, "encoding a typed nil *time.Time")

	const events = 5
	for range events {
		_, err := healthy.Run(&beat.Event{Timestamp: time.Now(), Fields: mapstr.M{"a": "b"}})
		require.NoError(t, err)
	}
	assert.Len(t, observed.FilterMessageSnippet("Publish event").All(), events, "debug log entries of the healthy client")
}

// TestDebugPrintConcurrentUse runs one chain per goroutine with debug logging
// on, so the shared debugPrint processor encodes events concurrently.
func TestDebugPrintConcurrentUse(t *testing.T) {
	log, observed := logptest.NewTestingLoggerWithObserver(t, "")
	support, err := MakeDefaultSupport(true, nil)(beat.Info{Beat: "test", Version: "0.1", Paths: tmpPaths(t)}, log, config.NewConfig())
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, support.Close()) })

	const format = `Publish event: {
  "@timestamp": "2026-11-07T12:30:00.000Z",
  "@metadata": {
    "beat": "test",
    "type": "_doc",
    "version": "0.1"
  },
  "id": %q
}`
	const goroutines, events = 16, 100
	var want []string
	var wg sync.WaitGroup
	for g := range goroutines {
		chain, err := support.Create(beat.ProcessingConfig{}, false)
		require.NoError(t, err)

		ids := make([]string, events)
		for i := range ids {
			ids[i] = fmt.Sprintf("%d-%d-%s", g, i, strings.Repeat("x", i%10))
			want = append(want, fmt.Sprintf(format, ids[i]))
		}
		wg.Go(func() {
			for _, id := range ids {
				_, err := chain.Run(&beat.Event{
					Timestamp: time.Date(2026, 11, 7, 12, 30, 0, 0, time.UTC),
					Fields:    mapstr.M{"id": id},
				})
				assert.NoError(t, err)
			}
		})
	}
	wg.Wait()

	var got []string
	for _, entry := range observed.FilterMessageSnippet("Publish event").All() {
		got = append(got, entry.Message)
	}
	slices.Sort(want)
	slices.Sort(got)
	assert.Equal(t, want, got, "debug event log messages")
}

func TestProcessingDiagnostics(t *testing.T) {
	factory, err := MakeDefaultSupport(true, nil)(beat.Info{Paths: tmpPaths(t)}, logp.L(), config.NewConfig())
	require.NoError(t, err)

	p := factory.Processors()
	assert.Empty(t, p)
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

func tmpPaths(t testing.TB) *paths.Path {
	dir := t.TempDir()
	return &paths.Path{
		Home:   dir,
		Config: dir,
		Data:   dir,
		Logs:   dir,
	}
}

// testBuiltinFields returns a beat.Info and the builtin fields that
// MakeDefaultBeatSupport adds to every event for it.
func testBuiltinFields(t *testing.T) (beat.Info, mapstr.M) {
	info := beat.Info{
		Beat:    "test",
		Name:    "test.host.name",
		Version: "0.1",
		Paths:   tmpPaths(t),
	}
	return info, mapstr.M{
		"ecs":  mapstr.M{"version": ecs.Version},
		"host": mapstr.M{"name": "test.host.name"},
		"agent": mapstr.M{
			"ephemeral_id": uuid.Nil.String(),
			"id":           uuid.Nil.String(),
			"name":         "test.host.name",
			"type":         "test",
			"version":      "0.1",
		},
	}
}

// chainProcessor returns the first processor of chain whose String starts with
// prefix.
func chainProcessor(t *testing.T, chain beat.Processor, prefix string) beat.Processor {
	t.Helper()
	g, ok := chain.(*group)
	require.Truef(t, ok, "chain is %T, want *group", chain)
	for _, p := range g.list {
		if strings.HasPrefix(p.String(), prefix) {
			return p
		}
	}
	require.Failf(t, "processor not found", "no processor with prefix %q in %s", prefix, g)
	return nil
}

func sameMap(a, b mapstr.M) bool {
	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}
