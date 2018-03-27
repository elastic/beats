package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

func TestGenerateHints(t *testing.T) {
	tests := []struct {
		annotations map[string]string
		result      common.MapStr
	}{
		// Empty annotations should return empty hints
		{
			annotations: map[string]string{},
			result:      common.MapStr{},
		},

		// Scenarios being tested:
		// logs/multiline.pattern must be a nested common.MapStr under hints.logs
		// metrics/module must be found in hints.metrics
		// not.to.include must not be part of hints
		// period is annotated at both container and pod level. Container level value must be in hints
		{
			annotations: map[string]string{
				"co.elastic.logs/multiline.pattern": "^test",
				"co.elastic.metrics/module":         "prometheus",
				"co.elastic.metrics/period":         "10s",
				"co.elastic.metrics.foobar/period":  "15s",
				"co.elastic.metrics.foobar1/period": "15s",
				"not.to.include":                    "true",
			},
			result: common.MapStr{
				"logs": common.MapStr{
					"multiline": common.MapStr{
						"pattern": "^test",
					},
				},
				"metrics": common.MapStr{
					"module": "prometheus",
					"period": "15s",
				},
			},
		},
	}

	for _, test := range tests {
		annMap := common.MapStr{}
		for k, v := range test.annotations {
			annMap.Put(k, v)
		}
		assert.Equal(t, GenerateHints(annMap, "foobar", "co.elastic"), test.result)
	}
}
