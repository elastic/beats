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

package perf

import (
	"fmt"
	"os"
	"syscall"

	"github.com/hodgesds/perf-utils"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/process"
)

// procInfo contains all the controlling information on a given PID.
type procInfo struct {
	PID          int
	IsDead       bool
	deadCount    int
	Metadata     common.MapStr
	SoftwareProc perf.SoftwareProfiler
	HardwareProc perf.HardwareProfiler
}

// tryFindNewPid will look to see if a similar process is associated with the same pid.
// This is a tad inexact, and in certan cases this may be inexact
// If we can't re-find the process after a few attempts, mark the PID as dead and give up.
func (p *procInfo) tryFindNewPid(proclist []common.MapStr) {

	cmdline, _ := p.Metadata["cmdline"]
	cwd, _ := p.Metadata["cwd"]
	username, _ := p.Metadata["username"]

	newPid := -1
	newMeta := common.MapStr{}
	for _, proc := range proclist {
		testcmdline, ok1 := proc["cmdline"]
		testCwd, ok2 := proc["cwd"]
		testUsername, ok3 := proc["username"]
		matches := 0
		if ok1 && testcmdline == cmdline {
			matches++
		}
		if ok2 && testCwd == cwd {
			matches++
		}
		if ok3 && testUsername == username {
			matches++
		}

		if matches >= 2 {
			newPid = proc["pid"].(int)
			newMeta = proc
			break
		}

	}

	if newPid == -1 {
		p.deadCount++
		if p.deadCount == 5 {
			p.IsDead = true
			p.SoftwareProc.Close()
			p.HardwareProc.Close()
		}
		return
	}
	p.deadCount = 0

	p.PID = newPid
	p.Metadata = newMeta

	if p.SoftwareProc != nil {
		p.SoftwareProc.Close()
		sw := perf.NewSoftwareProfiler(p.PID, -1)
		p.SoftwareProc = sw
	}

	if p.HardwareProc != nil {
		p.HardwareProc.Close()
		hw := perf.NewHardwareProfiler(p.PID, -1)
		p.HardwareProc = hw
	}

}

// checkAndReplace checks to see if the PID still exists.
// the underlying reads that perf depends on will just renturn 0 if a pid no longer exists
// to check to see if a service has been restarted, we have to actually look for the pid
func (p *procInfo) checkAndReplace() error {
	logger := logp.NewLogger("perf")

	if p.IsDead {
		return fmt.Errorf("PID %d is dead", p.PID)
	}

	//On Linux this will never return an error
	proc, _ := os.FindProcess(p.PID)
	if proc != nil {
		return nil
	}

	err := proc.Signal(syscall.Signal(0))
	if err == nil {
		return nil
	}

	logger.Warn("PID %d has been lost. Attempting to find process.", p.PID)
	procname := p.Metadata["name"].(string)
	config := &process.Stats{Procs: []string{procname}}

	err = config.Init()
	if err != nil {
		return errors.Wrap(err, "error initializing process list")
	}

	procs, err := config.Get()
	if err != nil {
		return errors.Wrap(err, "error fetching processes")
	}

	// if we only have one proc, just skip the search
	if len(procs) == 1 {
		p.PID = procs[0]["pid"].(int)
		p.Metadata = procs[0]
		if p.SoftwareProc != nil {
			p.SoftwareProc.Close()
			sw := perf.NewSoftwareProfiler(p.PID, -1)
			p.SoftwareProc = sw
		}

		if p.HardwareProc != nil {
			p.HardwareProc.Close()
			hw := perf.NewHardwareProfiler(p.PID, -1)
			p.HardwareProc = hw
		}
	}

	p.tryFindNewPid(procs)

	return nil

}
