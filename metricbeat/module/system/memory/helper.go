// +build darwin freebsd linux openbsd windows

package memory

import "github.com/elastic/beats/metricbeat/module/system"

func GetPercentage(t1 uint64, t2 uint64) float64 {

	if t2 == 0 {
		return 0.0
	}

	perc := float64(t1) / float64(t2)
	return system.Round(perc, .5, 4)
}
