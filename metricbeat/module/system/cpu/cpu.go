// +build darwin freebsd linux openbsd windows

package cpu

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/metric/system/cpu"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "cpu", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system CPU metrics.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	cpu    *cpu.Monitor
}

// New is a mb.MetricSetFactory that returns a cpu.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if config.CPUTicks != nil && *config.CPUTicks {
		config.Metrics = append(config.Metrics, "ticks")
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		cpu:           new(cpu.Monitor),
	}, nil
}

// Fetch fetches CPU metrics from the OS.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	sample, err := m.cpu.Sample()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch CPU times")
	}

	event := common.MapStr{"cores": cpu.NumCores}

	for _, metric := range m.config.Metrics {
		switch strings.ToLower(metric) {
		case percentages:
			pct := sample.Percentages()
			event.Put("user.pct", pct.User)
			event.Put("system.pct", pct.System)
			event.Put("idle.pct", pct.Idle)
			event.Put("iowait.pct", pct.IOWait)
			event.Put("irq.pct", pct.IRQ)
			event.Put("nice.pct", pct.Nice)
			event.Put("softirq.pct", pct.SoftIRQ)
			event.Put("steal.pct", pct.Steal)
			event.Put("total.pct", pct.Total)
		case normalizedPercentages:
			normalizedPct := sample.NormalizedPercentages()
			event.Put("user.norm.pct", normalizedPct.User)
			event.Put("system.norm.pct", normalizedPct.System)
			event.Put("idle.norm.pct", normalizedPct.Idle)
			event.Put("iowait.norm.pct", normalizedPct.IOWait)
			event.Put("irq.norm.pct", normalizedPct.IRQ)
			event.Put("nice.norm.pct", normalizedPct.Nice)
			event.Put("softirq.norm.pct", normalizedPct.SoftIRQ)
			event.Put("steal.norm.pct", normalizedPct.Steal)
			event.Put("total.norm.pct", normalizedPct.Total)
		case ticks:
			ticks := sample.Ticks()
			event.Put("user.ticks", ticks.User)
			event.Put("system.ticks", ticks.System)
			event.Put("idle.ticks", ticks.Idle)
			event.Put("iowait.ticks", ticks.IOWait)
			event.Put("irq.ticks", ticks.IRQ)
			event.Put("nice.ticks", ticks.Nice)
			event.Put("softirq.ticks", ticks.SoftIRQ)
			event.Put("steal.ticks", ticks.Steal)
		}
	}

	return event, nil
}
