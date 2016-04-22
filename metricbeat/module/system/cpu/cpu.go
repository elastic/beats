package cpu

import (
	"github.com/elastic/beats/metricbeat/helper"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/topbeat/system"
)

func init() {
	helper.Registry.AddMetricSeter("system", "cpu", New)
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

	cpuStat, err := system.GetCpuTimes()
	if err != nil {
		logp.Warn("Getting cpu times: %v", err)
		return nil, err
	}

	event = system.GetCpuStatEvent(cpuStat)

	return event, nil
}

func (m *MetricSeter) Cleanup() error {
	return nil
}
