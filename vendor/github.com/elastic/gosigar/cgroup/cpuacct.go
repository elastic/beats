package cgroup

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/elastic/gosigar/sys/linux"
)

var clockTicks = uint64(linux.GetClockTicks())

// CPUAccountingSubsystem contains metrics from the "cpuacct" subsystem.
type CPUAccountingSubsystem struct {
	Metadata
	TotalNanos  uint64   `json:"total_nanos"`
	UsagePerCPU []uint64 `json:"usage_percpu_nanos"`
	// CPU time statistics for tasks in this cgroup.
	Stats CPUAccountingStats `json:"stats,omitempty"`
}

// CPUAccountingStats contains the stats reported from the cpuacct subsystem.
type CPUAccountingStats struct {
	UserNanos   uint64 `json:"user_nanos"`
	SystemNanos uint64 `json:"system_nanos"`
}

// get reads metrics from the "cpuacct" subsystem. path is the filepath to the
// cgroup hierarchy to read.
func (cpuacct *CPUAccountingSubsystem) get(path string) error {
	if err := cpuacctStat(path, cpuacct); err != nil {
		return err
	}

	if err := cpuacctUsage(path, cpuacct); err != nil {
		return err
	}

	if err := cpuacctUsagePerCPU(path, cpuacct); err != nil {
		return err
	}

	return nil
}

func cpuacctStat(path string, cpuacct *CPUAccountingSubsystem) error {
	f, err := os.Open(filepath.Join(path, "cpuacct.stat"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t, v, err := parseCgroupParamKeyValue(sc.Text())
		if err != nil {
			return err
		}
		switch t {
		case "user":
			cpuacct.Stats.UserNanos = convertJiffiesToNanos(v)
		case "system":
			cpuacct.Stats.SystemNanos = convertJiffiesToNanos(v)
		}
	}

	return sc.Err()
}

func cpuacctUsage(path string, cpuacct *CPUAccountingSubsystem) error {
	var err error
	cpuacct.TotalNanos, err = parseUintFromFile(path, "cpuacct.usage")
	if err != nil {
		return err
	}

	return nil
}

func cpuacctUsagePerCPU(path string, cpuacct *CPUAccountingSubsystem) error {
	contents, err := ioutil.ReadFile(filepath.Join(path, "cpuacct.usage_percpu"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var values []uint64
	usages := bytes.Fields(contents)
	for _, usage := range usages {
		value, err := parseUint(usage)
		if err != nil {
			return err
		}

		values = append(values, value)
	}
	cpuacct.UsagePerCPU = values

	return nil
}

func convertJiffiesToNanos(j uint64) uint64 {
	return (j * uint64(time.Second)) / clockTicks
}
