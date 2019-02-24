package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestNewMetricSet(t *testing.T) {
	t.Run("pass in host", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"module":     "redis",
			"metricsets": []string{"info"},
			"hosts": []string{
				"redis://me:secret@localhost:123",
			},
		})
		if err != nil {
			t.Fatal(err)
		}

		ms := mbtest.NewReportingMetricSetV2(t, c)
		assert.Equal(t, "secret", ms.HostData().Password)
	})

	t.Run("password in config", func(t *testing.T) {
		c, err := common.NewConfigFrom(map[string]interface{}{
			"module":     "redis",
			"metricsets": []string{"info"},
			"hosts": []string{
				"redis://localhost:123",
			},
			"password": "secret",
		})
		if err != nil {
			t.Fatal(err)
		}

		ms := mbtest.NewReportingMetricSetV2(t, c)
		assert.Equal(t, "secret", ms.HostData().Password)
	})
}
