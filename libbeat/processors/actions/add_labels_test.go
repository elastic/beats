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

package actions

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

func TestAddLabels(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	cases := map[string]struct {
		event common.MapStr
		want  common.MapStr
		cfg   []string
	}{
		"add label": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{"label": "test"},
			},
			cfg: single(`{labels: {label: test}}`),
		},
		"custom target": {
			event: common.MapStr{},
			want: common.MapStr{
				"my": common.MapStr{"label": "test"},
			},
			cfg: single(`{target: my, labels: {label: test}}`),
		},
		"overwrite existing label": {
			event: common.MapStr{
				"labels": common.MapStr{"label": "old"},
			},
			want: common.MapStr{
				"labels": common.MapStr{"label": "test"},
			},
			cfg: single(`{labels: {label: test}}`),
		},
		"merge with existing labels": {
			event: common.MapStr{
				"labels": common.MapStr{"existing": "a"},
			},
			want: common.MapStr{
				"labels": common.MapStr{"existing": "a", "label": "test"},
			},
			cfg: single(`{labels: {label: test}}`),
		},
		"combine 2 processors": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{
					"l1": "a",
					"l2": "b",
				},
			},
			cfg: multi(
				`{labels: {l1: a}}`,
				`{labels: {l2: b}}`,
			),
		},
		"different targets": {
			event: common.MapStr{},
			want: common.MapStr{
				"a": common.MapStr{"l1": "a"},
				"b": common.MapStr{"l2": "b"},
			},
			cfg: multi(
				`{target: a, labels: {l1: a}}`,
				`{target: b, labels: {l2: b}}`,
			),
		},
		"under root": {
			event: common.MapStr{},
			want: common.MapStr{
				"a": common.MapStr{"b": "test"},
			},
			cfg: single(
				`{target: "", labels: {a.b: test}}`,
			),
		},
		"merge under root": {
			event: common.MapStr{
				"a": common.MapStr{"old": "value"},
			},
			want: common.MapStr{
				"a": common.MapStr{"old": "value", "new": "test"},
			},
			cfg: single(
				`{target: "", labels: {a.new: test}}`,
			),
		},
		"overwrite existing under root": {
			event: common.MapStr{
				"a": common.MapStr{"keep": "value", "change": "a"},
			},
			want: common.MapStr{
				"a": common.MapStr{"keep": "value", "change": "b"},
			},
			cfg: single(
				`{target: "", labels: {a.change: b}}`,
			),
		},
	}

	for name, test := range cases {
		test := test
		t.Run(name, func(t *testing.T) {
			ps := make([]*processors.Processors, len(test.cfg))
			for i := range test.cfg {
				config, err := common.NewConfigWithYAML([]byte(test.cfg[i]), "test")
				if err != nil {
					t.Fatalf("Failed to create config(%v): %+v", i, err)
				}

				ps[i], err = processors.New(processors.PluginConfig{
					{
						"add_labels": config,
					},
				})
				if err != nil {
					t.Fatalf("Failed to create add_tags processor(%v): %+v", i, err)
				}
			}

			current := &beat.Event{Fields: test.event.Clone()}
			for i, processor := range ps {
				current = processor.Run(current)
				if current == nil {
					t.Fatalf("Event dropped(%v)", i)
				}
			}

			assert.Equal(t, test.want, current.Fields)
		})
	}
}
