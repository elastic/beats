// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package hostprocesses

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/client"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/hostfs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/proc"
	elastichostprocesses "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_host_processes"
)

const (
	clkTck      = 100
	msIn1CLKTCK = (1000 / clkTck)
)

func init() {
	elastichostprocesses.RegisterGenerateFunc(func(ctx context.Context, queryContext table.QueryContext, log *logger.Logger, _ *client.ResilientClient) ([]elastichostprocesses.Result, error) {
		return getResults(ctx, queryContext, log)
	})
}

func getResults(_ context.Context, queryContext table.QueryContext, log *logger.Logger) ([]elastichostprocesses.Result, error) {
	root := hostfs.GetPath("")
	log.Infof("generating host_processes table with root path: %s", root)
	return genProcesses(root, queryContext)
}

func dirExists(dirp string) (ok bool, err error) {
	var stat os.FileInfo
	if stat, err = os.Stat(dirp); err == nil && stat.IsDir() {
		ok = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return ok, err
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
	if len(pidset) == 0 {
		return proc.List(root)
	}
	pids := make([]string, 0, len(pidset))
	for pid := range pidset {
		pids = append(pids, pid)
	}
	return pids, nil
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

func atoi32(s string) int32 {
	n, _ := strconv.ParseInt(s, 10, 32)
	return int32(n)
}

func formatClicks(clicks string) int64 {
	n, err := strconv.ParseUint(clicks, 10, 64)
	if err != nil {
		return 0
	}
	return int64(n * msIn1CLKTCK)
}

func genProcesses(root string, queryContext table.QueryContext) ([]elastichostprocesses.Result, error) {
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
	var res []elastichostprocesses.Result
	for _, pid := range pids {
		rec := genProcess(root, pid, systemBootTime)
		if rec != nil {
			res = append(res, *rec)
		}
	}
	return res, nil
}

func genProcess(root string, pid string, systemBootTime int64) *elastichostprocesses.Result {
	pstat, err := proc.ReadStat(root, pid)
	if err != nil {
		return nil
	}
	r := &elastichostprocesses.Result{
		Pid:          atoi64(pid),
		Parent:       atoi64(pstat.Parent),
		Path:         mustString(proc.ReadLink(root, pid, "exe")),
		Name:         pstat.Name,
		Pgroup:       atoi64(pstat.Group),
		State:        pstat.State,
		Nice:         atoi32(pstat.Nice),
		Threads:      atoi32(pstat.Threads),
		Cmdline:      mustString(proc.ReadCmdLine(root, pid)),
		Cwd:          mustString(proc.ReadLink(root, pid, "cwd")),
		Root:         mustString(proc.ReadLink(root, pid, "root")),
		Uid:          atoi64(pstat.RealUID),
		Euid:         atoi64(pstat.EffectiveUID),
		Suid:         atoi64(pstat.SavedUID),
		Gid:          atoi64(pstat.RealGID),
		Egid:         atoi64(pstat.EffectiveGID),
		Sgid:         atoi64(pstat.SavedGID),
		OnDisk:       -1,
		WiredSize:    0,
		ResidentSize: atoi64(pstat.ResidentSize),
		TotalSize:    atoi64(pstat.TotalSize),
		UserTime:     formatClicks(pstat.UserTime),
		SystemTime:   formatClicks(pstat.SystemTime),
	}
	if procIO, err := proc.ReadIO(root, pid); err == nil {
		r.DiskBytesRead = atoi64(procIO.ReadBytes)
		written := atoi64(procIO.WriteBytes)
		cancelled := atoi64(procIO.CancelledWriteBytes)
		r.DiskBytesWritten = written - cancelled
	}
	pst, err := strconv.ParseInt(pstat.StartTime, 10, 64)
	if err != nil || systemBootTime == 0 {
		r.StartTime = -1
	} else {
		r.StartTime = systemBootTime + pst/clkTck
	}
	return r
}

func mustString(s string, _ error) string {
	return s
}
