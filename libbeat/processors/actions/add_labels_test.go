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

func TestAddLabels(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	testProcessors(t, map[string]testCase{
		"add label": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{"label": "test"},
			},
			cfg: single(`{add_labels: {labels: {label: test}}}`),
		},
		"add dotted label": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{"a.b": "test"},
			},
			cfg: single(`{add_labels: {labels: {a.b: test}}}`),
		},
		"add nested labels": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{"a.b": "test", "a.c": "test2"},
			},
			cfg: single(`{add_labels: {labels: {a: {b: test, c: test2}}}}`),
		},
		"merge labels": {
			event: common.MapStr{},
			want: common.MapStr{
				"labels": common.MapStr{"l1": "a", "l2": "b", "lc": "b"},
			},
			cfg: multi(
				`{add_labels.labels: {l1: a, lc: a}}`,
				`{add_labels.labels: {l2: b, lc: b}}`,
			),
		},
	})
}
