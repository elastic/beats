package system

import (
	"strconv"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	sigar "github.com/elastic/gosigar"
)

type CPU struct {
	CpuPerCore       bool
	LastCpuTimes     *sigar.Cpu
	LastCpuTimesList []sigar.Cpu
	CpuTicks         bool
}

func GetCpuTimes() (*sigar.Cpu, error) {

	cpu := sigar.Cpu{}
	err := cpu.Get()
	if err != nil {
		return nil, err
	}

	return &cpu, nil
}

func GetCpuTimesList() ([]sigar.Cpu, error) {

	cpuList := sigar.CpuList{}
	err := cpuList.Get()
	if err != nil {
		return nil, err
	}

	cpuTimes := make([]sigar.Cpu, len(cpuList.List))

	for i, cpu := range cpuList.List {
		cpuTimes[i] = cpu
	}

	return cpuTimes, nil
}

func calculateCpuPercentages(last, current *sigar.Cpu) common.MapStr {

	emptyMapStr := common.MapStr{
		"user_p":    0.0,
		"system_p":  0.0,
		"idle_p":    0.0,
		"iowait_p":  0.0,
		"irq_p":     0.0,
		"softirq_p": 0.0,
		"nice_p":    0.0,
		"steal_p":   0.0,
	}

	if last != nil && current != nil {
		all_delta := current.Total() - last.Total()

		if all_delta == 0 {
			// first inquiry
			return emptyMapStr
		}

		calculate := func(field2 uint64, field1 uint64) float64 {

			perc := 0.0
			delta := int64(field2 - field1)
			perc = float64(delta) / float64(all_delta)
			logp.Debug("system", "perc %f", perc)
			return Round(perc, .5, 4)
		}
		return common.MapStr{
			"user_p":    calculate(current.User, last.User),
			"system_p":  calculate(current.Sys, last.Sys),
			"idle_p":    calculate(current.Idle, last.Idle),
			"iowait_p":  calculate(current.Wait, last.Wait),
			"irq_p":     calculate(current.Irq, last.Irq),
			"nice_p":    calculate(current.Nice, last.Nice),
			"softirq_p": calculate(current.SoftIrq, last.SoftIrq),
			"steal_p":   calculate(current.Stolen, last.Stolen),
		}
	}
	return emptyMapStr
}

func (cpu *CPU) generateCpuStatEvent(last, current *sigar.Cpu) common.MapStr {

	cpuStats := calculateCpuPercentages(last, current)

	if cpu.CpuTicks {
		m := common.MapStr{
			"user":    current.User,
			"system":  current.Sys,
			"nice":    current.Nice,
			"idle":    current.Idle,
			"iowait":  current.Wait,
			"irq":     current.Irq,
			"softirq": current.SoftIrq,
			"steal":   current.Stolen,
		}
		return common.MapStrUnion(cpuStats, m)
	}
	return cpuStats

}
func (cpu *CPU) saveCpuTimes(t *sigar.Cpu) {
	cpu.LastCpuTimes = t
}

func (cpu *CPU) saveCpuTimesList(t []sigar.Cpu) {
	cpu.LastCpuTimesList = t
}

func (cpu *CPU) GetCpuStatEvent(current *sigar.Cpu) common.MapStr {

	last := cpu.LastCpuTimes

	cpuStats := cpu.generateCpuStatEvent(last, current)

	cpu.saveCpuTimes(current)

	return cpuStats
}

func (cpu *CPU) GetCpuStatForCores(current []sigar.Cpu) common.MapStr {

	cpus := common.MapStr{}

	for coreNumber, stat := range current {
		if len(cpu.LastCpuTimesList) < coreNumber+1 {
			cpus["cpu"+strconv.Itoa(coreNumber)] = cpu.generateCpuStatEvent(nil, &stat)
		} else {
			cpus["cpu"+strconv.Itoa(coreNumber)] = cpu.generateCpuStatEvent(&cpu.LastCpuTimesList[coreNumber], &stat)
		}
	}

	cpu.saveCpuTimesList(current)

	return cpus
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

	memStat, err := GetMemory()
	if err != nil {
		logp.Warn("Getting memory details: %v", err)
		return nil, err
	}

	swapStat, err := GetSwap()
	if err != nil {
		logp.Warn("Getting swap details: %v", err)
		return nil, err
	}

	event := common.MapStr{
		"@timestamp": common.Time(time.Now()),
		"type":       "system",
		"load":       loadStat,
		"cpu":        cpu.GetCpuStatEvent(cpuStat),
		"mem":        GetMemoryEvent(memStat),
		"swap":       GetSwapEvent(swapStat),
	}

	if cpu.CpuPerCore {

		cpuCoreStat, err := GetCpuTimesList()
		if err != nil {
			logp.Warn("Getting cpu core times: %v", err)
			return nil, err
		}
		event["cpus"] = cpu.GetCpuStatForCores(cpuCoreStat)
	}

	return event, nil
}
