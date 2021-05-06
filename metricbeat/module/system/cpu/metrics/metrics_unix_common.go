// +build freebsd linux

package metrics

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
)

// cpu manages the CPU metrics from /proc/stat
// FreeBSD and and linux only use parts of these,
// but the APIs are similar enough that this is defined here,
// and the code that actually returns metrics to users will be OS-specific
type cpu struct {
	User uint64
	Nice uint64
	Sys  uint64
	Idle uint64
	// Linux-only below
	Wait    uint64
	Irq     uint64
	SoftIrq uint64
	Stolen  uint64
}

type cpuMetrics struct {
	totals cpu
	list   []cpu
}

// Get returns a metrics object for CPU data
func Get(procfs string) (MetricMap, error) {
	if procfs == "" {
		procfs = "/proc"
	}
	path := filepath.Join(procfs, "stat")
	fd, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error opening file %s", path)
	}

	return scanStatFile(bufio.NewScanner(fd))

}

// FillPercentages returns percentage data based on usage between two periods of CPU data
func (self cpuMetrics) FillPercentages(event *common.MapStr, prev MetricMap, numCPU int) {
	// TODO: Make this an error later
	if prev == nil {
		return
	}
	prevCPU, _ := prev.(cpuMetrics)

	timeDelta := self.Total() - prevCPU.Total()
	if timeDelta <= 0 {
		return
	}
	fillCPUMetrics(event, self, prevCPU, numCPU, timeDelta, ".pct")
}

// FillPercentages returns percentage data based on usage between two periods of CPU data, based on the average per-CPU usage
func (self cpuMetrics) FillNormalizedPercentages(event *common.MapStr, prev MetricMap) {
	// TODO: Make this an error later
	if prev == nil {
		return
	}
	prevCPU, _ := prev.(cpuMetrics)
	// "normalized" in this sense means when we multiply/subtract by the CPU count, we're getting percentages that amount to the average usage per-cpu, as opposed to system-wide
	normCPU := 1

	timeDelta := self.Total() - prevCPU.Total()
	if timeDelta <= 0 {
		return
	}

	fillCPUMetrics(event, self, prevCPU, normCPU, timeDelta, ".norm.pct")
}

func statScanner(scanner *bufio.Scanner, lineReader func(string) (cpu, error)) (cpuMetrics, error) {
	cpuData := cpuMetrics{}
	var err error

	for scanner.Scan() {
		text := scanner.Text()
		// Check to see if this is the global CPU line
		if isCPUGlobalLine(text) {
			cpuData.totals, err = lineReader(text)
			if err != nil {
				return cpuMetrics{}, errors.Wrap(err, "error parsing global CPU line")
			}
		}
		if isCPULine(text) {
			perCPU, err := lineReader(text)
			if err != nil {
				return cpuMetrics{}, errors.Wrap(err, "error parsing CPU line")
			}
			cpuData.list = append(cpuData.list, perCPU)

		}
	}
	return cpuData, nil
}

func isCPUGlobalLine(line string) bool {
	if len(line) > 4 && line[0:4] == "cpu " {
		return true
	}
	return false
}

func isCPULine(line string) bool {
	if len(line) > 3 && line[0:3] == "cpu" && line[3] != ' ' {
		return true
	}
	return false
}

func touint(val string) (uint64, error) {
	return strconv.ParseUint(val, 10, 64)
}
