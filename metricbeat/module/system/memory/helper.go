// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/system"
	sigar "github.com/elastic/gosigar"
)

type MemStat struct {
	sigar.Mem
	UsedPercent       float64 `json:"used_p"`
	ActualUsedPercent float64 `json:"actual_used_p"`
}

func GetMemory() (*MemStat, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &MemStat{Mem: mem}, nil
}

func AddMemPercentage(m *MemStat) {

	if m.Mem.Total == 0 {
		return
	}

	perc := float64(m.Mem.Used) / float64(m.Mem.Total)
	m.UsedPercent = system.Round(perc, .5, 4)

	actualPerc := float64(m.Mem.ActualUsed) / float64(m.Mem.Total)
	m.ActualUsedPercent = system.Round(actualPerc, .5, 4)
}

type SwapStat struct {
	sigar.Swap
	UsedPercent float64 `json:"used_p"`
}

func GetSwap() (*SwapStat, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	return &SwapStat{Swap: swap}, nil

}

func GetMemoryEvent(memStat *MemStat) common.MapStr {
	return common.MapStr{
		"total":         memStat.Total,
		"used":          memStat.Used,
		"free":          memStat.Free,
		"actual_used":   memStat.ActualUsed,
		"actual_free":   memStat.ActualFree,
		"used_p":        memStat.UsedPercent,
		"actual_used_p": memStat.ActualUsedPercent,
	}
}

func GetSwapEvent(swapStat *SwapStat) common.MapStr {
	return common.MapStr{
		"total":  swapStat.Total,
		"used":   swapStat.Used,
		"free":   swapStat.Free,
		"used_p": swapStat.UsedPercent,
	}
}

func AddSwapPercentage(s *SwapStat) {
	if s.Swap.Total == 0 {
		return
	}

	perc := float64(s.Swap.Used) / float64(s.Swap.Total)
	s.UsedPercent = system.Round(perc, .5, 4)
}
