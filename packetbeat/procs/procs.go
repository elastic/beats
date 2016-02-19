package procs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// SocketInfo holds information about a socket connection.
type SocketInfo struct {
	SrcIP, DstIP     uint32
	SrcPort, DstPort uint16

	UID   uint16
	Inode int64
}

// PortProcMapping maps the port information to a specific process ID (PID)
type PortProcMapping struct {
	Port uint16
	Pid  int
	Proc *Process
}

// Process contains information about the process that is being sniffed.
type Process struct {
	Name    string
	Grepper string
	Pids    []int

	proc *ProcessesWatcher

	RefreshPidsTimer <-chan time.Time
}

// ProcessesWatcher contains the information about which processes are being
// watched
type ProcessesWatcher struct {
	PortProcMap   map[uint16]PortProcMapping
	LastMapUpdate time.Time
	Processes     []*Process
	LocalAddrs    []net.IP

	// config
	ReadFromProc    bool
	MaxReadFreq     time.Duration
	RefreshPidsFreq time.Duration

	// test helpers
	procPrefix  string
	TestSignals *chan bool
}

// ProcsConfig contains information about whether a specific process should have
// it's traffic sniffed / monitored and the characteristics of that behavior.
type ProcsConfig struct {
	Enabled         bool
	MaxProcReadFreq int
	Monitored       []ProcConfig
	RefreshPidsFreq int
}

// ProcConfig contains what configuration would be used to identify a process.
type ProcConfig struct {
	Process     string
	CmdlineGrep string
}

// ProcWatcher is an instance of ProcessesWatcher which can be worked with
// globally throughout the procs package.
var ProcWatcher ProcessesWatcher

// Init intializes the ProcessesWatcher with the necessary configuration
// information to run on the machine.
func (proc *ProcessesWatcher) Init(config ProcsConfig) error {
	proc.procPrefix = ""
	proc.PortProcMap = make(map[uint16]PortProcMapping)
	proc.LastMapUpdate = time.Now()

	proc.ReadFromProc = config.Enabled
	if proc.ReadFromProc {
		if runtime.GOOS != "linux" {
			proc.ReadFromProc = false
			logp.Info("Disabled /proc/ reading because not on linux")
		} else {
			logp.Info("Process matching enabled")
		}
	}

	if config.MaxProcReadFreq == 0 {
		proc.MaxReadFreq = 10 * time.Millisecond
	} else {
		proc.MaxReadFreq = time.Duration(config.MaxProcReadFreq) *
			time.Millisecond
	}

	if config.RefreshPidsFreq == 0 {
		proc.RefreshPidsFreq = 1 * time.Second
	} else {
		proc.RefreshPidsFreq = time.Duration(config.RefreshPidsFreq) *
			time.Millisecond
	}

	// Read the local IP addresses
	var err error
	proc.LocalAddrs, err = common.LocalIpAddrs()
	if err != nil {
		logp.Err("Error getting local IP addresses: %s", err)
		proc.LocalAddrs = []net.IP{}
	}

	if proc.ReadFromProc {
		for _, procConfig := range config.Monitored {

			grepper := procConfig.CmdlineGrep
			if len(grepper) == 0 {
				grepper = procConfig.Process
			}

			p, err := NewProcess(proc, procConfig.Process, grepper, time.Tick(proc.RefreshPidsFreq))
			if err != nil {
				logp.Err("NewProcess: %s", err)
			} else {
				proc.Processes = append(proc.Processes, p)
			}
		}
	}

	return nil
}

// NewProcess creates a new Process object representing a specific process on
// the operating system
func NewProcess(proc *ProcessesWatcher, name string, grepper string,
	refreshPidsTimer <-chan time.Time) (*Process, error) {

	p := &Process{Name: name, proc: proc, Grepper: grepper,
		RefreshPidsTimer: refreshPidsTimer}

	// start periodic timer in its own goroutine
	go p.RefreshPids()

	return p, nil
}

// RefreshPids handles refreshing the PID values for the processes which should
// be sniffed / monitored by packetbeat.
func (p *Process) RefreshPids() {
	logp.Debug("procs", "In RefreshPids")
	for range p.RefreshPidsTimer {
		logp.Debug("procs", "In RefreshPids tick")
		var err error
		p.Pids, err = FindPidsByCmdlineGrep(p.proc.procPrefix, p.Grepper)
		if err != nil {
			logp.Err("Error finding PID files for %s: %s", p.Name, err)
		}
		logp.Debug("procs", "RefreshPids found pids %s for process %s", p.Pids, p.Name)

		if p.proc.TestSignals != nil {
			*p.proc.TestSignals <- true
		}
	}
}

// FindPidsByCmdlineGrep returns the pids for the processes that should be
// monitored / sniffed by using the grep information provided during intial
// configuration.
func FindPidsByCmdlineGrep(prefix string, process string) ([]int, error) {
	defer logp.Recover("FindPidsByCmdlineGrep exception")
	pids := []int{}

	proc, err := os.Open(filepath.Join(prefix, "/proc"))
	if err != nil {
		return pids, fmt.Errorf("Open /proc: %s", err)
	}
	defer proc.Close()

	names, err := proc.Readdirnames(0)
	if err != nil {
		return pids, fmt.Errorf("Readdirnames: %s", err)
	}

	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}

		cmdline, err := ioutil.ReadFile(filepath.Join(prefix, "/proc/", name, "cmdline"))
		if err != nil {
			continue
		}

		if strings.Index(string(cmdline), process) >= 0 {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

func (proc *ProcessesWatcher) FindProcessesTuple(tuple *common.IpPortTuple) (proc_tuple *common.CmdlineTuple) {
	proc_tuple = &common.CmdlineTuple{}

	if !proc.ReadFromProc {
		return
	}

	if proc.IsLocalIp(tuple.Src_ip) {
		logp.Debug("procs", "Looking for port %d", tuple.Src_port)
		proc_tuple.Src = []byte(proc.FindProc(tuple.Src_port))
		if len(proc_tuple.Src) > 0 {
			logp.Debug("procs", "Found device %s for port %d", proc_tuple.Src, tuple.Src_port)
		}
	}

	if proc.IsLocalIp(tuple.Dst_ip) {
		logp.Debug("procs", "Looking for port %d", tuple.Dst_port)
		proc_tuple.Dst = []byte(proc.FindProc(tuple.Dst_port))
		if len(proc_tuple.Dst) > 0 {
			logp.Debug("procs", "Found device %s for port %d", proc_tuple.Dst, tuple.Dst_port)
		}
	}

	return
}

func (proc *ProcessesWatcher) FindProc(port uint16) (procname string) {
	procname = ""
	defer logp.Recover("FindProc exception")

	p, exists := proc.PortProcMap[port]
	if exists {
		return p.Proc.Name
	}

	now := time.Now()

	if now.Sub(proc.LastMapUpdate) > proc.MaxReadFreq {
		proc.LastMapUpdate = now
		proc.UpdateMap()

		// try again
		p, exists := proc.PortProcMap[port]
		if exists {
			return p.Proc.Name
		}
	}

	return ""
}

func hex_to_ip_port(str []byte) (uint32, uint16, error) {
	words := bytes.Split(str, []byte(":"))
	if len(words) < 2 {
		return 0, 0, errors.New("Didn't find ':' as a separator")
	}

	ip, err := strconv.ParseInt(string(words[0]), 16, 64)
	if err != nil {
		return 0, 0, err
	}

	port, err := strconv.ParseInt(string(words[1]), 16, 32)
	if err != nil {
		return 0, 0, err
	}

	return uint32(ip), uint16(port), nil
}

func (proc *ProcessesWatcher) UpdateMap() {

	logp.Debug("procs", "UpdateMap()")
	file, err := os.Open("/proc/net/tcp")
	if err != nil {
		logp.Err("Open: %s", err)
		return
	}
	defer file.Close()
	socks, err := Parse_Proc_Net_Tcp(file)
	if err != nil {
		logp.Err("Parse_Proc_Net_Tcp: %s", err)
		return
	}
	socks_map := map[int64]*SocketInfo{}
	for _, s := range socks {
		socks_map[s.Inode] = s
	}

	for _, p := range proc.Processes {
		for _, pid := range p.Pids {
			inodes, err := FindSocketsOfPid(proc.procPrefix, pid)
			if err != nil {
				logp.Err("FindSocketsOfPid: %s", err)
				continue
			}

			for _, inode := range inodes {
				sockInfo, exists := socks_map[inode]
				if exists {
					proc.UpdateMappingEntry(sockInfo.SrcPort, pid, p)
				}
			}

		}
	}

}

// Parses the /proc/net/tcp file
func Parse_Proc_Net_Tcp(input io.Reader) ([]*SocketInfo, error) {
	buf := bufio.NewReader(input)

	sockets := []*SocketInfo{}
	var err error = nil
	var line []byte
	for err != io.EOF {
		line, err = buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			logp.Err("Error reading /proc/net/tcp: %s", err)
			return nil, err
		}

		words := bytes.Fields(line)
		if len(words) < 10 || bytes.Equal(words[0], []byte("sl")) {
			logp.Debug("procs", "Less then 10 words (%d) or starting with 'sl': %s", len(words), words)
			continue
		}

		var sock SocketInfo
		var err_ error

		sock.SrcIP, sock.SrcPort, err_ = hex_to_ip_port(words[1])
		if err_ != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err_)
			continue
		}

		sock.DstIP, sock.DstPort, err_ = hex_to_ip_port(words[2])
		if err_ != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err_)
			continue
		}

		uid, _ := strconv.Atoi(string(words[7]))
		sock.UID = uint16(uid)
		inode, _ := strconv.Atoi(string(words[9]))
		sock.Inode = int64(inode)

		sockets = append(sockets, &sock)
	}
	return sockets, nil
}

func (proc *ProcessesWatcher) UpdateMappingEntry(port uint16, pid int, p *Process) {
	entry := PortProcMapping{Port: port, Pid: pid, Proc: p}

	// Simply overwrite old entries for now.
	// We never expire entries from this map. Since there are 65k possible
	// ports, the size of the dict can be max 1.5 MB, which we consider
	// reasonable.
	proc.PortProcMap[port] = entry

	logp.Debug("procsdetailed", "UpdateMappingEntry(): port=%d pid=%d", port, p.Name)
}

func FindSocketsOfPid(prefix string, pid int) (inodes []int64, err error) {

	dirname := filepath.Join(prefix, "/proc", strconv.Itoa(pid), "fd")
	procfs, err := os.Open(dirname)
	if err != nil {
		return []int64{}, fmt.Errorf("Open: %s", err)
	}
	defer procfs.Close()
	names, err := procfs.Readdirnames(0)
	if err != nil {
		return []int64{}, fmt.Errorf("Readdirnames: %s", err)
	}

	for _, name := range names {
		link, err := os.Readlink(filepath.Join(dirname, name))
		if err != nil {
			logp.Debug("procs", "Readlink %s: %s", name, err)
			continue
		}

		if strings.HasPrefix(link, "socket:[") {
			inode, err := strconv.ParseInt(link[8:len(link)-1], 10, 64)
			if err != nil {
				logp.Debug("procs", "ParseInt: %s:", err)
				continue
			}

			inodes = append(inodes, int64(inode))
		}
	}

	return inodes, nil
}

func (proc *ProcessesWatcher) IsLocalIp(ip net.IP) bool {

	if ip.IsLoopback() {
		return true
	}

	for _, addr := range proc.LocalAddrs {
		if ip.Equal(addr) {
			return true
		}
	}

	return false
}
