// +build darwin linux windows

/*

An example event looks as following:


    {
      "@timestamp": "2016-04-26T19:24:19.108Z",
      "beat": {
        "hostname": "ruflin",
        "name": "ruflin"
      },
      "metricset": "process",
      "module": "system",
      "rtt": 20982,
      "system-process": {
        "cmdline": ".\/metricbeat -e -d * -c metricbeat.dev.yml",
        "cpu": {
          "start_time": "21:24",
          "system": 32,
          "total": 79,
          "total_p": 1.2791,
          "user": 47
        },
        "mem": {
          "rss": 11538432,
          "rss_p": 0,
          "share": 0,
          "size": 587196518400
        },
        "name": "metricbeat",
        "pid": 27769,
        "ppid": 26608,
        "state": "running",
        "username": "ruflin"
      },
      "type": "metricsets"
    }

*/

package process

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/topbeat/system"
)

func init() {
	if err := mb.Registry.AddMetricSet("system", "process", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	stats *system.ProcStats
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	m := &MetricSet{
		BaseMetricSet: base,
		stats:         &system.ProcStats{},
	}

	m.stats.Procs = []string{".*"} //all processes
	m.stats.ProcStats = true
	m.stats.InitProcStats()
	return m, nil
}

func (m *MetricSet) Fetch(host string) (events []common.MapStr, err error) {
	return m.stats.GetProcStats()
}
