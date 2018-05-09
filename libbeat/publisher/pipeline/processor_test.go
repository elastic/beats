package pipeline

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

func TestProcessors(t *testing.T) {
	info := beat.Info{}

	type local struct {
		config   beat.ClientConfig
		events   []common.MapStr
		expected []common.MapStr
	}

	tests := []struct {
		name   string
		global pipelineProcessors
		local  []local
	}{
		{
			"user global fields and tags",
			pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			[]local{
				{
					config: beat.ClientConfig{},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{"value": "abc", "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"no normalization",
			pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			[]local{
				{
					config: beat.ClientConfig{SkipNormalization: true},
					events: []common.MapStr{{"value": "abc", "user": nil}},
					expected: []common.MapStr{
						{"value": "abc", "user": nil, "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"beat local fields",
			pipelineProcessors{},
			[]local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "local": 1}},
				},
			},
		},
		{
			"beat local and user global fields",
			pipelineProcessors{
				fields: common.MapStr{"global": 1},
				tags:   []string{"tag"},
			},
			[]local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "global": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"user global fields overwrite beat local fields",
			pipelineProcessors{
				fields: common.MapStr{"global": 1, "shared": "global"},
				tags:   []string{"tag"},
			},
			[]local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1, "shared": "local"},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "global": 1, "shared": "global", "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"beat local fields isolated",
			pipelineProcessors{},
			[]local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "local": 1}},
				},
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 2},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "local": 2}},
				},
			},
		},

		{
			"beat local fields + user global fields isolated",
			pipelineProcessors{
				fields: common.MapStr{"global": 0},
			},
			[]local{
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 1},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "global": 0, "local": 1}},
				},
				{
					config: beat.ClientConfig{
						Fields: common.MapStr{"local": 2},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "global": 0, "local": 2}},
				},
			},
		},
		{
			"user local fields and tags",
			pipelineProcessors{},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
							Tags:   []string{"tag"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "fields": common.MapStr{"local": 1}, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"user local fields (under root) and tags",
			pipelineProcessors{},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1},
							FieldsUnderRoot: true,
							Tags:            []string{"tag"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{"value": "abc", "local": 1, "tags": []string{"tag"}},
					},
				},
			},
		},
		{
			"user local fields overwrite user global fields",
			pipelineProcessors{
				fields: common.MapStr{"global": 0, "shared": "global"},
				tags:   []string{"global"},
			},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1, "shared": "local"},
							FieldsUnderRoot: true,
							Tags:            []string{"local"},
						},
					},
					events: []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{
						{
							"value":  "abc",
							"global": 0, "local": 1, "shared": "local",
							"tags": []string{"global", "local"},
						},
					},
				},
			},
		},
		{
			"user local fields isolated",
			pipelineProcessors{},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "fields": common.MapStr{"local": 1}}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 2},
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "fields": common.MapStr{"local": 2}}},
				},
			},
		},
		{
			"user local + global fields isolated",
			pipelineProcessors{
				fields: common.MapStr{"fields": common.MapStr{"global": 0}},
			},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 1},
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "fields": common.MapStr{"global": 0, "local": 1}}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields: common.MapStr{"local": 2},
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "fields": common.MapStr{"global": 0, "local": 2}}},
				},
			},
		},
		{
			"user local + global fields isolated (fields with root)",
			pipelineProcessors{
				fields: common.MapStr{"global": 0},
			},
			[]local{
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 1},
							FieldsUnderRoot: true,
						},
					},
					events:   []common.MapStr{{"value": "abc"}},
					expected: []common.MapStr{{"value": "abc", "global": 0, "local": 1}},
				},
				{
					config: beat.ClientConfig{
						EventMetadata: common.EventMetadata{
							Fields:          common.MapStr{"local": 2},
							FieldsUnderRoot: true,
						},
					},
					events:   []common.MapStr{{"value": "def"}},
					expected: []common.MapStr{{"value": "def", "global": 0, "local": 2}},
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			// create processor pipelines
			programs := make([]beat.Processor, len(test.local))
			for i, local := range test.local {
				programs[i] = newProcessorPipeline(info, test.global, local.config)
			}

			// run processor pipelines in parallel
			var (
				wg      sync.WaitGroup
				mux     sync.Mutex
				results = make([][]common.MapStr, len(programs))
			)
			for id, local := range test.local {
				wg.Add(1)
				id, program, local := id, programs[id], local
				go func() {
					defer wg.Done()

					actual := make([]common.MapStr, len(local.events))
					for i, event := range local.events {
						out, _ := program.Run(&beat.Event{
							Timestamp: time.Now(),
							Fields:    event,
						})
						actual[i] = out.Fields
					}

					mux.Lock()
					defer mux.Unlock()
					results[id] = actual
				}()
			}
			wg.Wait()

			// validate
			for i, local := range test.local {
				assert.Equal(t, local.expected, results[i])
			}
		})
	}
}
