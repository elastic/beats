package collector

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestSplitTagsFromMetricName(t *testing.T) {
	for _, testCase := range []struct {
		title string
		name  string
		key   string
		tags  common.MapStr
	}{
		{
			title: "no tags",
			name:  "my_metric1",
		}, {
			title: "parameter",
			name:  "metric/{something}/other",
		}, {
			title: "trailing parameter",
			name:  "metric/{notakey}",
		}, {
			title: "standard tags",
			name:  "metric{key1=var1, key2=var2}",
			key:   "metric",
			tags:  common.MapStr{"key1": "var1", "key2": "var2"},
		}, {
			title: "empty parameter",
			name:  "metric/{}",
		}, {
			title: "empty key or value",
			name:  "metric{=var1, key2=}",
			key:   "metric",
			tags:  common.MapStr{"": "var1", "key2": ""},
		}, {
			title: "empty key and value",
			name:  "metric{=}",
			key:   "metric",
			tags:  common.MapStr{"": ""},
		}, {
			title: "extra comma",
			name:  "metric{a=b,}",
			key:   "metric",
			tags:  common.MapStr{"a": "b"},
		}, {
			title: "extra comma and space",
			name:  "metric{a=b, }",
			key:   "metric",
			tags:  common.MapStr{"a": "b"},
		},
	} {
		t.Run(testCase.title, func(t *testing.T) {
			key, tags := splitTagsFromMetricName(testCase.name)
			if testCase.key == "" && tags == nil {
				assert.Equal(t, testCase.name, key)
			} else {
				assert.Equal(t, testCase.key, key)
				assert.Equal(t, testCase.tags, tags)
			}
		})
	}
}
