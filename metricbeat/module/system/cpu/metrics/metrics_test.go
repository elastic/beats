package metrics

import (
	"runtime"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestCPUGet(t *testing.T) {
	root := ""
	if runtime.GOOS == "freebsd" {
		root = "/compat/linux/proc/"
	}
	metrics, err := Get(root)

	assert.NoError(t, err, "error in Get")
	assert.NotZero(t, metrics.Total(), "got total zero")

	time.Sleep(time.Second * 3)

	secondMetrics, err := Get(root)
	assert.NoError(t, err, "error in Get")
	assert.NotZero(t, metrics.Total(), "got total zero")

	events := common.MapStr{}
	secondMetrics.FillPercentages(&events, metrics)

	total, err := events.GetValue("total.pct")
	assert.NoError(t, err, "error finding total.pct")
	assert.NotZero(t, total.(float64), "total is zero")

	secondMetrics.FillNormalizedPercentages(&events, metrics)

	totalNorm, err := events.GetValue("total.norm.pct")
	assert.NoError(t, err, "error finding total.pct")
	assert.NotZero(t, totalNorm.(float64), "total is zero")

	secondMetrics.FillTicks(&events)

	t.Logf("Got metrics: \n%s", events.StringToPrint())
}
