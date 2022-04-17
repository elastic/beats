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
	"path/filepath"
	"strings"
	"time"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/packetbeat/protos/applayer"
	"github.com/menderesk/go-sysinfo"
)

// This controls how often process info for a running process is reloaded
// A big value means less unnecessary refreshes at a higher risk of missing
// a PID being recycled by the OS
const processCacheExpiration = time.Second * 30

var (
	anyIPv4 = net.IPv4zero.String()
	anyIPv6 = net.IPv6unspecified.String()
)

type endpoint struct {
	address string
	port    uint16
}

type portProcMapping struct {
	endpoint endpoint
	pid      int
	proc     *process
}

type process struct {
	pid, ppid      int
	name, exe, cwd string
	args           []string
	startTime      time.Time

	// To control cache expiration
	expiration time.Time
}

// Allow the OS-dependent implementation to be replaced by a mock for testing
type processWatcherImpl interface {
	// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
	// that owns them.
	GetLocalPortToPIDMapping(transport applayer.Transport) (ports map[endpoint]int, err error)
	// GetProcess returns the process metadata.
	GetProcess(pid int) *process
	// GetLocalIPs returns the list of local addresses.
	GetLocalIPs() ([]net.IP, error)
}

type ProcessesWatcher struct {
	portProcMap  map[applayer.Transport]map[endpoint]portProcMapping
	localAddrs   []net.IP
	processCache map[int]*process

	// config
	enabled    bool
	procConfig []ProcConfig

	impl processWatcherImpl
}

func (proc *ProcessesWatcher) Init(config ProcsConfig) error {
	return proc.initWithImpl(config, proc)
}

func (proc *ProcessesWatcher) initWithImpl(config ProcsConfig, impl processWatcherImpl) error {
	proc.impl = impl
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
func (proc *ProcessesWatcher) FindProcessesTuple(tuple *common.IPPortTuple, transport applayer.Transport) (procTuple *common.ProcessTuple) {
	procTuple = &common.ProcessTuple{}

	if !proc.enabled {
		return
	}

	if proc.isLocalIP(tuple.SrcIP) {
		if p := proc.findProc(tuple.SrcIP, tuple.SrcPort, transport); p != nil {
			procTuple.Src.PID = p.pid
			procTuple.Src.PPID = p.ppid
			procTuple.Src.Name = p.name
			procTuple.Src.Args = p.args
			procTuple.Src.Exe = p.exe
			procTuple.Src.StartTime = p.startTime
			if logp.IsDebug("procs") {
				logp.Debug("procs", "Found process '%s' (pid=%d) for %s:%d/%s", p.name, p.pid, tuple.SrcIP, tuple.SrcPort, transport)
			}
		}
	}

	if proc.isLocalIP(tuple.DstIP) {
		if p := proc.findProc(tuple.DstIP, tuple.DstPort, transport); p != nil {
			procTuple.Dst.PID = p.pid
			procTuple.Dst.PPID = p.ppid
			procTuple.Dst.Name = p.name
			procTuple.Dst.Args = p.args
			procTuple.Dst.Exe = p.exe
			procTuple.Dst.StartTime = p.startTime
			if logp.IsDebug("procs") {
				logp.Debug("procs", "Found process '%s' (pid=%d) for %s:%d/%s", p.name, p.pid, tuple.DstIP, tuple.DstPort, transport)
			}
		}
	}

	return
}

func (proc *ProcessesWatcher) findProc(address net.IP, port uint16, transport applayer.Transport) *process {
	defer logp.Recover("FindProc exception")

	procMap, ok := proc.portProcMap[transport]
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
		return
	}

	nullAddr := anyIPv4
	if asIPv4 := address.To4(); asIPv4 == nil {
		nullAddr = anyIPv6
	}
	p, found = procMap[endpoint{nullAddr, port}]
	return
}

func (proc *ProcessesWatcher) updateMap(transport applayer.Transport) {
	if logp.HasSelector("procsdetailed") {
		start := time.Now()
		defer func() {
			logp.Debug("procsdetailed", "updateMap() took %v", time.Since(start))
		}()
	}

	endpoints, err := proc.impl.GetLocalPortToPIDMapping(transport)
	if err != nil {
		logp.Err("unable to list local ports: %v", err)
	}

	proc.expireProcessCache()

	for e, pid := range endpoints {
		proc.updateMappingEntry(transport, e, pid)
	}
}

func (proc *ProcessesWatcher) updateMappingEntry(transport applayer.Transport, e endpoint, pid int) {
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
	p := proc.impl.GetProcess(pid)
	if p == nil {
		return nil
	}

	// The packetbeat.procs.monitored*.cmdline_grep allows you to overwrite
	// the process name with an alias.
	for _, match := range proc.procConfig {
		if strings.Contains(strings.Join(p.args, " "), match.CmdlineGrep) {
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

	name := info.Name
	if len(info.Args) > 0 {
		// Workaround the 20 char limit on comm values on Linux.
		name = filepath.Base(info.Args[0])
	}
	return &process{
		pid:        info.PID,
		ppid:       info.PPID,
		name:       name,
		exe:        info.Exe,
		cwd:        info.CWD,
		args:       info.Args,
		startTime:  info.StartTime,
		expiration: time.Now().Add(processCacheExpiration),
	}
}

// GetLocalIPs returns the list of local addresses.
func (proc *ProcessesWatcher) GetLocalIPs() ([]net.IP, error) {
	return common.LocalIPAddrs()
}
