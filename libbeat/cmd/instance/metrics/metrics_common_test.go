package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/version"
)

func TestMonitoring(t *testing.T) {
	metrics := monitoring.Default.GetRegistry("beat")
	metricsSnapshot := monitoring.CollectFlatSnapshot(metrics, monitoring.Full, true)
	assert.Equal(t, version.GetDefaultVersion(), metricsSnapshot.Strings["info.version"])
}
