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

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors/actions"
	"github.com/elastic/ecs/code/go/ecs"
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

	ecsFields := common.MapStr{"version": ecs.Version}

	cases := map[string]struct {
		factory  SupportFactory
		global   string
		local    beat.ProcessingConfig
		drop     bool
		event    string
		want     common.MapStr
		wantMeta common.MapStr
		infoMod  func(beat.Info) beat.Info
	}{
		"user global fields and tags": {
			global: "{fields: {global: 1}, fields_under_root: true, tags: [tag]}",
			event:  `{"value": "abc"}`,
			want: common.MapStr{
				"value":  "abc",
				"global": uint64(1),
				"tags":   []string{"tag"},
			},
		},
		"beat local fields": {
			global: "",
			local: beat.ProcessingConfig{
				Fields: common.MapStr{"local": 1},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value": "abc",
				"local": 1,
			},
		},
		"beat local and user global fields": {
			global: "{fields: {global: 1}, fields_under_root: true, tags: [tag]}",
			local: beat.ProcessingConfig{
				Fields: common.MapStr{"local": 1},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value":  "abc",
				"global": uint64(1),
				"local":  1,
				"tags":   []string{"tag"},
			},
		},
		"user global fields overwrite beat local fields": {
			global: "{fields: {global: a, shared: global}, fields_under_root: true}",
			local: beat.ProcessingConfig{
				Fields: common.MapStr{"local": "b", "shared": "local"},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value":  "abc",
				"local":  "b",
				"global": "a",
				"shared": "global",
			},
		},
		"user local fields and tags": {
			local: beat.ProcessingConfig{
				EventMetadata: common.EventMetadata{
					Fields: common.MapStr{"local": "a"},
					Tags:   []string{"tag"},
				},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value": "abc",
				"fields": common.MapStr{
					"local": "a",
				},
				"tags": []string{"tag"},
			},
		},
		"user local fields (under root) and tags": {
			local: beat.ProcessingConfig{
				EventMetadata: common.EventMetadata{
					Fields:          common.MapStr{"local": "a"},
					FieldsUnderRoot: true,
					Tags:            []string{"tag"},
				},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value": "abc",
				"local": "a",
				"tags":  []string{"tag"},
			},
		},
		"user local fields overwrite user global fields": {
			global: `{fields: {global: a, shared: global}, fields_under_root: true, tags: [global]}`,
			local: beat.ProcessingConfig{
				EventMetadata: common.EventMetadata{
					Fields: common.MapStr{
						"local":  "a",
						"shared": "local",
					},
					FieldsUnderRoot: true,
					Tags:            []string{"local"},
				},
			},
			event: `{"value": "abc"}`,
			want: common.MapStr{
				"value":  "abc",
				"global": "a",
				"local":  "a",
				"shared": "local",
				"tags":   []string{"global", "local"},
			},
		},
		"with client metadata": {
			local: beat.ProcessingConfig{
				Meta: common.MapStr{"index": "test"},
			},
			event:    `{"value": "abc"}`,
			want:     common.MapStr{"value": "abc"},
			wantMeta: common.MapStr{"index": "test"},
		},
		"with client processor": {
			local: beat.ProcessingConfig{
				Processor: func() beat.ProcessorList {
					g := newGroup("test", logp.L())
					g.add(actions.NewAddFields(common.MapStr{"custom": "value"}, true, true))
					return g
				}(),
			},
			event: `{"value": "abc"}`,
			want:  common.MapStr{"value": "abc", "custom": "value"},
		},
		"with beat default fields": {
			factory: MakeDefaultBeatSupport(true),
			global:  `{fields: {global: a, agent.foo: bar}, fields_under_root: true, tags: [tag]}`,
			event:   `{"value": "abc"}`,
			want: common.MapStr{
				"ecs": ecsFields,
				"host": common.MapStr{
					"name": "test.host.name",
				},
				"agent": common.MapStr{
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
		"with beat default fields and custom name": {
			factory: MakeDefaultBeatSupport(true),
			global:  `{fields: {global: a, agent.foo: bar}, fields_under_root: true, tags: [tag]}`,
			event:   `{"value": "abc"}`,
			infoMod: func(info beat.Info) beat.Info {
				info.Name = "other.test.host.name"
				return info
			},
			want: common.MapStr{
				"ecs": ecsFields,
				"host": common.MapStr{
					"name": "other.test.host.name",
				},
				"agent": common.MapStr{
					"ephemeral_id": "123e4567-e89b-12d3-a456-426655440000",
					"hostname":     "test.host.name",
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
			want: common.MapStr{
				"ecs": ecsFields,
				"observer": common.MapStr{
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
		in        common.MapStr
		mod       common.MapStr
		want      common.MapStr
	}{
		"no sharing if normalized": {
			normalize: true,
			in:        common.MapStr{"a": "b"},
			mod:       common.MapStr{"change": "x"},
			want:      common.MapStr{"a": "b"},
		},
		"data sharing if not normalized": {
			normalize: false,
			in:        common.MapStr{"a": "b"},
			mod:       common.MapStr{"change": "x"},
			want:      common.MapStr{"a": "b", "change": "x"},
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
		})
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
}

func TestDynamicFields(t *testing.T) {
	factory, err := MakeDefaultSupport(true)(beat.Info{}, logp.L(), common.NewConfig())
	require.NoError(t, err)

	dynFields := common.NewMapStrPointer(common.MapStr{})
	prog, err := factory.Create(beat.ProcessingConfig{
		DynamicFields: &dynFields,
	}, false)
	require.NoError(t, err)

	actual, err := prog.Run(&beat.Event{Fields: common.MapStr{"hello": "world"}})
	require.NoError(t, err)
	assert.Equal(t, common.MapStr{"hello": "world"}, actual.Fields)

	dynFields.Set(common.MapStr{"dyn": "field"})
	actual, err = prog.Run(&beat.Event{Fields: common.MapStr{"hello": "world"}})
	require.NoError(t, err)
	assert.Equal(t, common.MapStr{"hello": "world", "dyn": "field"}, actual.Fields)
}

func fromJSON(in string) common.MapStr {
	var tmp common.MapStr
	err := json.Unmarshal([]byte(in), &tmp)
	if err != nil {
		panic(err)
	}
	return tmp
}
