// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidProcStatHeader  = errors.New("invalid /proc/stat header")
	ErrInvalidProcStatContent = errors.New("invalid /proc/stat content")
)

type ProcStat struct {
	Name         string
	RealUID      string
	RealGID      string
	EffectiveUID string
	EffectiveGID string
	SavedUID     string
	SavedGID     string
	ResidentSize string
	TotalSize    string
	State        string
	Parent       string
	Group        string
	Nice         string
	Threads      string
	UserTime     string
	SystemTime   string
	StartTime    string
}

func getProcAttr(root, pid, attr string) string {
	return filepath.Join(root, "/proc", pid, attr)
}

// ReadStat ReadProcStat reads proccess stats from /proc/<pid>/stat.
// The parsing code logic is borrowed from osquery C++ implementation and translated to Go.
// This makes the data returned from the `host_processes` table
// consistent with data returned from the original osquery `processes` table.
// https://github.com/osquery/osquery/blob/master/osquery/tables/system/linux/processes.cpp
func ReadStat(root string, pid string) (stat ProcStat, err error) {
	return ReadStatFS(os.DirFS("/"), root, pid)
}

func ReadStatFS(sysfs fs.FS, root string, pid string) (stat ProcStat, err error) {
	fn := getProcAttr(root, pid, "stat")
	b, err := fs.ReadFile(sysfs, fn)
	if err != nil {
		return
	}
	// Proc stat example
	// 6462 (bash) S 6402 6462 6462 34817 37849 4194304 14126 901131 0 191 15 9 3401 725 20 0 1 0 134150 20156416 1369 18446744073709551615 94186936238080 94186936960773 140723699470016 0 0 0 65536 3670020 1266777851 1 0 0 17 7 0 0 0 0 0 94186937191664 94186937239044 94186967023616 140723699476902 140723699476912 140723699476912 140723699478510 0
	pos := bytes.IndexByte(b, ')')
	if pos == -1 {
		return stat, ErrInvalidProcStatHeader
	}

	content := bytesToString(b[pos+2:])
	details := strings.Split(content, " ")
	if len(details) < 19 {
		return stat, ErrInvalidProcStatContent
	}

	stat.State = details[0]
	stat.Parent = details[1]
	stat.Group = details[2]
	stat.UserTime = details[11]
	stat.SystemTime = details[12]
	stat.Nice = details[16]
	stat.Threads = details[17]
	stat.StartTime = details[19]

	fn = getProcAttr(root, pid, "status")
	b, err = fs.ReadFile(sysfs, fn)
	if err != nil {
		return
	}

	lines := bytes.Split(b, []byte{'\n'})
	for _, line := range lines {
		detail := bytes.SplitN(line, []byte{':'}, 2)
		if len(detail) != 2 {
			continue
		}

		k := strings.TrimSpace(bytesToString(detail[0]))
		v := bytesToString(detail[1])
		switch k {
		case "Name":
			stat.Name = strings.TrimSpace(v)
		case "VmRSS":
			if len(v) >= 3 {
				stat.ResidentSize = strings.TrimSpace(v[:len(v)-3] + "000")
			}
		case "VmSize":
			if len(v) >= 3 {
				stat.TotalSize = strings.TrimSpace(v[:len(v)-3] + "000")
			}
		case "Gid":
			arr := strings.Split(v, "\t")
			if len(arr) == 4 {
				stat.RealGID = arr[0]
				stat.EffectiveGID = arr[1]
				stat.SavedGID = arr[2]
			}
		case "Uid":
			arr := strings.Split(v, "\t")
			if len(arr) == 4 {
				stat.RealUID = arr[0]
				stat.EffectiveUID = arr[1]
				stat.SavedUID = arr[2]
			}
		}
	}
	return stat, err
}
