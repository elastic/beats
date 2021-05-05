package metrics

import (
	"bufio"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

type cpu struct {
	User    uint64
	Nice    uint64
	Sys     uint64
	Idle    uint64
	Wait    uint64
	Irq     uint64
	SoftIrq uint64
	Stolen  uint64
}

type cpuMetrics struct {
	totals cpu
	list   []cpu
}

func (self cpuMetrics) Total() uint64 {
	return self.totals.User + self.totals.Nice + self.totals.Sys + self.totals.Idle +
		self.totals.Wait + self.totals.Irq + self.totals.SoftIrq + self.totals.Stolen
}

func (self cpuMetrics) FillTicks(event *common.MapStr) {

	event.Put("user.pct", self.totals.User)
	event.Put("system.pct", self.totals.Sys)
	event.Put("idle.pct", self.totals.Idle)
	event.Put("nice.ticks", self.totals.Nice)
	event.Put("irq.ticks", self.totals.Irq)
	event.Put("iowait.ticks", self.totals.Wait)
	event.Put("softirq.ticks", self.totals.SoftIrq)
	event.Put("steal.ticks", self.totals.Stolen)

}

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
	// IOWait time is excluded from the total as per #7627.
	idleTime := cpuMetricTimeDelta(prevCPU.totals.Idle, self.totals.Idle, timeDelta, numCPU) + cpuMetricTimeDelta(prevCPU.totals.Wait, self.totals.Wait, timeDelta, numCPU)
	totalPct := common.Round(float64(numCPU)-idleTime, common.DefaultDecimalPlacesCount)

	event.Put("total.pct", totalPct)
	event.Put("user.pct", cpuMetricTimeDelta(prevCPU.totals.User, self.totals.User, timeDelta, numCPU))
	event.Put("system.pct", cpuMetricTimeDelta(prevCPU.totals.Sys, self.totals.Sys, timeDelta, numCPU))
	event.Put("idle.pct", cpuMetricTimeDelta(prevCPU.totals.Idle, self.totals.Idle, timeDelta, numCPU))
	event.Put("nice.pct", cpuMetricTimeDelta(prevCPU.totals.Nice, self.totals.Nice, timeDelta, numCPU))
	event.Put("irq.pct", cpuMetricTimeDelta(prevCPU.totals.Irq, self.totals.Irq, timeDelta, numCPU))
	event.Put("softirq.pct", cpuMetricTimeDelta(prevCPU.totals.SoftIrq, self.totals.SoftIrq, timeDelta, numCPU))
	event.Put("iowait.pct", cpuMetricTimeDelta(prevCPU.totals.Wait, self.totals.Wait, timeDelta, numCPU))
	event.Put("steal.pct", cpuMetricTimeDelta(prevCPU.totals.Stolen, self.totals.Stolen, timeDelta, numCPU))

}

func (self cpuMetrics) FillNormalizedPercentages(event *common.MapStr, prev MetricMap) {

}

func scanStatFile(scanner *bufio.Scanner) (MetricMap, error) {
	cpuData := cpuMetrics{}
	for scanner.Scan() {
		text := scanner.Text()
		// Check to see if this is the global CPU line
		var err error
		if isCPUGlobalLine(text) {
			cpuData.totals, err = parseCPULine(text)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing global CPU line")
			}
		}
		if isCPULine(text) {
			perCPU, err := parseCPULine(text)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing CPU line")
			}
			cpuData.list = append(cpuData.list, perCPU)
		}
	}
	return cpuData, nil
}

func parseCPULine(line string) (cpu, error) {
	cpuData := cpu{}
	fields := strings.Fields(line)
	var errs multierror.Errors
	var err error
	cpuData.User, err = touint(fields[1])
	errs = append(errs, err)
	cpuData.Nice, err = touint(fields[2])
	errs = append(errs, err)
	cpuData.Sys, err = touint(fields[3])
	errs = append(errs, err)
	cpuData.Idle, err = touint(fields[4])
	errs = append(errs, err)
	cpuData.Wait, err = touint(fields[5])
	errs = append(errs, err)
	cpuData.Irq, err = touint(fields[6])
	errs = append(errs, err)
	cpuData.SoftIrq, err = touint(fields[7])
	errs = append(errs, err)
	cpuData.Stolen, err = touint(fields[8])
	errs = append(errs, err)

	return cpuData, errs.Err()
}
