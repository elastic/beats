// +build darwin freebsd linux openbsd windows

package memory

import (
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

	actual_perc := float64(m.Mem.ActualUsed) / float64(m.Mem.Total)
	m.ActualUsedPercent = system.Round(actual_perc, .5, 4)
}
