// +build darwin freebsd linux openbsd windows

package system

import (
	"math"
	"runtime"

	sigar "github.com/elastic/gosigar"
)

// maxDecimalPlaces is the maximum number of decimal places that the Round
// function return.
const maxDecimalPlaces = 4

var (
	// NumCPU is the number of CPU cores in the system. Changes to operating
	// system CPU allocation after process startup are not reflected.
	NumCPU = runtime.NumCPU()
)

// CPU Monitor

// CPUMonitor is used to monitor the overal CPU usage of the system.
type CPUMonitor struct {
	lastSample *sigar.Cpu
}

// Sample collects a new sample of the CPU usage metrics.
func (m *CPUMonitor) Sample() (*CPUMetrics, error) {
	cpuSample := &sigar.Cpu{}
	if err := cpuSample.Get(); err != nil {
		return nil, err
	}

	oldLastSample := m.lastSample
	m.lastSample = cpuSample
	return &CPUMetrics{oldLastSample, cpuSample}, nil
}

type CPUPercentages struct {
	User    float64
	System  float64
	Idle    float64
	IOWait  float64
	IRQ     float64
	Nice    float64
	SoftIRQ float64
	Steal   float64
}

type CPUTicks struct {
	User    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	Nice    uint64
	SoftIRQ uint64
	Steal   uint64
}

type CPUMetrics struct {
	previousSample *sigar.Cpu
	currentSample  *sigar.Cpu
}

// NormalizedPercentages returns CPU percentage usage information that is
// normalized by the number of CPU cores (NumCPU). The values will range from
// 0 to 100%.
func (m *CPUMetrics) NormalizedPercentages() CPUPercentages {
	return cpuPercentages(m.previousSample, m.currentSample, 1)
}

// Percentages returns CPU percentage usage information. The values range from
// 0 to 100% * NumCPU.
func (m *CPUMetrics) Percentages() CPUPercentages {
	return cpuPercentages(m.previousSample, m.currentSample, NumCPU)
}

// cpuPercentages calculates the amount of CPU time used between the two given
// samples. The CPU percentages are divided by given numCPU value and rounded
// using Round.
func cpuPercentages(s0, s1 *sigar.Cpu, numCPU int) CPUPercentages {
	if s0 == nil || s1 == nil {
		return CPUPercentages{}
	}

	// timeDelta is the total amount of CPU time available across all CPU cores.
	timeDelta := s1.Total() - s0.Total()
	if timeDelta <= 0 {
		return CPUPercentages{}
	}

	calculatePct := func(v0, v1 uint64) float64 {
		cpuDelta := int64(v1 - v0)
		pct := float64(cpuDelta) / float64(timeDelta)
		return Round(pct * float64(numCPU))
	}

	return CPUPercentages{
		User:    calculatePct(s0.User, s1.User),
		System:  calculatePct(s0.Sys, s1.Sys),
		Idle:    calculatePct(s0.Idle, s1.Idle),
		IOWait:  calculatePct(s0.Wait, s1.Wait),
		IRQ:     calculatePct(s0.Irq, s1.Irq),
		Nice:    calculatePct(s0.Nice, s1.Nice),
		SoftIRQ: calculatePct(s0.SoftIrq, s1.SoftIrq),
		Steal:   calculatePct(s0.Stolen, s1.Stolen),
	}
}

func (m *CPUMetrics) Ticks() CPUTicks {
	return CPUTicks{
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

// CPUCoreMonitor is used to monitor the usage of individual CPU cores.
type CPUCoreMetrics CPUMetrics

// Percentages returns CPU percentage usage information for the core. The values
// range from [0, 100%].
func (m *CPUCoreMetrics) Percentages() CPUPercentages { return (*CPUMetrics)(m).NormalizedPercentages() }

// Ticks returns the raw number of "ticks". The value is a counter (though it
// may roll over).
func (m *CPUCoreMetrics) Ticks() CPUTicks { return (*CPUMetrics)(m).Ticks() }

// CPUCoresMonitor is used to monitor the usage information of all the CPU
// cores in the system.
type CPUCoresMonitor struct {
	lastSample []sigar.Cpu
}

// Sample collects a new sample of the metrics from all CPU cores.
func (m *CPUCoresMonitor) Sample() ([]CPUCoreMetrics, error) {
	var cores sigar.CpuList
	if err := cores.Get(); err != nil {
		return nil, err
	}

	lastSample := m.lastSample
	m.lastSample = cores.List

	cpuMetrics := make([]CPUCoreMetrics, len(cores.List))
	for i := 0; i < len(cores.List); i++ {
		if len(lastSample) > i {
			cpuMetrics[i] = CPUCoreMetrics{&lastSample[i], &cores.List[i]}
		} else {
			cpuMetrics[i] = CPUCoreMetrics{nil, &cores.List[i]}
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

type LoadMetrics struct {
	sample *sigar.LoadAverage
}

type LoadAverages struct {
	OneMinute     float64
	FiveMinute    float64
	FifteenMinute float64
}

// Averages return the CPU load averages. These values should range from
// 0 to NumCPU.
func (m *LoadMetrics) Averages() LoadAverages {
	return LoadAverages{
		OneMinute:     Round(m.sample.One),
		FiveMinute:    Round(m.sample.Five),
		FifteenMinute: Round(m.sample.Fifteen),
	}
}

// NormalizedAverages return the CPU load averages normalized by the NumCPU.
// These values should range from 0 to 1.
func (m *LoadMetrics) NormalizedAverages() LoadAverages {
	return LoadAverages{
		OneMinute:     Round(m.sample.One / float64(NumCPU)),
		FiveMinute:    Round(m.sample.Five / float64(NumCPU)),
		FifteenMinute: Round(m.sample.Fifteen / float64(NumCPU)),
	}
}

// Helpers

// Round rounds the given float64 value and ensures that it has a maximum of
// four decimal places.
func Round(val float64) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(maxDecimalPlaces))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= 0.5 {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}
