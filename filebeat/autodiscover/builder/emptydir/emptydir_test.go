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

package emptydir

import (
	"testing"

	"github.com/stretchr/testify/assert"

	_ "github.com/elastic/beats/filebeat/autodiscover/builder/hints"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

func TestEmptyDir(t *testing.T) {
	tests := []struct {
		event  bus.Event
		len    int
		result common.MapStr
	}{
		// Hints without host should return nothing
		{
			event: bus.Event{
				"hints": common.MapStr{
					"logs": common.MapStr{},
				},
			},
			len:    0,
			result: common.MapStr{},
		},
		// Hints with emptydir log path for kubernetes should return appropriate config
		{
			event: bus.Event{
				"host": "1.2.3.4",
				"kubernetes": common.MapStr{
					"pod": common.MapStr{
						"name": "podname",
						"uid":  "pid",
					},
					"container": common.MapStr{
						"name": "containername",
						"id":   "abc",
					},
					"namespace": "foo",
				},
				"container": common.MapStr{
					"name": "containername",
					"id":   "abc",
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"emptydir": common.MapStr{
							"testdir": common.MapStr{
								"1": common.MapStr{
									"name":      "foo.log",
									"namespace": "test",
									"paths":     "/var/log/foo*log",
								},
							},
						},
					},
				},
			},
			len: 1,
			result: common.MapStr{
				"type":  "log",
				"paths": []interface{}{"/var/lib/kubelet/pods/pid/volumes/kubernetes.io~empty-dir/testdir/var/log/foo*log"},
			},
		},
	}

	for _, test := range tests {
		cfg := defaultConfig()
		c, err := common.NewConfigFrom(&cfg)
		assert.Nil(t, err)
		l, err := NewEmptyDirBuilder(c)
		assert.Nil(t, err)
		cfgs := l.CreateConfig(test.event)
		assert.Equal(t, len(cfgs), test.len)

		if len(cfgs) != 0 {
			config := common.MapStr{}
			err := cfgs[0].Unpack(&config)
			assert.Nil(t, err)
			assert.Equal(t, test.result, config)
		}

	}
}
