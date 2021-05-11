package metrics

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
)

// MetricMap is an interface that all OS-specific code must impliment in order to return metrics upstream to metricbeat
type MetricMap interface {
	// Total is the total CPU time.
	Total() uint64
	// FillTicks populates a given event with the `ticks` values
	// This value is calculated on windows, and comes directly from OS APIs on other platforms
	FillTicks(event *common.MapStr)
	// FillPercentages populates a given event with CPU percentages, which requires a previous event to calculate a time delta.
	FillPercentages(event *common.MapStr, prev MetricMap)
	// FillNormalizedPercentages populates a given event with CPU percentages, which requires a previous event to calculate a time delta. This number is averaged across the known CPUs
	FillNormalizedPercentages(event *common.MapStr, prev MetricMap)
	// CPUCount is the count of online CPUs known to the OS APIs.
	CPUCount() int
}

func cpuMetricTimeDelta(v0, v1, timeDelta uint64, numCPU int) float64 {
	cpuDelta := int64(v1 - v0)
	pct := float64(cpuDelta) / float64(timeDelta)
	return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
}

// Monitor is used to monitor the overall CPU usage of the system.
type Monitor struct {
	lastSample MetricMap
	hostfs     string
}

// Metrics stores the current and the last sample collected by a Beat.
type Metrics struct {
	previousSample MetricMap
	currentSample  MetricMap
}

func (m *Metrics) NormalizedPercentages(event *common.MapStr) {
	m.currentSample.FillNormalizedPercentages(event, m.previousSample)
}

func (m *Metrics) Percentages(event *common.MapStr) {
	m.currentSample.FillPercentages(event, m.previousSample)
}

func (m *Metrics) Ticks(event *common.MapStr) {
	m.currentSample.FillTicks(event)
}

func (m *Metrics) CPUCount() int {
	return m.currentSample.CPUCount()
}

// Sample collects a new sample of the CPU usage metrics.
func (m *Monitor) Sample() (*Metrics, error) {
	metric, err := Get(m.hostfs)
	if err != nil {
		return nil, errors.Wrap(err, "Error fetching CPU metrics")
	}

	oldLastSample := m.lastSample
	m.lastSample = metric

	return &Metrics{oldLastSample, metric}, nil
}
