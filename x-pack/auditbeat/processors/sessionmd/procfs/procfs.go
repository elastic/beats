// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package procfs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/timeutils"
	"github.com/elastic/beats/v7/x-pack/auditbeat/processors/sessionmd/types"
	"github.com/elastic/elastic-agent-libs/logp"
)

func MajorTTY(ttyNr uint32) uint16 {
	return uint16((ttyNr >> 8) & 0xf)
}

func MinorTTY(ttyNr uint32) uint16 {
	return uint16(((ttyNr & 0xfff00000) >> 20) | (ttyNr & 0xff))
}

// this interface exists so that we can inject a mock procfs reader for deterministic testing
type Reader interface {
	GetProcess(pid uint32) (ProcessInfo, error)
	GetAllProcesses() ([]ProcessInfo, error)
}

type ProcfsReader struct {
	logger logp.Logger
}

func NewProcfsReader(logger logp.Logger) ProcfsReader {
	return ProcfsReader{
		logger: logger,
	}
}

type Stat procfs.ProcStat

type ProcessInfo struct {
	PIDs       types.PIDInfo
	Creds      types.CredInfo
	CTTY       types.TTYDev
	Argv       []string
	Cwd        string
	Env        map[string]string
	Filename   string
	CGroupPath string
}

func credsFromProc(proc procfs.Proc) (types.CredInfo, error) {
	status, err := proc.NewStatus()
	if err != nil {
		return types.CredInfo{}, err
	}

	ruid, err := strconv.Atoi(status.UIDs[0])
	if err != nil {
		return types.CredInfo{}, err
	}

	euid, err := strconv.Atoi(status.UIDs[1])
	if err != nil {
		return types.CredInfo{}, err
	}

	suid, err := strconv.Atoi(status.UIDs[2])
	if err != nil {
		return types.CredInfo{}, err
	}

	rgid, err := strconv.Atoi(status.GIDs[0])
	if err != nil {
		return types.CredInfo{}, err
	}

	egid, err := strconv.Atoi(status.GIDs[1])
	if err != nil {
		return types.CredInfo{}, err
	}

	sgid, err := strconv.Atoi(status.GIDs[2])
	if err != nil {
		return types.CredInfo{}, err
	}

	// procfs library doesn't grab CapEff or CapPrm, make the direct syscall
	hdr := unix.CapUserHeader{
		Version: unix.LINUX_CAPABILITY_VERSION_3,
		Pid:     int32(proc.PID),
	}
	var data [2]unix.CapUserData
	err = unix.Capget(&hdr, &data[0])
	if err != nil {
		return types.CredInfo{}, err
	}
	permitted := uint64(data[1].Permitted) << 32
	permitted += uint64(data[0].Permitted)
	effective := uint64(data[1].Effective) << 32
	effective += uint64(data[0].Effective)

	return types.CredInfo{
		Ruid:         uint32(ruid),
		Euid:         uint32(euid),
		Suid:         uint32(suid),
		Rgid:         uint32(rgid),
		Egid:         uint32(egid),
		Sgid:         uint32(sgid),
		CapPermitted: permitted,
		CapEffective: effective,
	}, nil
}

func (r ProcfsReader) getProcessInfo(proc procfs.Proc) (ProcessInfo, error) {
	pid := uint32(proc.PID)
	// All other info can be best effort, but failing to get pid info and
	// start time is needed to register the process in the database
	stat, err := proc.Stat()
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("failed to read /proc/%d/stat: %w", pid, err)
	}

	argv, err := proc.CmdLine()
	if err != nil {
		argv = []string{}
	}

	exe, err := proc.Executable()
	if err != nil {
		if len(argv) > 0 {
			r.logger.Debugf("pid %d: got executable from cmdline: %s", pid, argv[0])
			exe = argv[0]
		} else {
			r.logger.Debugf("pid %d: failed to get executable path: %v", pid, err)
			exe = ""
		}
	}

	environ, err := r.getEnviron(pid)
	if err != nil {
		environ = nil
	}

	cwd, err := proc.Cwd()
	if err != nil {
		cwd = ""
	}

	creds, err := credsFromProc(proc)
	if err != nil {
		creds = types.CredInfo{}
	}

	cGroupPath := ""
	cgroups, err := proc.Cgroups()
	if err == nil {
	out:
		// Find the cgroup path from the PID controller.
		// NOTE: This does not support the unified hierarchy from cgroup v2, as bpf also does not currently support it.
		// When support is added for unified hierarchies, it should be added in bpf and userspace at the same time.
		// (Currently all supported cgroup v2 systems (GKE) are working as they send backwards compatible v1 hierarchies as well)
		for _, cgroup := range cgroups {
			for _, controller := range cgroup.Controllers {
				if controller == "pids" {
					cGroupPath = cgroup.Path
					break out
				}
			}
		}
	}

	startTimeNs := timeutils.TicksToNs(stat.Starttime)
	return ProcessInfo{
		PIDs: types.PIDInfo{
			StartTimeNS: startTimeNs,
			Tid:         pid,
			Tgid:        pid,
			Ppid:        uint32(stat.PPID),
			Pgid:        uint32(stat.PGRP),
			Sid:         uint32(stat.Session),
		},
		Creds: creds,
		CTTY: types.TTYDev{
			Major: MajorTTY(uint32(stat.TTY)),
			Minor: MinorTTY(uint32(stat.TTY)),
		},
		Cwd:        cwd,
		Argv:       argv,
		Env:        environ,
		Filename:   exe,
		CGroupPath: cGroupPath,
	}, nil
}

func (r ProcfsReader) GetProcess(pid uint32) (ProcessInfo, error) {
	proc, err := procfs.NewProc(int(pid))
	if err != nil {
		return ProcessInfo{}, err
	}
	return r.getProcessInfo(proc)
}

// returns empty slice on error
func (r ProcfsReader) GetAllProcesses() ([]ProcessInfo, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}

	ret := make([]ProcessInfo, 0)
	for _, proc := range procs {
		process_info, err := r.getProcessInfo(proc)
		if err != nil {
			r.logger.Warnf("failed to read process info for %v", proc.PID)
		}
		ret = append(ret, process_info)
	}

	return ret, nil
}

func (r ProcfsReader) getEnviron(pid uint32) (map[string]string, error) {
	proc, err := procfs.NewProc(int(pid))
	if err != nil {
		return nil, err
	}

	flatEnviron, err := proc.Environ()
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string)
	for _, entry := range flatEnviron {
		index := strings.Index(entry, "=")
		if index == -1 {
			continue
		}

		ret[entry[0:index]] = entry[index:]
	}

	return ret, nil
}
