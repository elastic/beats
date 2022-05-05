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

	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
				"docker": mapstr.M{
					"container": mapstr.M{
						"id":   "abc",
						"name": "foobar",
					},
				},
			},
			result: bus.Event{
				"container": mapstr.M{
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
				"docker": mapstr.M{
					"container": mapstr.M{
						"id":   "abc",
						"name": "foobar",
						"labels": getNestedAnnotations(mapstr.M{
							"do.not.include":          "true",
							"co.elastic.logs/disable": "true",
						}),
					},
				},
			},
			result: bus.Event{
				"container": mapstr.M{
					"id":   "abc",
					"name": "foobar",
					"labels": getNestedAnnotations(mapstr.M{
						"do.not.include":          "true",
						"co.elastic.logs/disable": "true",
					}),
				},
				"hints": mapstr.M{
					"logs": mapstr.M{
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

func getNestedAnnotations(in mapstr.M) mapstr.M {
	out := mapstr.M{}

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
		Docker: mapstr.M{
			"container": mapstr.M{
				"id":    "abc",
				"name":  "foobar",
				"image": "",
				"labels": mapstr.M{
					"do": mapstr.M{"not": mapstr.M{"include": "true"}},
					"co": mapstr.M{"elastic": mapstr.M{"logs/disable": "true"}},
				},
			},
		},
		Container: mapstr.M{
			"id":   "abc",
			"name": "foobar",
			"image": mapstr.M{
				"name": "",
			},
			"labels": mapstr.M{
				"do": mapstr.M{"not": mapstr.M{"include": "true"}},
				"co": mapstr.M{"elastic": mapstr.M{"logs/disable": "true"}},
			},
		},
		Metadata: mapstr.M{
			"container": mapstr.M{
				"id":   "abc",
				"name": "foobar",
				"image": mapstr.M{
					"name": "",
				},
			},
			"docker": mapstr.M{
				"container": mapstr.M{
					"labels": mapstr.M{
						"do": mapstr.M{"not": mapstr.M{"include": "true"}},
						"co": mapstr.M{"elastic": mapstr.M{"logs/disable": "true"}},
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
		Docker: mapstr.M{
			"container": mapstr.M{
				"id":    "abc",
				"name":  "foobar",
				"image": "",
				"labels": mapstr.M{
					"do": mapstr.M{"not": mapstr.M{"include": "true"}},
					"co": mapstr.M{"elastic": mapstr.M{"logs/disable": "true"}},
				},
			},
		},
		Container: mapstr.M{
			"id":   "abc",
			"name": "foobar",
			"image": mapstr.M{
				"name": "",
			},
			"labels": mapstr.M{
				"do": mapstr.M{"not": mapstr.M{"include": "true"}},
				"co": mapstr.M{"elastic": mapstr.M{"logs/disable": "true"}},
			},
		},
		Metadata: mapstr.M{
			"container": mapstr.M{
				"id":   "abc",
				"name": "foobar",
				"image": mapstr.M{
					"name": "",
				},
			},
			"docker": mapstr.M{
				"container": mapstr.M{
					"labels": mapstr.M{
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
