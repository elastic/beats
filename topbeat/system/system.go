package system

import (
	"github.com/elastic/beats/libbeat/common"
	sigar "github.com/elastic/gosigar"
)

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`
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

func GetMemory() (*sigar.Mem, error) {

	mem := sigar.Mem{}
	err := mem.Get()
	if err != nil {
		return nil, err
	}

	return &mem, nil
}

func GetMemoryEvent(memStat *sigar.Mem) common.MapStr {

	stats := getMemPercentage(memStat)

	return common.MapStrUnion(stats,
		common.MapStr{
			"total":       memStat.Total,
			"used":        memStat.Used,
			"free":        memStat.Free,
			"actual_used": memStat.ActualUsed,
			"actual_free": memStat.ActualFree,
		})
}

func GetSwap() (*sigar.Swap, error) {

	swap := sigar.Swap{}
	err := swap.Get()
	if err != nil {
		return nil, err
	}

	return &swap, nil

}

func GetSwapEvent(swapStat *sigar.Swap) common.MapStr {
	stats := getSwapPercentage(swapStat)

	return common.MapStrUnion(stats,
		common.MapStr{
			"total": swapStat.Total,
			"used":  swapStat.Used,
			"free":  swapStat.Free,
		})
}

func getMemPercentage(m *sigar.Mem) common.MapStr {

	if m.Total == 0 {
		return common.MapStr{
			"used_p":        0.0,
			"actual_used_p": 0.0,
		}
	}

	used_p := float64(m.Used) / float64(m.Total)
	actual_used_p := float64(m.ActualUsed) / float64(m.Total)

	return common.MapStr{
		"used_p":        Round(used_p, .5, 4),
		"actual_used_p": Round(actual_used_p, .5, 4),
	}
}

func getSwapPercentage(s *sigar.Swap) common.MapStr {

	if s.Total == 0 {
		return common.MapStr{
			"used_p": 0.0,
		}
	}

	perc := float64(s.Used) / float64(s.Total)
	return common.MapStr{
		"used_p": Round(perc, .5, 4),
	}
}
