package util

import (
	"sort"
	"testing"

	"github.com/mitchellh/hashstructure"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestMergeEvents(t *testing.T) {
	tests := []struct {
		Test    string
		EventsA []common.MapStr
		EventsB []common.MapStr
		Filter  map[string]string
		Delete  []string
		Match   []string
		Result  []common.MapStr
	}{
		{
			Test: "Merge events",
			EventsA: []common.MapStr{
				common.MapStr{
					"original": "event",
					"foo":      "bar",
					"bar":      "foo",
				},
			},
			EventsB: []common.MapStr{
				common.MapStr{
					"foo":  "bar",
					"bar":  "foo",
					"with": "change",
				},
			},
			Match: []string{"foo", "bar"},
			Result: []common.MapStr{
				common.MapStr{
					"bar":      "foo",
					"foo":      "bar",
					"original": "event",
					"with":     "change",
				},
			},
		},
		{
			Test: "Merge events with delete",
			EventsA: []common.MapStr{
				common.MapStr{
					"original": "event",
					"foo":      "bar",
				},
			},
			EventsB: []common.MapStr{
				common.MapStr{
					"foo":      "bar",
					"bar":      "foo",
					"with":     "change",
					"todelete": "thisfield",
				},
			},
			Delete: []string{"todelete"},
			Match:  []string{"foo"},
			Result: []common.MapStr{
				common.MapStr{
					"bar":      "foo",
					"foo":      "bar",
					"original": "event",
					"with":     "change",
				},
			},
		},
		{
			Test: "Append events when there is no match",
			EventsA: []common.MapStr{
				common.MapStr{
					"original": "event",
					"foo":      "bar",
					"bar":      "foo",
				},
			},
			EventsB: []common.MapStr{
				common.MapStr{
					"foo":  "bar",
					"bar":  "bar",
					"with": "change",
				},
			},
			Match: []string{"foo", "bar"},
			Result: []common.MapStr{
				common.MapStr{
					"bar":      "foo",
					"foo":      "bar",
					"original": "event",
				},
				common.MapStr{
					"foo":  "bar",
					"bar":  "bar",
					"with": "change",
				},
			},
		},
		{
			Test: "Filter events in B",
			EventsA: []common.MapStr{
				common.MapStr{
					"original": "event",
					"foo":      "bar",
					"bar":      "foo",
				},
			},
			EventsB: []common.MapStr{
				common.MapStr{
					"foo":  "bar",
					"bar":  "foo",
					"with": "change",
				},
			},
			Filter: map[string]string{
				"with": "nomatch",
			},
			Match: []string{"foo", "bar"},
			Result: []common.MapStr{
				common.MapStr{
					"bar":      "foo",
					"foo":      "bar",
					"original": "event",
				},
			},
		},
		{
			Test: "Filter, delete, match and append working together",
			EventsA: []common.MapStr{
				common.MapStr{
					"first":         "event",
					"matchingfield": "one",
				},
				common.MapStr{
					"second":        "event",
					"matchingfield": "two",
					"foo":           "bar",
				},
				common.MapStr{
					"third": "event",
					"foo":   "bar",
				},
			},
			EventsB: []common.MapStr{
				common.MapStr{
					"matchingfield": "one",
					"with":          "change1",
				},
				common.MapStr{
					"matchingfield": "two",
					"with":          "change2",
				},
				common.MapStr{
					"notmatching": "event",
					"foo":         "bar",
					"bar":         "bar",
				},
			},
			Delete: []string{"bar"},
			Match:  []string{"matchingfield"},
			Result: []common.MapStr{
				common.MapStr{
					"second":        "event",
					"matchingfield": "two",
					"foo":           "bar",
					"with":          "change2",
				},
				common.MapStr{
					"first":         "event",
					"matchingfield": "one",
					"with":          "change1",
				},
				common.MapStr{
					"foo":         "bar",
					"notmatching": "event",
					"third":       "event",
				},
			},
		},
	}

	for _, test := range tests {
		result := MergeEvents(test.EventsA, test.EventsB, test.Filter, test.Delete, test.Match)
		sort.SliceStable(result, func(i, j int) bool {
			h1, _ := hashstructure.Hash(result[i], nil)
			h2, _ := hashstructure.Hash(result[j], nil)
			return h1 < h2
		})
		assert.Equal(t, test.Result, result, test.Test)
	}
}
