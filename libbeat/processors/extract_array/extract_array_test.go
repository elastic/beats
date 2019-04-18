package extract_array

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestExtractArrayProcessor_String(t *testing.T) {
	p, err := NewExtractArray(common.MustNewConfigFrom(common.MapStr{
		"field": "csv",
		"mappings": common.MapStr{
			"source.ip":         0,
			"network.transport": 2,
			"destination.ip":    99,
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "extract_array={field=csv, mappings=[{0 source.ip} {2 network.transport} {99 destination.ip}]}", p.String())
}

func TestExtractArrayProcessor_Run(t *testing.T) {
	tests := map[string]struct {
		config   common.MapStr
		input    beat.Event
		expected beat.Event
		fail     bool
		afterFn  func(e *beat.Event)
	}{
		"sample": {
			config: common.MapStr{
				"field": "array",
				"mappings": common.MapStr{
					"dest.one": 1,
					"dest.two": 2,
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{"zero", 1, common.MapStr{"two": 2}},
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"array":    []interface{}{"zero", 1, common.MapStr{"two": 2}},
					"dest.one": 1,
					"dest.two": common.MapStr{"two": 2},
				},
			},
		},

		"modified elements": {
			config: common.MapStr{
				"field": "array",
				"mappings": common.MapStr{
					"dest.one": 1,
					"dest.two": 2,
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{"zero", 1, common.MapStr{"two": 2}},
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"array":    []interface{}{"zero", 1, common.MapStr{"two": 2}},
					"dest.one": 1,
					"dest.two": common.MapStr{"two": 3},
				},
			},
			afterFn: func(e *beat.Event) {
				e.PutValue("dest.two.two", 3)
			},
		},

		"out of range mapping": {
			config: common.MapStr{
				"field": "array",
				"mappings": common.MapStr{
					"source.ip":      0,
					"destination.ip": 999,
				},
			},
			input: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{"127.0.0.1"},
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{"127.0.0.1"},
				},
			},
			fail: true,
		},

		"ignore errors": {
			config: common.MapStr{
				"field": "array",
				"mappings": common.MapStr{
					"a":   0,
					"b.c": 1,
				},
				"fail_on_error": false,
			},
			input: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{3.14, 9000.0},
					"b":     true,
				},
			},
			expected: beat.Event{
				Fields: common.MapStr{
					"array": []interface{}{3.14, 9000.0},
					"a":     3.14,
					"b":     true,
				},
			},
		},
	}
	for title, tt := range tests {
		t.Run(title, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(tt.config)
			processor, err := NewExtractArray(cfg)
			if err != nil {
				t.Fatal(err)
			}
			result, err := processor.Run(&tt.input)
			if tt.afterFn != nil {
				tt.afterFn(result)
			}
			if tt.fail {
				assert.Error(t, err)
				t.Log("got expected error", err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected.Fields.Flatten(), result.Fields.Flatten())
			assert.Equal(t, tt.expected.Timestamp, result.Timestamp)
			t.Log(result)
		})
	}
}
