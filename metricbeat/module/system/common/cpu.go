package common

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
		all_delta := current.Cpu.Total() - last.Cpu.Total()

		if all_delta == 0 {
			// first inquiry
			return current
		}

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(all_delta)
			return Round(perc, .5, 4)
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
			return Round(perc, .5, 4)
		}

		for i := 0; i < len(last); i++ {
			all_delta := current[i].Cpu.Total() - last[i].Cpu.Total()
			current[i].UserPercent = calculate(current[i].Cpu.User, last[i].Cpu.User, all_delta)
			current[i].SystemPercent = calculate(current[i].Cpu.Sys, last[i].Cpu.Sys, all_delta)
			current[i].IdlePercent = calculate(current[i].Cpu.Idle, last[i].Cpu.Idle, all_delta)
			current[i].IOwaitPercent = calculate(current[i].Cpu.Wait, last[i].Cpu.Wait, all_delta)
			current[i].IrqPercent = calculate(current[i].Cpu.Irq, last[i].Cpu.Irq, all_delta)
			current[i].NicePercent = calculate(current[i].Cpu.Nice, last[i].Cpu.Nice, all_delta)
			current[i].SoftIrqPercent = calculate(current[i].Cpu.SoftIrq, last[i].Cpu.SoftIrq, all_delta)
			current[i].StealPercent = calculate(current[i].Cpu.Stolen, last[i].Cpu.Stolen, all_delta)

		}

	}

	return current
}

func (cpu *CPU) GetCpuStatEvent(cpuStat *CpuTimes) common.MapStr {
	result := common.MapStr{
		"user_p":    cpuStat.UserPercent,
		"system_p":  cpuStat.SystemPercent,
		"idle_p":    cpuStat.IdlePercent,
		"iowait_p":  cpuStat.IOwaitPercent,
		"irq_p":     cpuStat.IrqPercent,
		"nice_p":    cpuStat.NicePercent,
		"softirq_p": cpuStat.SoftIrqPercent,
		"steal_p":   cpuStat.StealPercent,
	}

	if cpu.CpuTicks {
		m := common.MapStr{
			"user":    cpuStat.User,
			"system":  cpuStat.Sys,
			"nice":    cpuStat.Nice,
			"idle":    cpuStat.Idle,
			"iowait":  cpuStat.Wait,
			"irq":     cpuStat.Irq,
			"softirq": cpuStat.SoftIrq,
			"steal":   cpuStat.Stolen,
		}
		return common.MapStrUnion(result, m)
	}
	return result

}

func (cpu *CPU) AddCpuPercentage(t2 *CpuTimes) {
	cpu.LastCpuTimes = GetCpuPercentage(cpu.LastCpuTimes, t2)
}

func (cpu *CPU) AddCpuPercentageList(t2 []CpuTimes) {
	cpu.LastCpuTimesList = GetCpuPercentageList(cpu.LastCpuTimesList, t2)
}

func (cpu *CPU) GetSystemStats() (common.MapStr, error) {
	loadStat, err := GetSystemLoad()
	if err != nil {
		logp.Warn("Getting load statistics: %v", err)
		return nil, err
	}
	cpuStat, err := GetCpuTimes()
	if err != nil {
		logp.Warn("Getting cpu times: %v", err)
		return nil, err
	}

	cpu.AddCpuPercentage(cpuStat)

	memStat, err := GetMemory()
	if err != nil {
		logp.Warn("Getting memory details: %v", err)
		return nil, err
	}
	AddMemPercentage(memStat)

	swapStat, err := GetSwap()
	if err != nil {
		logp.Warn("Getting swap details: %v", err)
		return nil, err
	}
	AddSwapPercentage(swapStat)

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "system",
		"load":       loadStat,
		"cpu":        cpu.GetCpuStatEvent(cpuStat),
		"mem":        GetMemoryEvent(memStat),
		"swap":       GetSwapEvent(swapStat),
	}

	return event, nil
}

func (cpu *CPU) GetCoreStats() ([]common.MapStr, error) {

	events := []common.MapStr{}

	cpuCoreStat, err := GetCpuTimesList()
	if err != nil {
		logp.Warn("Getting cpu core times: %v", err)
		return nil, err
	}
	cpu.AddCpuPercentageList(cpuCoreStat)

	for coreNumber, stat := range cpuCoreStat {
		coreStat := cpu.GetCpuStatEvent(&stat)
		coreStat["id"] = coreNumber

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "core",
			"core":       coreStat,
		}
		events = append(events, event)
	}

	return events, nil
}
