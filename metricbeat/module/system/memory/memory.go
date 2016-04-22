package memory

import (
	"github.com/elastic/beats/metricbeat/helper"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	_ "github.com/elastic/beats/metricbeat/module/system"
	"github.com/elastic/beats/topbeat/system"
)

func init() {
	helper.Registry.AddMetricSeter("system", "memory", New)
}

// New creates new instance of MetricSeter
func New() helper.MetricSeter {
	return &MetricSeter{}
}

type MetricSeter struct{}

func (m *MetricSeter) Setup(ms *helper.MetricSet) error {
	return nil
}

func (m *MetricSeter) Fetch(ms *helper.MetricSet, host string) (event common.MapStr, err error) {

	memStat, err := system.GetMemory()
	if err != nil {
		logp.Warn("Getting memory details: %v", err)
		return nil, err
	}

	swapStat, err := system.GetSwap()
	if err != nil {
		logp.Warn("Getting swap details: %v", err)
		return nil, err
	}

	event = common.MapStr{
		"mem":  system.GetMemoryEvent(memStat),
		"swap": system.GetSwapEvent(swapStat),
	}

	return event, nil
}
