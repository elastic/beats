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
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-sysinfo"
)

// This controls how often process info for a running process is reloaded
// A big value means less unnecessary refreshes at a higher risk of missing
// a PID being recycled by the OS
const processCacheExpiration = 30 * time.Second

var (
	anyIPv4 = net.IPv4zero.String()
	anyIPv6 = net.IPv6unspecified.String()
)

// ProcessWatcher implements process enrichment for network traffic.
type ProcessesWatcher struct {
	mu           sync.Mutex
	portProcMap  map[applayer.Transport]map[endpoint]portProcMapping
	localAddrs   []net.IP         // localAddrs lists IP addresses that are to be treated as local.
	processCache map[int]*process // processCache is a time-expiration cache of process details keyed on PID.

	enabled   bool         // enabled specifier whether the ProcessWatcher will be active.
	monitored []ProcConfig // monitored is the set of processes that are monitored by the ProcessWatcher.

	// watcher is the OS-dependent engine for the ProcessWatcher.
	watcher processWatcher
}

// endpoint is a network address/port number complex.
type endpoint struct {
	address string
	port    uint16
}

// portProcMapping is an association between an endpoint and a process.
type portProcMapping struct {
	endpoint endpoint // FIXME: This is never used.
	pid      int
	proc     *process
}

// process describes an OS process.
type process struct {
	pid, ppid      int
	name, exe, cwd string
	args           []string
	startTime      time.Time

	// expires is the time at which the process will be dropped
	// from the cache during enrichment queries.
	expires time.Time
}

// Init initializes the ProcessWatcher with the provided configuration.
func (proc *ProcessesWatcher) Init(config ProcsConfig) error {
	return proc.init(config, proc)
}

// processWatcher allows the OS-dependent implementation to be replaced by a mock for testing
type processWatcher interface {
	// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
	// that owns them.
	GetLocalPortToPIDMapping(transport applayer.Transport) (ports map[endpoint]int, err error)

	// GetProcess returns the process metadata.
	GetProcess(pid int) *process

	// GetLocalIPs returns the list of local addresses. If the returned error
	// is non-nil, the IP slice is nil.
	GetLocalIPs() ([]net.IP, error)
}

// init sets up the necessary data structures for the ProcessWatcher.
func (proc *ProcessesWatcher) init(config ProcsConfig, watcher processWatcher) error {
	proc.watcher = watcher
	proc.portProcMap = map[applayer.Transport]map[endpoint]portProcMapping{
		applayer.TransportUDP: make(map[endpoint]portProcMapping),
		applayer.TransportTCP: make(map[endpoint]portProcMapping),
	}

	proc.processCache = make(map[int]*process)

	proc.enabled = config.Enabled
	if proc.enabled {
		logp.Info("Process watcher enabled")
	} else {
		logp.Info("Process watcher disabled")
	}

	// Read the local IP addresses.
	var err error
	proc.localAddrs, err = watcher.GetLocalIPs()
	if err != nil {
		logp.Err("Error getting local IP addresses: %s", err)
	}

	proc.monitored = config.Monitored

	return nil
}

// FindProcessesTupleTCP looks up local process information for the source and
// destination addresses of TCP tuple
func (proc *ProcessesWatcher) FindProcessesTupleTCP(tuple *common.IPPortTuple) (procTuple *common.ProcessTuple) {
	return proc.FindProcessesTuple(tuple, applayer.TransportTCP)
}

// FindProcessesTupleUDP looks up local process information for the source and
// destination addresses of UDP tuple
func (proc *ProcessesWatcher) FindProcessesTupleUDP(tuple *common.IPPortTuple) (procTuple *common.ProcessTuple) {
	return proc.FindProcessesTuple(tuple, applayer.TransportUDP)
}

// FindProcessesTuple looks up local process information for the source and
// destination addresses of a tuple for the given transport protocol
func (proc *ProcessesWatcher) FindProcessesTuple(tuple *common.IPPortTuple, transport applayer.Transport) *common.ProcessTuple {
	var procTuple common.ProcessTuple
	if !proc.enabled {
		return &procTuple
	}
	proc.enrich(&procTuple.Src, tuple.SrcIP, tuple.SrcPort, transport)
	proc.enrich(&procTuple.Dst, tuple.DstIP, tuple.DstPort, transport)
	return &procTuple
}

// enrich adds process information to dst for the process associated with the given IP, port and
// transport if the IP is not local and the information is available to the ProcessWatcher.
func (proc *ProcessesWatcher) enrich(dst *common.Process, ip net.IP, port uint16, transport applayer.Transport) {
	if !proc.isLocalIP(ip) {
		return
	}
	p := proc.findProc(ip, port, transport)
	if p == nil {
		return
	}
	dst.PID = p.pid
	dst.PPID = p.ppid
	dst.Name = p.name
	dst.Args = p.args
	dst.Exe = p.exe
	dst.StartTime = p.startTime
	if logp.IsDebug("procs") {
		logp.Debug("procs", "Found process '%s' (pid=%d) for %s:%d/%s", p.name, p.pid, ip, port, transport)
	}
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

func (proc *ProcessesWatcher) findProc(address net.IP, port uint16, transport applayer.Transport) *process {
	proc.mu.Lock()
	procMap, ok := proc.portProcMap[transport]
	proc.mu.Unlock()
	if !ok {
		return nil
	}

	p, exists := lookupMapping(address, port, procMap)
	if exists {
		return p.proc
	}

	proc.updateMap(transport)

	p, exists = lookupMapping(address, port, procMap)
	if exists {
		return p.proc
	}

	return nil
}

func lookupMapping(address net.IP, port uint16, procMap map[endpoint]portProcMapping) (p portProcMapping, found bool) {
	// Precedence when one socket is bound to a specific IP:port and another one
	// to INADDR_ANY and same port is not clear. Seems that the last one to bind
	// takes precedence, and we don't have a way to tell.
	// This function takes the naive approach of giving precedence to the more
	// specific address and then to INADDR_ANY.
	if p, found = procMap[endpoint{address.String(), port}]; found {
		return p, found
	}

	nullAddr := anyIPv4
	if asIPv4 := address.To4(); asIPv4 == nil {
		nullAddr = anyIPv6
	}
	p, found = procMap[endpoint{nullAddr, port}]
	return p, found
}

func (proc *ProcessesWatcher) updateMap(transport applayer.Transport) {
	if logp.HasSelector("procsdetailed") {
		start := time.Now()
		defer func() {
			logp.Debug("procsdetailed", "updateMap() took %v", time.Since(start))
		}()
	}

	endpoints, err := proc.watcher.GetLocalPortToPIDMapping(transport)
	if err != nil {
		logp.Err("unable to list local ports: %v", err)
	}

	proc.expireProcessCache()

	for e, pid := range endpoints {
		proc.updateMappingEntry(transport, e, pid)
	}
}

func (proc *ProcessesWatcher) expireProcessCache() {
	now := time.Now()
	for pid, info := range proc.processCache {
		if now.After(info.expires) {
			delete(proc.processCache, pid)
		}
	}
}

func (proc *ProcessesWatcher) updateMappingEntry(transport applayer.Transport, e endpoint, pid int) {
	proc.mu.Lock()
	defer proc.mu.Unlock()
	prev, ok := proc.portProcMap[transport][e]
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
	proc.portProcMap[transport][e] = portProcMapping{endpoint: e, pid: pid, proc: p}

	if logp.IsDebug("procsdetailed") {
		logp.Debug("procsdetailed", "updateMappingEntry(): local=%s:%d/%s pid=%d process='%s'",
			e.address, e.port, transport, pid, p.name)
	}
}

// getProcessInfo returns a potentially cached process corresponding to the
// provided process ID.
//
// If any part of the process's argv contains a substring in proc.monitored.CmdlineGrep,
// the name of the process is replaced with the corresponding proc.monitored.Process.
// This behaviour is not recommended to be used and is not available to integrations
// packages by design.
func (proc *ProcessesWatcher) getProcessInfo(pid int) *process {
	if p, ok := proc.processCache[pid]; ok {
		return p
	}
	// Not in cache, resolve process info
	p := proc.watcher.GetProcess(pid)
	if p == nil {
		return nil
	}

	// The packetbeat.procs.monitored*.cmdline_grep allows you to overwrite
	// the process name with an alias.
	for _, match := range proc.monitored {
		if strings.Contains(strings.Join(p.args, " "), match.CmdlineGrep) {
			p.name = match.Process
			break
		}
	}
	proc.processCache[pid] = p
	return p
}

// GetProcess returns the process metadata.
func (proc *ProcessesWatcher) GetProcess(pid int) *process {
	if pid <= 0 {
		return nil
	}

	p, err := sysinfo.Process(pid)
	if err != nil {
		logp.Err("Unable to get command-line for PID %d: %v", pid, err)
		return nil
	}

	info, err := p.Info()
	if err != nil {
		logp.Err("Unable to get command-line for PID %d: %v", pid, err)
		return nil
	}

	return &process{
		pid:       info.PID,
		ppid:      info.PPID,
		name:      procName(info),
		exe:       info.Exe,
		cwd:       info.CWD,
		args:      info.Args,
		startTime: info.StartTime,
		expires:   time.Now().Add(processCacheExpiration),
	}
}

// GetLocalIPs returns the list of local addresses.
func (proc *ProcessesWatcher) GetLocalIPs() ([]net.IP, error) {
	return common.LocalIPAddrs()
}
