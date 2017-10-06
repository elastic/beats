package dashboards

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestReplaceStringInDashboard(t *testing.T) {
	tests := []struct {
		content  common.MapStr
		old      string
		new      string
		expected common.MapStr
	}{
		{
			content:  common.MapStr{"test": "CHANGEME"},
			old:      "CHANGEME",
			new:      "hostname",
			expected: common.MapStr{"test": "hostname"},
		},
		{
			content:  common.MapStr{"test": "hello"},
			old:      "CHANGEME",
			new:      "hostname",
			expected: common.MapStr{"test": "hello"},
		},
		{
			content:  common.MapStr{"test": map[string]interface{}{"key": "\"CHANGEME\""}},
			old:      "CHANGEME",
			new:      "hostname.local",
			expected: common.MapStr{"test": map[string]interface{}{"key": "\"hostname.local\""}},
		},
		{
			content: common.MapStr{
				"kibanaSavedObjectMeta": map[string]interface{}{
					"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"CHANGEME_HOSTNAME\\\"\",\"language\":\"lucene\"}}"}},

			old: "CHANGEME_HOSTNAME",
			new: "hostname.local",
			expected: common.MapStr{
				"kibanaSavedObjectMeta": map[string]interface{}{
					"searchSourceJSON": "{\"filter\":[],\"highlightAll\":true,\"version\":true,\"query\":{\"query\":\"beat.name:\\\"hostname.local\\\"\",\"language\":\"lucene\"}}"}},
		},
	}

	for _, test := range tests {
		result, err := ReplaceStringInDashboard(test.old, test.new, test.content)
		assert.NoError(t, err)
		assert.Equal(t, test.expected, result)
	}
}
