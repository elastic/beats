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

package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/docker"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		event  bus.Event
		result bus.Event
	}{
		// Empty events should return empty hints
		{
			event:  bus.Event{},
			result: bus.Event{},
		},
		// Docker meta must be present in the hints
		{
			event: bus.Event{
				"docker": common.MapStr{
					"container": common.MapStr{
						"id":   "abc",
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"container": common.MapStr{
					"id":   "abc",
					"name": "foobar",
				},
			},
		},
		// Docker labels are testing with the following scenarios
		// do.not.include must not be part of the hints
		// logs/disable should be present in hints.logs.disable=true
		{
			event: bus.Event{
				"docker": common.MapStr{
					"container": common.MapStr{
						"id":   "abc",
						"name": "foobar",
						"labels": getNestedAnnotations(common.MapStr{
							"do.not.include":          "true",
							"co.elastic.logs/disable": "true",
						}),
					},
				},
			},
			result: bus.Event{
				"container": common.MapStr{
					"id":   "abc",
					"name": "foobar",
					"labels": getNestedAnnotations(common.MapStr{
						"do.not.include":          "true",
						"co.elastic.logs/disable": "true",
					}),
				},
				"hints": common.MapStr{
					"logs": common.MapStr{
						"disable": "true",
					},
				},
			},
		},
	}

	cfg := defaultConfig()

	p := Provider{
		config: cfg,
	}
	for _, test := range tests {
		assert.Equal(t, p.generateHints(test.event), test.result)
	}
}

func getNestedAnnotations(in common.MapStr) common.MapStr {
	out := common.MapStr{}

	for k, v := range in {
		out.Put(k, v)
	}
	return out
}

func TestGenerateMetaDockerNoDedot(t *testing.T) {
	event := bus.Event{
		"container": &docker.Container{
			ID:   "abc",
			Name: "foobar",
			Labels: map[string]string{
				"do.not.include":          "true",
				"co.elastic.logs/disable": "true",
			},
		},
	}

	cfg := defaultConfig()
	cfg.Dedot = false
	p := Provider{
		config: cfg,
	}
	_, meta := p.generateMetaDocker(event)
	expectedMeta := &dockerMetadata{
		Docker: common.MapStr{
			"container": common.MapStr{
				"id":    "abc",
				"name":  "foobar",
				"image": "",
				"labels": common.MapStr{
					"do": common.MapStr{"not": common.MapStr{"include": "true"}},
					"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
				},
			},
		},
		Container: common.MapStr{
			"id":   "abc",
			"name": "foobar",
			"image": common.MapStr{
				"name": "",
			},
			"labels": common.MapStr{
				"do": common.MapStr{"not": common.MapStr{"include": "true"}},
				"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
			},
		},
		Metadata: common.MapStr{
			"container": common.MapStr{
				"id":   "abc",
				"name": "foobar",
				"image": common.MapStr{
					"name": "",
				},
			},
			"docker": common.MapStr{
				"container": common.MapStr{
					"labels": common.MapStr{
						"do": common.MapStr{"not": common.MapStr{"include": "true"}},
						"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
					},
				},
			},
		},
	}
	assert.Equal(t, expectedMeta.Docker, meta.Docker)
	assert.Equal(t, expectedMeta.Container, meta.Container)
	assert.Equal(t, expectedMeta.Metadata, meta.Metadata)
}

func TestGenerateMetaDockerWithDedot(t *testing.T) {
	event := bus.Event{
		"container": &docker.Container{
			ID:   "abc",
			Name: "foobar",
			Labels: map[string]string{
				"do.not.include":          "true",
				"co.elastic.logs/disable": "true",
			},
		},
	}

	cfg := defaultConfig()
	cfg.Dedot = true
	p := Provider{
		config: cfg,
	}
	_, meta := p.generateMetaDocker(event)
	expectedMeta := &dockerMetadata{
		Docker: common.MapStr{
			"container": common.MapStr{
				"id":    "abc",
				"name":  "foobar",
				"image": "",
				"labels": common.MapStr{
					"do": common.MapStr{"not": common.MapStr{"include": "true"}},
					"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
				},
			},
		},
		Container: common.MapStr{
			"id":   "abc",
			"name": "foobar",
			"image": common.MapStr{
				"name": "",
			},
			"labels": common.MapStr{
				"do": common.MapStr{"not": common.MapStr{"include": "true"}},
				"co": common.MapStr{"elastic": common.MapStr{"logs/disable": "true"}},
			},
		},
		Metadata: common.MapStr{
			"container": common.MapStr{
				"id":   "abc",
				"name": "foobar",
				"image": common.MapStr{
					"name": "",
				},
			},
			"docker": common.MapStr{
				"container": common.MapStr{
					"labels": common.MapStr{
						"do_not_include":          "true",
						"co_elastic_logs/disable": "true",
					},
				},
			},
		},
	}
	assert.Equal(t, expectedMeta.Docker, meta.Docker)
	assert.Equal(t, expectedMeta.Container, meta.Container)
	assert.Equal(t, expectedMeta.Metadata, meta.Metadata)
}
