package perf

import (
	"github.com/elastic/beats/v7/libbeat/metric/system/process"
	"github.com/hodgesds/perf-utils"
	"github.com/pkg/errors"
)

// matchProcesses takes a config list and returns a list of associated processes.
// This is basically a search, so a single process term could return multiple processes.
// the ioctls that underpin perf require a pid.
func matchProcesses(procList []sampleConfig) ([]procInfo, error) {

	var monitorProcs []procInfo

	for _, proc := range procList {

		config := &process.Stats{Procs: []string{proc.ProcessGlob}}

		err := config.Init()
		if err != nil {
			return nil, errors.Wrap(err, "error initializing process list")
		}

		matches, err := config.Get()
		if err != nil {
			return nil, errors.Wrap(err, "Erorr fetching matching processes")
		}

		for _, match := range matches {
			pi := procInfo{}
			pid := match["pid"].(int)

			if proc.Events.HardwareEvents {
				hw := perf.NewHardwareProfiler(pid, -1)
				pi.HardwareProc = hw
			}
			if proc.Events.SoftwareEvents {
				sw := perf.NewSoftwareProfiler(pid, -1)
				pi.SoftwareProc = sw
			}

			pi.PID = pid
			pi.Metadata = match
			monitorProcs = append(monitorProcs, pi)
		}

	} // end of proc iteration

	return monitorProcs, nil
}
