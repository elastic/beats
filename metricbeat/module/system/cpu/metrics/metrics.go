package metrics

import "github.com/elastic/beats/v7/libbeat/common"

// MetricMap is an interface that all OS-specific code must impliment in order to return metrics upstream to metricbeat
type MetricMap interface {
	Total() uint64
	FillTicks(event *common.MapStr)
	FillPercentages(event *common.MapStr, prev MetricMap, numCPU int)
	FillNormalizedPercentages(event *common.MapStr, prev MetricMap)
}

func cpuMetricTimeDelta(v0, v1, timeDelta uint64, numCPU int) float64 {
	cpuDelta := int64(v1 - v0)
	pct := float64(cpuDelta) / float64(timeDelta)
	return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
}
