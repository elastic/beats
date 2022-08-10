// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package proc

import (
	"bytes"
	"io/ioutil"
	"strings"
)

type ProcIO struct {
	ReadBytes           string
	WriteBytes          string
	CancelledWriteBytes string
}

// ReadProcStat reads proccess stats from /proc/<pid>/io.
// The parsing code logic is borrowed from osquery C++ implementation and translated to Go.
// This makes the data returned from the `host_processes` table
// consistent with data returned from the original osquery `processes` table.
// https://github.com/osquery/osquery/blob/master/osquery/tables/system/linux/processes.cpp
func ReadIO(root string, pid string) (procio ProcIO, err error) {
	// Proc IO example
	// rchar: 1527371144
	// wchar: 1495591102
	// syscr: 481186
	// syscw: 255942
	// read_bytes: 14401536
	// write_bytes: 815329280
	// cancelled_write_bytes: 40976384
	fn := getProcAttr(root, pid, "io")
	b, err := ioutil.ReadFile(fn)
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
		switch k {
		case "read_bytes":
			procio.ReadBytes = strings.TrimSpace(bytesToString(detail[1]))
		case "write_bytes":
			procio.WriteBytes = strings.TrimSpace(bytesToString(detail[1]))
		case "cancelled_write_bytes":
			procio.CancelledWriteBytes = strings.TrimSpace(bytesToString(detail[1]))
		}
	}
	return procio, nil
}
