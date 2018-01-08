// +build darwin freebsd linux openbsd windows

package cpu

import (
	"runtime"

	"github.com/elastic/beats/libbeat/common"
	sigar "github.com/elastic/gosigar"
)

var (
	// NumCores is the number of CPU cores in the system. Changes to operating
	// system CPU allocation after process startup are not reflected.
	NumCores = runtime.NumCPU()
)

// CPU Monitor

// Monitor is used to monitor the overal CPU usage of the system.
type Monitor struct {
	lastSample *sigar.Cpu
}

// Sample collects a new sample of the CPU usage metrics.
func (m *Monitor) Sample() (*Metrics, error) {
	cpuSample := &sigar.Cpu{}
	if err := cpuSample.Get(); err != nil {
		return nil, err
	}

	oldLastSample := m.lastSample
	m.lastSample = cpuSample
	return &Metrics{oldLastSample, cpuSample}, nil
}

// Percentages stores all CPU values in percentages collected by a Beat.
type Percentages struct {
	User    float64
	System  float64
	Idle    float64
	IOWait  float64
	IRQ     float64
	Nice    float64
	SoftIRQ float64
	Steal   float64
	Total   float64
}

// Ticks stores all CPU values in number of tick collected by a Beat.
type Ticks struct {
	User    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	Nice    uint64
	SoftIRQ uint64
	Steal   uint64
}

// Metrics stores the current and the last sample collected by a Beat.
type Metrics struct {
	previousSample *sigar.Cpu
	currentSample  *sigar.Cpu
}

// NormalizedPercentages returns CPU percentage usage information that is
// normalized by the number of CPU cores (NumCores). The values will range from
// 0 to 100%.
func (m *Metrics) NormalizedPercentages() Percentages {
	return cpuPercentages(m.previousSample, m.currentSample, 1)
}

// Percentages returns CPU percentage usage information. The values range from
// 0 to 100% * NumCores.
func (m *Metrics) Percentages() Percentages {
	return cpuPercentages(m.previousSample, m.currentSample, NumCores)
}

// cpuPercentages calculates the amount of CPU time used between the two given
// samples. The CPU percentages are divided by given numCPU value and rounded
// using Round.
func cpuPercentages(s0, s1 *sigar.Cpu, numCPU int) Percentages {
	if s0 == nil || s1 == nil {
		return Percentages{}
	}

	// timeDelta is the total amount of CPU time available across all CPU cores.
	timeDelta := s1.Total() - s0.Total()
	if timeDelta <= 0 {
		return Percentages{}
	}

	calculatePct := func(v0, v1 uint64) float64 {
		cpuDelta := int64(v1 - v0)
		pct := float64(cpuDelta) / float64(timeDelta)
		return common.Round(pct*float64(numCPU), common.DefaultDecimalPlacesCount)
	}

	calculateTotalPct := func() float64 {
		return common.Round(float64(numCPU)-calculatePct(s0.Idle, s1.Idle), common.DefaultDecimalPlacesCount)
	}

	return Percentages{
		User:    calculatePct(s0.User, s1.User),
		System:  calculatePct(s0.Sys, s1.Sys),
		Idle:    calculatePct(s0.Idle, s1.Idle),
		IOWait:  calculatePct(s0.Wait, s1.Wait),
		IRQ:     calculatePct(s0.Irq, s1.Irq),
		Nice:    calculatePct(s0.Nice, s1.Nice),
		SoftIRQ: calculatePct(s0.SoftIrq, s1.SoftIrq),
		Steal:   calculatePct(s0.Stolen, s1.Stolen),
		Total:   calculateTotalPct(),
	}
}

// Ticks returns the number of CPU ticks from the last collected sample.
func (m *Metrics) Ticks() Ticks {
	return Ticks{
		User:    m.currentSample.User,
		System:  m.currentSample.Sys,
		Idle:    m.currentSample.Idle,
		IOWait:  m.currentSample.Wait,
		IRQ:     m.currentSample.Irq,
		Nice:    m.currentSample.Nice,
		SoftIRQ: m.currentSample.SoftIrq,
		Steal:   m.currentSample.Stolen,
	}
}

// CPU Core Monitor

// CoreMetrics is used to monitor the usage of individual CPU cores.
type CoreMetrics Metrics

// Percentages returns CPU percentage usage information for the core. The values
// range from [0, 100%].
func (m *CoreMetrics) Percentages() Percentages { return (*Metrics)(m).NormalizedPercentages() }

// Ticks returns the raw number of "ticks". The value is a counter (though it
// may roll overfunc (m *CoreMetrics) Ticks() Ticks { return (*Metrics)(m).Ticks() }
func (m *CoreMetrics) Ticks() Ticks { return (*Metrics)(m).Ticks() }

// CoresMonitor is used to monitor the usage information of all the CPU
// cores in the system.
type CoresMonitor struct {
	lastSample []sigar.Cpu
}

// Sample collects a new sample of the metrics from all CPU cores.
func (m *CoresMonitor) Sample() ([]CoreMetrics, error) {
	var cores sigar.CpuList
	if err := cores.Get(); err != nil {
		return nil, err
	}

	lastSample := m.lastSample
	m.lastSample = cores.List

	cpuMetrics := make([]CoreMetrics, len(cores.List))
	for i := 0; i < len(cores.List); i++ {
		if len(lastSample) > i {
			cpuMetrics[i] = CoreMetrics{&lastSample[i], &cores.List[i]}
		} else {
			cpuMetrics[i] = CoreMetrics{nil, &cores.List[i]}
		}
	}

	return cpuMetrics, nil
}

// CPU Load

// Load returns CPU load information for the previous 1, 5, and 15 minute
// periods.
func Load() (*LoadMetrics, error) {
	load := &sigar.LoadAverage{}
	if err := load.Get(); err != nil {
		return nil, err
	}

	return &LoadMetrics{load}, nil
}

// LoadMetrics stores the sampled load average values of the host.
type LoadMetrics struct {
	sample *sigar.LoadAverage
}

// LoadAverages stores the values of load averages of the last 1, 5 and 15 minutes.
type LoadAverages struct {
	OneMinute     float64
	FiveMinute    float64
	FifteenMinute float64
}

// Averages return the CPU load averages. These values should range from
// 0 to NumCores.
func (m *LoadMetrics) Averages() LoadAverages {
	return LoadAverages{
		OneMinute:     common.Round(m.sample.One, common.DefaultDecimalPlacesCount),
		FiveMinute:    common.Round(m.sample.Five, common.DefaultDecimalPlacesCount),
		FifteenMinute: common.Round(m.sample.Fifteen, common.DefaultDecimalPlacesCount),
	}
}

// NormalizedAverages return the CPU load averages normalized by the NumCores.
// These values should range from 0 to 1.
func (m *LoadMetrics) NormalizedAverages() LoadAverages {
	return LoadAverages{
		OneMinute:     common.Round(m.sample.One/float64(NumCores), common.DefaultDecimalPlacesCount),
		FiveMinute:    common.Round(m.sample.Five/float64(NumCores), common.DefaultDecimalPlacesCount),
		FifteenMinute: common.Round(m.sample.Fifteen/float64(NumCores), common.DefaultDecimalPlacesCount),
	}
}
