package metrics

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/monitoring"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/stretchr/testify/assert"
)
 
func TestMonitoring(t *testing.T) {
	metrics := monitoring.Default.GetRegistry("beat")
	metricsSnapshot := monitoring.CollectFlatSnapshot(metrics, monitoring.Full, true)
	assert.Equal(t, version.GetDefaultVersion(), metricsSnapshot.Strings["info.version"])
}
