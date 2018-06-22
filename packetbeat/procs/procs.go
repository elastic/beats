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

package procs

import (
	"net"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/gosigar"
)

// This controls how often process info for a running process is reloaded
// A big value means less unnecessary refreshes at a higher risk of missing
// a PID being recycled by the OS
const processCacheExpiration = time.Second * 30

type portProcMapping struct {
	port uint16
	pid  int
	proc *process
}

type process struct {
	name        string
	commandLine string
	// To control cache expiration
	expiration time.Time
}

// Allow the OS-dependant implementation to be replaced by a mock for testing
type processWatcherImpl interface {
	// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
	// that owns them.
	GetLocalPortToPIDMapping() (ports map[uint16]int, err error)
	// GetProcessCommandLine returns the command line for a given process.
	GetProcessCommandLine(pid int) string
	// GetLocalIPs returns the list of local addresses.
	GetLocalIPs() ([]net.IP, error)
}

type ProcessesWatcher struct {
	portProcMap  map[uint16]portProcMapping
	localAddrs   []net.IP
	processCache map[int]*process

	// config
	enabled    bool
	procConfig []ProcConfig

	impl processWatcherImpl
}

var ProcWatcher ProcessesWatcher

func (proc *ProcessesWatcher) Init(config ProcsConfig) error {
	return proc.initWithImpl(config, proc)
}

func (proc *ProcessesWatcher) initWithImpl(config ProcsConfig, impl processWatcherImpl) error {
	proc.impl = impl
	proc.portProcMap = make(map[uint16]portProcMapping)
	proc.processCache = make(map[int]*process)

	proc.enabled = config.Enabled
	if proc.enabled {
		logp.Info("Process watcher enabled")
	} else {
		logp.Info("Process watcher disabled")
	}

	// Read the local IP addresses
	var err error
	proc.localAddrs, err = impl.GetLocalIPs()
	if err != nil {
		logp.Err("Error getting local IP addresses: %s", err)
		proc.localAddrs = []net.IP{}
	}

	proc.procConfig = config.Monitored

	return nil
}

func (proc *ProcessesWatcher) FindProcessesTuple(tuple *common.IPPortTuple) (procTuple *common.CmdlineTuple) {
	procTuple = &common.CmdlineTuple{}

	if !proc.enabled {
		return
	}

	if proc.isLocalIP(tuple.SrcIP) {
		if p := proc.findProc(tuple.SrcPort); p != nil {
			procTuple.Src = []byte(p.name)
			procTuple.SrcCommand = []byte(p.commandLine)
			logp.Debug("procs", "Found process '%s' (%s) for port %d", p.commandLine, p.name, tuple.SrcPort)
		}
	}

	if proc.isLocalIP(tuple.DstIP) {
		if p := proc.findProc(tuple.DstPort); p != nil {
			procTuple.Dst = []byte(p.name)
			procTuple.DstCommand = []byte(p.commandLine)
			logp.Debug("procs", "Found process '%s' (%s) for port %d", p.commandLine, p.name, tuple.DstPort)
		}
	}

	return
}

func (proc *ProcessesWatcher) findProc(port uint16) *process {
	defer logp.Recover("FindProc exception")

	p, exists := proc.portProcMap[port]
	if exists {
		return p.proc
	}

	proc.updateMap()

	p, exists = proc.portProcMap[port]
	if exists {
		return p.proc
	}

	return nil
}

func (proc *ProcessesWatcher) updateMap() {
	if logp.HasSelector("procsdetailed") {
		start := time.Now()
		defer func() {
			logp.Debug("procsdetailed", "updateMap() took %v", time.Now().Sub(start))
		}()
	}

	ports, err := proc.impl.GetLocalPortToPIDMapping()
	if err != nil {
		logp.Err("unable to list local ports: %v", err)
	}

	proc.expireProcessCache()

	for port, pid := range ports {
		proc.updateMappingEntry(port, pid)
	}
}

func (proc *ProcessesWatcher) updateMappingEntry(port uint16, pid int) {
	prev, ok := proc.portProcMap[port]
	if ok && prev.pid == pid {
		// This port->pid mapping already exists
		return
	}

	p := proc.getProcessInfo(pid)
	if p == nil {
		return
	}

	// Simply overwrite old entries for now.
	// We never expire entries from this map. Since there are 65k possible
	// ports, the size of the dict can be max 1.5 MB, which we consider
	// reasonable.
	proc.portProcMap[port] = portProcMapping{port: port, pid: pid, proc: p}

	logp.Debug("procsdetailed", "updateMappingEntry(): port=%d pid=%d process='%s' name=%s",
		port, pid, p.commandLine, p.name)
}

func (proc *ProcessesWatcher) isLocalIP(ip net.IP) bool {
	if ip.IsLoopback() {
		return true
	}

	for _, addr := range proc.localAddrs {
		if ip.Equal(addr) {
			return true
		}
	}

	return false
}

func (proc *ProcessesWatcher) getProcessInfo(pid int) *process {
	if p, ok := proc.processCache[pid]; ok {
		return p
	}
	// Not in cache, resolve process info
	p := &process{
		commandLine: proc.impl.GetProcessCommandLine(pid),
		expiration:  time.Now().Add(processCacheExpiration),
	}
	// see if the command-line matches any 'grep' pattern
	for _, match := range proc.procConfig {
		if strings.Contains(p.commandLine, match.CmdlineGrep) {
			p.name = match.Process
			break
		}
	}
	proc.processCache[pid] = p
	return p
}

func (proc *ProcessesWatcher) expireProcessCache() {
	now := time.Now()
	for pid, info := range proc.processCache {
		if now.After(info.expiration) {
			delete(proc.processCache, pid)
		}
	}
}

// GetProcessCommandLine returns the command line for a given process.
func (proc *ProcessesWatcher) GetProcessCommandLine(pid int) (cmdLine string) {
	var procArgs gosigar.ProcArgs
	if err := procArgs.Get(pid); err == nil {
		cmdLine = strings.Join(procArgs.List, " ")
	} else {
		// Save PID without command-line to avoid continued errors for this process
		logp.Err("Unable to get command-line for pid %d: %v", pid, err)
	}
	return cmdLine
}

// GetLocalIPs returns the list of local addresses.
func (proc *ProcessesWatcher) GetLocalIPs() ([]net.IP, error) {
	return common.LocalIPAddrs()
}
