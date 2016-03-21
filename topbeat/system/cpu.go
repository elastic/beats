package system

import (
	"github.com/elastic/beats/libbeat/common"
	sigar "github.com/elastic/gosigar"
)

type CpuTimes struct {
	sigar.Cpu
	UserPercent   float64 `json:"user_p"`
	SystemPercent float64 `json:"system_p"`
}

func GetCpuTimes() (*CpuTimes, error) {

	cpu := sigar.Cpu{}
	err := cpu.Get()
	if err != nil {
		return nil, err
	}

	return &CpuTimes{Cpu: cpu}, nil

}

func GetCpuTimesList() ([]CpuTimes, error) {

	cpuList := sigar.CpuList{}
	err := cpuList.Get()
	if err != nil {
		return nil, err
	}

	cpuTimes := make([]CpuTimes, len(cpuList.List))

	for i, cpu := range cpuList.List {
		cpuTimes[i] = CpuTimes{Cpu: cpu}
	}

	return cpuTimes, nil
}

func GetCpuPercentage(last *CpuTimes, current *CpuTimes) *CpuTimes {

	if last != nil && current != nil {
		all_delta := current.Cpu.Total() - last.Cpu.Total()

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 4)
		}

		current.UserPercent = calculate(current.Cpu.User, last.Cpu.User)
		current.SystemPercent = calculate(current.Cpu.Sys, last.Cpu.Sys)
	}

	return current
}

func GetCpuPercentageList(last, current []CpuTimes) []CpuTimes {

	if last != nil && current != nil && len(last) == len(current) {

		calculate := func(field2 uint64, field1 uint64, all_delta uint64) float64 {

			perc := 0.0
			delta := field2 - field1
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 4)
		}

		for i := 0; i < len(last); i++ {
			all_delta := current[i].Cpu.Total() - last[i].Cpu.Total()
			current[i].UserPercent = calculate(current[i].Cpu.User, last[i].Cpu.User, all_delta)
			current[i].SystemPercent = calculate(current[i].Cpu.Sys, last[i].Cpu.Sys, all_delta)
		}

	}

	return current
}

func GetCpuStatEvent(cpuStat *CpuTimes) common.MapStr {
	return common.MapStr{
		"user":     cpuStat.User,
		"system":   cpuStat.Sys,
		"nice":     cpuStat.Nice,
		"idle":     cpuStat.Idle,
		"iowait":   cpuStat.Wait,
		"irq":      cpuStat.Irq,
		"softirq":  cpuStat.SoftIrq,
		"steal":    cpuStat.Stolen,
		"user_p":   cpuStat.UserPercent,
		"system_p": cpuStat.SystemPercent,
	}
}
