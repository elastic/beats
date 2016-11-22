// +build darwin freebsd linux openbsd windows

package cpu

import (
	"github.com/elastic/beats/metricbeat/module/system"
	sigar "github.com/elastic/gosigar"
)

type CPU struct {
	CpuPerCore       bool
	LastCpuTimes     *CpuTimes
	LastCpuTimesList []CpuTimes
	CpuTicks         bool
}

type CpuTimes struct {
	sigar.Cpu
	UserPercent    float64 `json:"user_p"`
	SystemPercent  float64 `json:"system_p"`
	IdlePercent    float64 `json:"idle_p"`
	IOwaitPercent  float64 `json:"iowait_p"`
	IrqPercent     float64 `json:"irq_p"`
	NicePercent    float64 `json:"nice_p"`
	SoftIrqPercent float64 `json:"softirq_p"`
	StealPercent   float64 `json:"steal_p"`
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
		allDelta := current.Cpu.Total() - last.Cpu.Total()

		if allDelta == 0 {
			// first inquiry
			return current
		}

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(allDelta)
			return system.Round(perc, .5, 4)
		}

		current.UserPercent = calculate(current.Cpu.User, last.Cpu.User)
		current.SystemPercent = calculate(current.Cpu.Sys, last.Cpu.Sys)
		current.IdlePercent = calculate(current.Cpu.Idle, last.Cpu.Idle)
		current.IOwaitPercent = calculate(current.Cpu.Wait, last.Cpu.Wait)
		current.IrqPercent = calculate(current.Cpu.Irq, last.Cpu.Irq)
		current.NicePercent = calculate(current.Cpu.Nice, last.Cpu.Nice)
		current.SoftIrqPercent = calculate(current.Cpu.SoftIrq, last.Cpu.SoftIrq)
		current.StealPercent = calculate(current.Cpu.Stolen, last.Cpu.Stolen)
	}

	return current
}

func GetCpuPercentageList(last, current []CpuTimes) []CpuTimes {

	if last != nil && current != nil && len(last) == len(current) {

		calculate := func(field2 uint64, field1 uint64, all_delta uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(all_delta)
			return system.Round(perc, .5, 4)
		}

		for i := 0; i < len(last); i++ {
			allDelta := current[i].Cpu.Total() - last[i].Cpu.Total()
			current[i].UserPercent = calculate(current[i].Cpu.User, last[i].Cpu.User, allDelta)
			current[i].SystemPercent = calculate(current[i].Cpu.Sys, last[i].Cpu.Sys, allDelta)
			current[i].IdlePercent = calculate(current[i].Cpu.Idle, last[i].Cpu.Idle, allDelta)
			current[i].IOwaitPercent = calculate(current[i].Cpu.Wait, last[i].Cpu.Wait, allDelta)
			current[i].IrqPercent = calculate(current[i].Cpu.Irq, last[i].Cpu.Irq, allDelta)
			current[i].NicePercent = calculate(current[i].Cpu.Nice, last[i].Cpu.Nice, allDelta)
			current[i].SoftIrqPercent = calculate(current[i].Cpu.SoftIrq, last[i].Cpu.SoftIrq, allDelta)
			current[i].StealPercent = calculate(current[i].Cpu.Stolen, last[i].Cpu.Stolen, allDelta)

		}

	}

	return current
}

func (cpu *CPU) AddCpuPercentage(t2 *CpuTimes) {
	cpu.LastCpuTimes = GetCpuPercentage(cpu.LastCpuTimes, t2)
}

func (cpu *CPU) AddCpuPercentageList(t2 []CpuTimes) {
	cpu.LastCpuTimesList = GetCpuPercentageList(cpu.LastCpuTimesList, t2)
}
