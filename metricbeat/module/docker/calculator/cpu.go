package calculator

import (
	"github.com/elastic/beats/libbeat/common"
	"strconv"
)

type CPUCalculator interface {
	PerCpuUsage() common.MapStr
	TotalUsage() float64
	UsageInKernelmode() float64
	UsageInUsermode() float64
}

type CPUCalculatorImpl struct {
	Old CPUData
	New CPUData
}
type CPUData struct {
	PerCpuUsage       []uint64
	TotalUsage        uint64
	UsageInKernelmode uint64
	UsageInUsermode   uint64
}

func (c CPUCalculatorImpl) PerCpuUsage() common.MapStr {
	var output common.MapStr
	if cap(c.New.PerCpuUsage) == cap(c.Old.PerCpuUsage) {
		output = common.MapStr{}
		for index := range c.New.PerCpuUsage {
			output["cpu"+strconv.Itoa(index)] = c.calculateLoad(c.New.PerCpuUsage[index] - c.Old.PerCpuUsage[index])
		}
	}
	return output
}

func (c CPUCalculatorImpl) TotalUsage() float64 {
	return c.calculateLoad(c.New.TotalUsage - c.Old.TotalUsage)
}

func (c CPUCalculatorImpl) UsageInKernelmode() float64 {
	return c.calculateLoad(c.New.UsageInKernelmode - c.Old.UsageInKernelmode)
}

func (c CPUCalculatorImpl) UsageInUsermode() float64 {
	return c.calculateLoad(c.New.UsageInUsermode - c.Old.UsageInUsermode)
}

func (c CPUCalculatorImpl) calculateLoad(value uint64) float64 {
	// value is the count of CPU nanosecond in 1sec
	// TODO save the old stat timestamp and reuse here in case of docker read time changes...
	// 1s = 1000000000 ns
	// value / 1000000000
	return float64(value) / float64(1000000000)
}
