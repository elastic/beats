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
	LastCpuTimes     *CpuTimes
	LastCpuTimesList []CpuTimes
}

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
			delta := int64(field2 - field1)
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
		"cpu":        GetCpuStatEvent(cpuStat),
		"mem":        GetMemoryEvent(memStat),
		"swap":       GetSwapEvent(swapStat),
	}

	if cpu.CpuPerCore {

		cpuCoreStat, err := GetCpuTimesList()
		if err != nil {
			logp.Warn("Getting cpu core times: %v", err)
			return nil, err
		}
		cpu.AddCpuPercentageList(cpuCoreStat)

		cpus := common.MapStr{}

		for coreNumber, stat := range cpuCoreStat {
			cpus["cpu"+strconv.Itoa(coreNumber)] = GetCpuStatEvent(&stat)
		}
		event["cpus"] = cpus
	}

	return event, nil
}
