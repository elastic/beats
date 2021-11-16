// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package process

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/elastic/gosigar"
	"github.com/pkg/errors"
)

// GetSelfPid returns the PID for this process
func GetSelfPid() (int, error) {
	pid, err := os.Readlink(path.Join(gosigar.Procd, "self"))

	if err != nil {
		return 0, err
	}

	return strconv.Atoi(pid)
}

// FetchPids is the linux implementation of FetchPids
func FetchPids(hostfs string, filter func(name string) bool) ([]ProcState, error) {
	dir, err := os.Open(hostfs)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading from procfs %s", hostfs)
	}
	defer dir.Close()

	const readAllDirnames = -1 // see os.File.Readdirnames doc

	names, err := dir.Readdirnames(readAllDirnames)
	if err != nil {
		return nil, errors.Wrap(err, "error reading directory names")
	}

	list := make([]ProcState, 0)

	// Iterate over the directory, fetch just enough info so we can filter based on user input.
	logger := logp.L()
	for _, name := range names {

		if !dirIsPid(name) {
			continue
		}
		// Will this actually fail?
		pid, err := strconv.Atoi(name)
		if err != nil {
			logger.Debugf("Error converting PID name %s", name)
			continue
		}
		// Fetch proc state so we can get the name for filtering based on user's filter.
		status, err := GetInfoForPid(hostfs, pid)
		if err != nil {
			logger.Debugf("Skipping over PID=%d, due to: %d", pid, err)
			continue
		}
		// Filter based on user-supplied func
		if !filter(status.Name) {
			logger.Debugf("Process name does not matches the provided regex; PID=%d; name=%s", pid, status.Name)
			continue
		}
		list = append(list, status)
	}

	return list, nil
}

// GetInfoForPid fetches the basic hostinfo from /proc/[PID]/stat
func GetInfoForPid(hostfs string, pid int) (ProcState, error) {
	path := filepath.Join(hostfs, strconv.Itoa(pid), "stat")
	data, err := ioutil.ReadFile(path)
	// Transform the error into a more sensible error in cases where the directory doesn't exist, i.e the process is gone
	if err != nil {
		if os.IsNotExist(err) {
			return ProcState{}, syscall.ESRCH
		} else {
			return ProcState{}, errors.Wrapf(err, "error reading procdir %s", path)
		}
	}

	state := ProcState{}

	// Extract the comm value with is surrounded by parentheses.
	lIdx := bytes.Index(data, []byte("("))
	rIdx := bytes.LastIndex(data, []byte(")"))
	if lIdx < 0 || rIdx < 0 || lIdx >= rIdx || rIdx+2 >= len(data) {
		return state, fmt.Errorf("failed to extract comm for pid %d from '%v'", pid, string(data))
	}
	state.Name = string(data[lIdx+1 : rIdx])

	// Extract the rest of the fields that we are interested in.
	fields := bytes.Fields(data[rIdx+2:])
	if len(fields) <= 36 {
		return state, fmt.Errorf("expected more stat fields for pid %d from '%v'", pid, string(data))
	}

	interests := bytes.Join([][]byte{
		fields[0], // state
		fields[1], // ppid
		fields[2], // pgrp
	}, []byte(" "))

	var procState string
	var ppid, pgid int

	_, err = fmt.Fscan(bytes.NewBuffer(interests),
		&procState,
		&ppid,
		&pgid,
	)
	if err != nil {
		return state, fmt.Errorf("failed to parse stat fields for pid %d from '%v': %v", pid, string(data), err)
	}
	state.State = getProcState(procState[0])
	state.Ppid = opt.IntWith(ppid)
	state.Pgid = opt.IntWith(pgid)

	return state, nil
}

func dirIsPid(name string) bool {
	if name[0] < '0' || name[0] > '9' {
		return false
	}
	return true
}
