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

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type SocketInfo struct {
	Src_ip, Dst_ip     uint32
	Src_port, Dst_port uint16

	Uid   uint16
	Inode int64
}

type PortProcMapping struct {
	Port uint16
	Pid  int
	Proc *Process
}

type Process struct {
	Name    string
	Grepper string
	Pids    []int

	proc *ProcessesWatcher

	RefreshPidsTimer <-chan time.Time
}

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
	proc_prefix string
	TestSignals *chan bool
}

type ProcsConfig struct {
	Enabled            bool
	Max_proc_read_freq int
	Monitored          []ProcConfig
	Refresh_pids_freq  int
}

type ProcConfig struct {
	Process      string
	Cmdline_grep string
}

var ProcWatcher ProcessesWatcher

func (proc *ProcessesWatcher) Init(config ProcsConfig) error {

	proc.proc_prefix = ""
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

	if config.Max_proc_read_freq == 0 {
		proc.MaxReadFreq = 10 * time.Millisecond
	} else {
		proc.MaxReadFreq = time.Duration(config.Max_proc_read_freq) *
			time.Millisecond
	}

	if config.Refresh_pids_freq == 0 {
		proc.RefreshPidsFreq = 1 * time.Second
	} else {
		proc.RefreshPidsFreq = time.Duration(config.Refresh_pids_freq) *
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

			grepper := procConfig.Cmdline_grep
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

func NewProcess(proc *ProcessesWatcher, name string, grepper string,
	refreshPidsTimer <-chan time.Time) (*Process, error) {

	p := &Process{Name: name, proc: proc, Grepper: grepper,
		RefreshPidsTimer: refreshPidsTimer}

	// start periodic timer in its own goroutine
	go p.RefreshPids()

	return p, nil
}

func (p *Process) RefreshPids() {
	logp.Debug("procs", "In RefreshPids")
	for _ = range p.RefreshPidsTimer {
		logp.Debug("procs", "In RefreshPids tick")
		var err error
		p.Pids, err = FindPidsByCmdlineGrep(p.proc.proc_prefix, p.Grepper)
		if err != nil {
			logp.Err("Error finding PID files for %s: %s", p.Name, err)
		}
		logp.Debug("procs", "RefreshPids found pids %s for process %s", p.Pids, p.Name)

		if p.proc.TestSignals != nil {
			*p.proc.TestSignals <- true
		}
	}
}

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
			inodes, err := FindSocketsOfPid(proc.proc_prefix, pid)
			if err != nil {
				logp.Err("FindSocketsOfPid: %s", err)
				continue
			}

			for _, inode := range inodes {
				sockInfo, exists := socks_map[inode]
				if exists {
					proc.UpdateMappingEntry(sockInfo.Src_port, pid, p)
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

		sock.Src_ip, sock.Src_port, err_ = hex_to_ip_port(words[1])
		if err_ != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err_)
			continue
		}

		sock.Dst_ip, sock.Dst_port, err_ = hex_to_ip_port(words[2])
		if err_ != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err_)
			continue
		}

		uid, _ := strconv.Atoi(string(words[7]))
		sock.Uid = uint16(uid)
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
