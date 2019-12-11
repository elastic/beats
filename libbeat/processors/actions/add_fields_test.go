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

	"github.com/elastic/beats/libbeat/common"
)

func TestAddFields(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	testProcessors(t, map[string]testCase{
		"add field": {
			event: common.MapStr{},
			want: common.MapStr{
				"fields": common.MapStr{"field": "test"},
			},
			cfg: single(`{add_fields: {fields: {field: test}}}`),
		},
		"custom target": {
			event: common.MapStr{},
			want: common.MapStr{
				"my": common.MapStr{"field": "test"},
			},
			cfg: single(`{add_fields: {target: my, fields: {field: test}}}`),
		},
		"overwrite existing field": {
			event: common.MapStr{
				"fields": common.MapStr{"field": "old"},
			},
			want: common.MapStr{"fields": common.MapStr{"field": "test"}},
			cfg:  single(`{add_fields: {fields: {field: test}}}`),
		},
		"merge with existing fields": {
			event: common.MapStr{
				"fields": common.MapStr{"existing": "a"},
			},
			want: common.MapStr{
				"fields": common.MapStr{"existing": "a", "field": "test"},
			},
			cfg: single(`{add_fields: {fields: {field: test}}}`),
		},
		"combine 2 processors": {
			event: common.MapStr{},
			want: common.MapStr{
				"fields": common.MapStr{
					"l1": "a",
					"l2": "b",
				},
			},
			cfg: multi(
				`{add_fields: {fields: {l1: a}}}`,
				`{add_fields: {fields: {l2: b}}}`,
			),
		},
		"different targets": {
			event: common.MapStr{},
			want: common.MapStr{
				"a": common.MapStr{"l1": "a"},
				"b": common.MapStr{"l2": "b"},
			},
			cfg: multi(
				`{add_fields: {target: a, fields: {l1: a}}}`,
				`{add_fields: {target: b, fields: {l2: b}}}`,
			),
		},
		"under root": {
			event: common.MapStr{},
			want: common.MapStr{
				"a": common.MapStr{"b": "test"},
			},
			cfg: single(
				`{add_fields: {target: "", fields: {a.b: test}}}`,
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
				`{add_fields: {target: "", fields: {a.new: test}}}`,
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
				`{add_fields: {target: "", fields: {a.change: b}}}`,
			),
		},
	})
}
