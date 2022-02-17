// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tables

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/proc"
)

const (
	procDir = "/proc"

	// https://groups.google.com/g/fa.linux.kernel/c/JndVy0RgHHI/m/Nu7nkRfZ-c0J
	// CLK_TCK is 100 on x86. As it has always been. User land should never
	// care about whatever random value the kernel happens to use for the
	// actual timer tick at that particular moment. Especially since the kernel
	// internal timer tick may well be variable some day.

	// The fact that libproc believes that HZ can change is _their_ problem.
	// I've told people over and over that user-level HZ is a constant (and, on
	// x86, that constant is 100), and that won't change.

	// So in current 2.5.x times() still counts at 100Hz, and /proc files that
	// export clock_t still show the same 100Hz rate.

	// The fact that the kernel internally counts at some different rate should
	// be _totally_ invisible to user programs (except they get better latency
	// for stuff like select() and other timeouts).

	// Linus
	clkTck = 100

	msIn1CLKTCK = (1000 / clkTck)
)

func HostProcessesColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.BigIntColumn("pid"),
		table.TextColumn("name"),
		table.TextColumn("path"),
		table.TextColumn("cmdline"),
		table.TextColumn("state"),
		table.TextColumn("cwd"),
		table.TextColumn("root"),
		table.BigIntColumn("uid"),
		table.BigIntColumn("gid"),
		table.BigIntColumn("euid"),
		table.BigIntColumn("egid"),
		table.BigIntColumn("suid"),
		table.BigIntColumn("sgid"),
		table.IntegerColumn("on_disk"),
		table.BigIntColumn("wired_size"),
		table.BigIntColumn("resident_size"),
		table.BigIntColumn("total_size"),
		table.BigIntColumn("user_time"),
		table.BigIntColumn("system_time"),
		table.BigIntColumn("disk_bytes_read"),
		table.BigIntColumn("disk_bytes_written"),
		table.BigIntColumn("start_time"),
		table.BigIntColumn("parent"),
		table.BigIntColumn("pgroup"),
		table.IntegerColumn("threads"),
		table.IntegerColumn("nice"),
	}
}

func GetHostProcessesGenerateFunc() table.GenerateFunc {
	root := hostfs.GetPath("")

	return func(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
		return genProcesses(root, queryContext)
	}
}

func genProcesses(root string, queryContext table.QueryContext) ([]map[string]string, error) {
	systemBootTime, err := proc.ReadUptime(root)
	if err != nil {
		return nil, err
	}

	if systemBootTime > 0 {
		systemBootTime = time.Now().Unix() - systemBootTime
	}

	pids, err := getProcList(root, queryContext)
	if err != nil {
		return nil, err
	}

	var res []map[string]string
	for _, pid := range pids {
		rec := genProcess(root, pid, systemBootTime)
		if rec != nil {
			res = append(res, rec)
		}
	}

	return res, nil
}

func dirExists(dirp string) (ok bool, err error) {
	if stat, err := os.Stat(dirp); err == nil && stat.IsDir() {
		ok = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return
}

func getProcList(root string, queryContext table.QueryContext) ([]string, error) {
	pidset := make(map[string]struct{})
	if contraintList, ok := queryContext.Constraints["pid"]; ok && len(contraintList.Constraints) > 0 {
		for _, constraint := range contraintList.Constraints {
			if constraint.Operator == table.OperatorEquals {
				if ok, _ := dirExists(filepath.Join(root, constraint.Expression)); ok {
					pidset[constraint.Expression] = struct{}{}
				}
			}
		}
	}

	// Enumerate all processes pids
	if len(pidset) == 0 {
		return proc.List(root)
	}

	pids := make([]string, 0, len(pidset))
	for pid, _ := range pidset {
		pids = append(pids, pid)
	}
	return pids, nil
}

// genProcess can fail in multiple places, still return the full record.
// This is consistent with original osquery C++ implementation for processes records
func genProcess(root string, pid string, systemBootTime int64) map[string]string {
	pstat, err := proc.ReadStat(root, pid)
	if err != nil {
		return nil
	}

	r := make(map[string]string, 26)

	if procIO, err := proc.ReadIO(root, pid); err == nil {
		r["disk_bytes_read"] = procIO.ReadBytes
		written, _ := strconv.ParseUint(procIO.WriteBytes, 10, 64)
		cancelled, _ := strconv.ParseUint(procIO.CancelledWriteBytes, 10, 64)
		r["disk_bytes_written"] = strconv.FormatUint(written-cancelled, 10)
	}

	r["pid"] = pid
	r["parent"] = pstat.Parent
	r["path"] = mustString(proc.ReadLink(root, pid, "exe"))
	r["name"] = pstat.Name
	r["pgroup"] = pstat.Group
	r["state"] = pstat.State
	r["nice"] = pstat.Nice
	r["threads"] = pstat.Threads

	r["cmdline"] = mustString(proc.ReadCmdLine(root, pid))
	r["cwd"] = mustString(proc.ReadLink(root, pid, "cwd"))
	r["root"] = mustString(proc.ReadLink(root, pid, "root"))

	r["uid"] = pstat.RealUID
	r["euid"] = pstat.EffectiveUID
	r["suid"] = pstat.SavedUID

	r["gid"] = pstat.RealGID
	r["egid"] = pstat.EffectiveGID
	r["sgid"] = pstat.SavedGID

	// Can't check if the file exists on the host machine, setting to -1
	r["on_disk"] = "-1"

	r["wired_size"] = "0"
	r["resident_size"] = pstat.ResidentSize
	r["total_size"] = pstat.TotalSize

	r["user_time"] = formatClicks(pstat.UserTime)
	r["system_time"] = formatClicks(pstat.SystemTime)

	pst, err := strconv.ParseInt(pstat.StartTime, 10, 64)
	if err != nil || systemBootTime == 0 {
		r["start_time"] = "-1"
	} else {
		r["start_time"] = strconv.FormatInt(systemBootTime+pst/clkTck, 10)
	}

	return r
}

func mustString(s string, err error) string {
	return s
}

func formatClicks(clicks string) string {
	n, err := strconv.ParseUint(clicks, 10, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(n*msIn1CLKTCK, 10)
}
