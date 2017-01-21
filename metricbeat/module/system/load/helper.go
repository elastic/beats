// +build darwin freebsd linux openbsd

package load

import sigar "github.com/elastic/gosigar"

type SystemLoad struct {
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`

	/* normalized values load / cores */
	LoadNorm1  float64 `json:"load1_norm"`
	LoadNorm5  float64 `json:"load5_norm"`
	LoadNorm15 float64 `json:"load15_norm"`
}

func GetSystemLoad() (*SystemLoad, error) {

	concreteSigar := sigar.ConcreteSigar{}
	avg, err := concreteSigar.GetLoadAverage()
	if err != nil {
		return nil, err
	}

	cpuList := sigar.CpuList{}
	cpuList.Get()
	numCore := len(cpuList.List)

	return &SystemLoad{
		Load1:  avg.One,
		Load5:  avg.Five,
		Load15: avg.Fifteen,

		LoadNorm1:  avg.One / float64(numCore),
		LoadNorm5:  avg.Five / float64(numCore),
		LoadNorm15: avg.Fifteen / float64(numCore),
	}, nil
}
