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

type socketInfo struct {
	srcIP, dstIP     net.IP
	srcPort, dstPort uint16

	uid   uint32
	inode uint64
}

type portProcMapping struct {
	port uint16
	pid  int
	proc *process
}

type process struct {
	name    string
	grepper string
	pids    []int

	proc *ProcessesWatcher

	refreshPidsTimer <-chan time.Time
}

type ProcessesWatcher struct {
	portProcMap   map[uint16]portProcMapping
	lastMapUpdate time.Time
	processes     []*process
	localAddrs    []net.IP

	// config
	readFromProc    bool
	maxReadFreq     time.Duration
	refreshPidsFreq time.Duration

	// test helpers
	procPrefix  string
	testSignals *chan bool
}

var ProcWatcher ProcessesWatcher

func (proc *ProcessesWatcher) Init(config ProcsConfig) error {
	proc.procPrefix = ""
	proc.portProcMap = make(map[uint16]portProcMapping)
	proc.lastMapUpdate = time.Now()

	proc.readFromProc = config.Enabled
	if proc.readFromProc {
		if runtime.GOOS != "linux" {
			proc.readFromProc = false
			logp.Info("Disabled /proc/ reading because not on linux")
		} else {
			logp.Info("Process matching enabled")
		}
	} else {
		logp.Info("Process matching disabled")
	}

	if config.MaxProcReadFreq == 0 {
		proc.maxReadFreq = 10 * time.Millisecond
	} else {
		proc.maxReadFreq = config.MaxProcReadFreq
	}

	if config.RefreshPidsFreq == 0 {
		proc.refreshPidsFreq = 1 * time.Second
	} else {
		proc.refreshPidsFreq = config.RefreshPidsFreq
	}

	// Read the local IP addresses
	var err error
	proc.localAddrs, err = common.LocalIPAddrs()
	if err != nil {
		logp.Err("Error getting local IP addresses: %s", err)
		proc.localAddrs = []net.IP{}
	}

	if proc.readFromProc {
		for _, procConfig := range config.Monitored {

			grepper := procConfig.CmdlineGrep
			if len(grepper) == 0 {
				grepper = procConfig.Process
			}

			p, err := newProcess(proc, procConfig.Process, grepper, time.Tick(proc.refreshPidsFreq))
			if err != nil {
				logp.Err("NewProcess: %s", err)
			} else {
				proc.processes = append(proc.processes, p)
			}
		}
	}

	return nil
}

func newProcess(proc *ProcessesWatcher, name string, grepper string,
	refreshPidsTimer <-chan time.Time) (*process, error) {

	p := &process{name: name, proc: proc, grepper: grepper,
		refreshPidsTimer: refreshPidsTimer}

	// start periodic timer in its own goroutine
	go p.refreshPids()

	return p, nil
}

func (p *process) refreshPids() {
	logp.Debug("procs", "In RefreshPids")
	for range p.refreshPidsTimer {
		logp.Debug("procs", "In RefreshPids tick")
		var err error
		p.pids, err = findPidsByCmdlineGrep(p.proc.procPrefix, p.grepper)
		if err != nil {
			logp.Err("Error finding PID files for %s: %s", p.name, err)
		}
		logp.Debug("procs", "RefreshPids found pids %s for process %s", p.pids, p.name)

		if p.proc.testSignals != nil {
			*p.proc.testSignals <- true
		}
	}
}

func findPidsByCmdlineGrep(prefix string, process string) ([]int, error) {
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

		if strings.Contains(string(cmdline), process) {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}

func (proc *ProcessesWatcher) FindProcessesTuple(tuple *common.IPPortTuple) (procTuple *common.CmdlineTuple) {
	procTuple = &common.CmdlineTuple{}

	if !proc.readFromProc {
		return
	}

	if proc.isLocalIP(tuple.SrcIP) {
		logp.Debug("procs", "Looking for port %d", tuple.SrcPort)
		procTuple.Src = []byte(proc.findProc(tuple.SrcPort))
		if len(procTuple.Src) > 0 {
			logp.Debug("procs", "Found device %s for port %d", procTuple.Src, tuple.SrcPort)
		}
	}

	if proc.isLocalIP(tuple.DstIP) {
		logp.Debug("procs", "Looking for port %d", tuple.DstPort)
		procTuple.Dst = []byte(proc.findProc(tuple.DstPort))
		if len(procTuple.Dst) > 0 {
			logp.Debug("procs", "Found device %s for port %d", procTuple.Dst, tuple.DstPort)
		}
	}

	return
}

func (proc *ProcessesWatcher) findProc(port uint16) (procname string) {
	procname = ""
	defer logp.Recover("FindProc exception")

	p, exists := proc.portProcMap[port]
	if exists {
		return p.proc.name
	}

	now := time.Now()

	if now.Sub(proc.lastMapUpdate) > proc.maxReadFreq {
		proc.lastMapUpdate = now
		proc.updateMap()

		// try again
		p, exists := proc.portProcMap[port]
		if exists {
			return p.proc.name
		}
	}

	return ""
}

func hexToIpv4(word string) (net.IP, error) {
	ip, err := strconv.ParseInt(word, 16, 64)
	if err != nil {
		return nil, err
	}
	return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24)), nil
}

func hexToIpv6(word string) (net.IP, error) {
	p := make(net.IP, net.IPv6len)
	for i := 0; i < 4; i++ {
		part, err := strconv.ParseUint(word[i*8:(i+1)*8], 16, 32)
		if err != nil {
			return nil, err
		}
		p[i*4] = byte(part)
		p[i*4+1] = byte(part >> 8)
		p[i*4+2] = byte(part >> 16)
		p[i*4+3] = byte(part >> 24)
	}
	return p, nil
}

func hexToIP(word string, ipv6 bool) (net.IP, error) {
	if ipv6 {
		return hexToIpv6(word)
	}
	return hexToIpv4(word)
}

func hexToIPPort(str []byte, ipv6 bool) (net.IP, uint16, error) {
	words := bytes.Split(str, []byte(":"))
	if len(words) < 2 {
		return nil, 0, errors.New("Didn't find ':' as a separator")
	}

	ip, err := hexToIP(string(words[0]), ipv6)
	if err != nil {
		return nil, 0, err
	}

	port, err := strconv.ParseInt(string(words[1]), 16, 32)
	if err != nil {
		return nil, 0, err
	}

	return ip, uint16(port), nil
}

func (proc *ProcessesWatcher) updateMap() {
	logp.Debug("procs", "UpdateMap()")
	ipv4socks, err := socketsFromProc("/proc/net/tcp", false)
	if err != nil {
		logp.Err("Parse_Proc_Net_Tcp: %s", err)
		return
	}
	ipv6socks, err := socketsFromProc("/proc/net/tcp6", true)
	if err != nil {
		logp.Err("Parse_Proc_Net_Tcp ipv6: %s", err)
		return
	}
	socksMap := map[uint64]*socketInfo{}
	for _, s := range ipv4socks {
		socksMap[s.inode] = s
	}
	for _, s := range ipv6socks {
		socksMap[s.inode] = s
	}

	for _, p := range proc.processes {
		for _, pid := range p.pids {
			inodes, err := findSocketsOfPid(proc.procPrefix, pid)
			if err != nil {
				logp.Err("FindSocketsOfPid: %s", err)
				continue
			}

			for _, inode := range inodes {
				sockInfo, exists := socksMap[inode]
				if exists {
					proc.updateMappingEntry(sockInfo.srcPort, pid, p)
				}
			}

		}
	}
}

func socketsFromProc(filename string, ipv6 bool) ([]*socketInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseProcNetTCP(file, ipv6)
}

// Parses the /proc/net/tcp file
func parseProcNetTCP(input io.Reader, ipv6 bool) ([]*socketInfo, error) {
	buf := bufio.NewReader(input)

	sockets := []*socketInfo{}
	var err error
	var line []byte
	for err != io.EOF {
		line, err = buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			logp.Err("Error reading proc net tcp file: %s", err)
			return nil, err
		}
		words := bytes.Fields(line)
		if len(words) < 10 || bytes.Equal(words[0], []byte("sl")) {
			logp.Debug("procs", "Less then 10 words (%d) or starting with 'sl': %s", len(words), words)
			continue
		}

		var sock socketInfo
		var err error

		sock.srcIP, sock.srcPort, err = hexToIPPort(words[1], ipv6)
		if err != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err)
			continue
		}

		sock.dstIP, sock.dstPort, err = hexToIPPort(words[2], ipv6)
		if err != nil {
			logp.Debug("procs", "Error parsing IP and port: %s", err)
			continue
		}

		uid, _ := strconv.Atoi(string(words[7]))
		sock.uid = uint32(uid)
		inode, _ := strconv.Atoi(string(words[9]))
		sock.inode = uint64(inode)

		sockets = append(sockets, &sock)
	}
	return sockets, nil
}

func (proc *ProcessesWatcher) updateMappingEntry(port uint16, pid int, p *process) {
	entry := portProcMapping{port: port, pid: pid, proc: p}

	// Simply overwrite old entries for now.
	// We never expire entries from this map. Since there are 65k possible
	// ports, the size of the dict can be max 1.5 MB, which we consider
	// reasonable.
	proc.portProcMap[port] = entry

	logp.Debug("procsdetailed", "UpdateMappingEntry(): port=%d pid=%d", port, p.name)
}

func findSocketsOfPid(prefix string, pid int) (inodes []uint64, err error) {
	dirname := filepath.Join(prefix, "/proc", strconv.Itoa(pid), "fd")
	procfs, err := os.Open(dirname)
	if err != nil {
		return []uint64{}, fmt.Errorf("Open: %s", err)
	}
	defer procfs.Close()
	names, err := procfs.Readdirnames(0)
	if err != nil {
		return []uint64{}, fmt.Errorf("Readdirnames: %s", err)
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

			inodes = append(inodes, uint64(inode))
		}
	}

	return inodes, nil
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
