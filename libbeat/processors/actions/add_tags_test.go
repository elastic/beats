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

func TestAddTags(t *testing.T) {
	multi := func(strs ...string) []string { return strs }
	single := func(str string) []string { return multi(str) }

	cases := map[string]struct {
		event common.MapStr
		want  common.MapStr
		cfg   []string
	}{
		"create tags": {
			event: common.MapStr{},
			want:  common.MapStr{"tags": []string{"t1", "t2"}},
			cfg:   single(`{tags: [t1, t2]}`),
		},
		"append to tags": {
			event: common.MapStr{"tags": []string{"t1"}},
			want:  common.MapStr{"tags": []string{"t1", "t2", "t3"}},
			cfg:   single(`{tags: [t2, t3]}`),
		},
		"combine from 2 processors": {
			event: common.MapStr{},
			want:  common.MapStr{"tags": []string{"t1", "t2", "t3", "t4"}},
			cfg: multi(
				`{tags: [t1, t2]}`,
				`{tags: [t3, t4]}`,
			),
		},
		"with custom target": {
			event: common.MapStr{},
			want:  common.MapStr{"custom": []string{"t1", "t2"}},
			cfg:   single(`{tags: [t1, t2], target: custom}`),
		},
		"different targets": {
			event: common.MapStr{},
			want:  common.MapStr{"tags1": []string{"t1"}, "tags2": []string{"t2"}},
			cfg: multi(
				`{target: tags1, tags: [t1]}`,
				`{target: tags2, tags: [t2]}`,
			),
		},
		"single tag config without array notation": {
			event: common.MapStr{},
			want:  common.MapStr{"tags": []string{"t1"}},
			cfg:   single(`{tags: t1}`),
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
						"add_tags": config,
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
