package system

import (
	"github.com/elastic/beats/libbeat/common"
	sigar "github.com/elastic/gosigar"
)

type SwapStat struct {
	sigar.Swap
	UsedPercent float64 `json:"used_p"`
}

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
}

type MemStat struct {
	sigar.Mem
	UsedPercent       float64 `json:"used_p"`
	ActualUsedPercent float64 `json:"actual_used_p"`
}

func GetSystemLoad() (*SystemLoad, error) {

	concreteSigar := sigar.ConcreteSigar{}
	avg, err := concreteSigar.GetLoadAverage()
	if err != nil {
		return nil, err
	}

	return &SystemLoad{
		Load1:  avg.One,
		Load5:  avg.Five,
		Load15: avg.Fifteen,
	}, nil
}

func GetMemory() (*MemStat, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &MemStat{Mem: mem}, nil
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

func GetSwap() (*SwapStat, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	return &SwapStat{Swap: swap}, nil

}

func GetSwapEvent(swapStat *SwapStat) common.MapStr {
	return common.MapStr{
		"total":  swapStat.Total,
		"used":   swapStat.Used,
		"free":   swapStat.Free,
		"used_p": swapStat.UsedPercent,
	}
}

func AddMemPercentage(m *MemStat) {

	if m.Mem.Total == 0 {
		return
	}

	perc := float64(m.Mem.Used) / float64(m.Mem.Total)
	m.UsedPercent = Round(perc, .5, 2)

	actual_perc := float64(m.Mem.ActualUsed) / float64(m.Mem.Total)
	m.ActualUsedPercent = Round(actual_perc, .5, 2)
}

func AddSwapPercentage(s *SwapStat) {
	if s.Swap.Total == 0 {
		return
	}

	perc := float64(s.Swap.Used) / float64(s.Swap.Total)
	s.UsedPercent = Round(perc, .5, 2)
}
