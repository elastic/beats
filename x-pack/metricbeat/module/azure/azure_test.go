package azure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGroupMetricsDefinitionsByResourceId(t *testing.T) {

	t.Run("Group metrics definitions by resource ID", func(t *testing.T) {
		metrics := []Metric{
			{
				ResourceId: "test",
				Namespace:  "test",
				Names:      []string{"name1"},
			},
			{
				ResourceId: "test",
				Namespace:  "test",
				Names:      []string{"name2"},
			},
			{
				ResourceId: "test",
				Namespace:  "test",
				Names:      []string{"name3"},
			},
		}

		metricsByResourceId := groupMetricsDefinitionsByResourceId(metrics)

		assert.Equal(t, 1, len(metricsByResourceId))
		assert.Equal(t, 3, len(metricsByResourceId["test"]))
	})
}
