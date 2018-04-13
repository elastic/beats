// +build darwin freebsd linux windows

package process_summary

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/metric/system/process"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	sigar "github.com/elastic/gosigar"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("system", "process_summary", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	pids, err := process.Pids()
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch the list of PIDs")
	}

	var summary struct {
		sleeping int
		running  int
		idle     int
		stopped  int
		zombie   int
		unknown  int
	}

	for _, pid := range pids {
		state := sigar.ProcState{}
		err = state.Get(pid)
		if err != nil {
			summary.unknown += 1
			continue
		}

		switch byte(state.State) {
		case 'S':
			summary.sleeping++
		case 'R':
			summary.running++
		case 'D':
			summary.idle++
		case 'I':
			summary.idle++
		case 'T':
			summary.stopped++
		case 'Z':
			summary.zombie++
		default:
			logp.Err("Unknown state <%v> for process with pid %d", state.State, pid)
			summary.unknown++
		}
	}

	event := common.MapStr{
		"total":    len(pids),
		"sleeping": summary.sleeping,
		"running":  summary.running,
		"idle":     summary.idle,
		"stopped":  summary.stopped,
		"zombie":   summary.zombie,
		"unknown":  summary.unknown,
	}
	// change the name space to use . instead of _
	event[mb.NamespaceKey] = "process.summary"

	return event, nil
}
