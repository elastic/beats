package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
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
