// +build darwin freebsd linux openbsd

package swap

import (
	"github.com/elastic/beats/metricbeat/module/system"
	sigar "github.com/elastic/gosigar"
)

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

func AddSwapPercentage(s *SwapStat) {
	if s.Swap.Total == 0 {
		return
	}

	perc := float64(s.Swap.Used) / float64(s.Swap.Total)
	s.UsedPercent = system.Round(perc, .5, 4)
}
