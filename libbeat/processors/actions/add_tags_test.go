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

	"github.com/elastic/beats/v8/libbeat/common"
)

func TestAddTags(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	testProcessors(t, map[string]testCase{
		"create tags": {
			eventFields: common.MapStr{},
			wantFields:  common.MapStr{"tags": []string{"t1", "t2"}},
			cfg:         single(`{add_tags: {tags: [t1, t2]}}`),
		},
		"append to tags": {
			eventFields: common.MapStr{"tags": []string{"t1"}},
			wantFields:  common.MapStr{"tags": []string{"t1", "t2", "t3"}},
			cfg:         single(`{add_tags: {tags: [t2, t3]}}`),
		},
		"combine from 2 processors": {
			eventFields: common.MapStr{},
			wantFields:  common.MapStr{"tags": []string{"t1", "t2", "t3", "t4"}},
			cfg: multi(
				`{add_tags: {tags: [t1, t2]}}`,
				`{add_tags: {tags: [t3, t4]}}`,
			),
		},
		"with custom target": {
			eventFields: common.MapStr{},
			wantFields:  common.MapStr{"custom": []string{"t1", "t2"}},
			cfg:         single(`{add_tags: {tags: [t1, t2], target: custom}}`),
		},
		"different targets": {
			eventFields: common.MapStr{},
			wantFields:  common.MapStr{"tags1": []string{"t1"}, "tags2": []string{"t2"}},
			cfg: multi(
				`{add_tags: {target: tags1, tags: [t1]}}`,
				`{add_tags: {target: tags2, tags: [t2]}}`,
			),
		},
		"single tag config without array notation": {
			eventFields: common.MapStr{},
			wantFields:  common.MapStr{"tags": []string{"t1"}},
			cfg:         single(`{add_tags: {tags: t1}}`),
		},
	})
}
