package main

import (
    "io/ioutil"
    "net"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)

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
    TestSignals chan bool
}

var procWatcher ProcessesWatcher

// Config
type tomlProcs struct {
    Dont_read_from_proc bool
    Max_proc_read_freq  int
    Monitored           map[string]tomlProc
    Refresh_pids_freq   int
}

type tomlProc struct {
    Cmdline_grep string
}

func (proc *ProcessesWatcher) Init(config *tomlProcs) error {

    proc.proc_prefix = ""
    proc.PortProcMap = make(map[uint16]PortProcMapping)
    proc.LastMapUpdate = time.Now()

    proc.ReadFromProc = !config.Dont_read_from_proc

    if config.Max_proc_read_freq == 0 {
        proc.MaxReadFreq = 10 * time.Millisecond
    } else {
        proc.MaxReadFreq = time.Duration(config.Max_proc_read_freq) * time.Millisecond
    }

    if config.Refresh_pids_freq == 0 {
        proc.RefreshPidsFreq = 1 * time.Second
    } else {
        proc.RefreshPidsFreq = time.Duration(config.Refresh_pids_freq) * time.Millisecond
    }

    // Read the local IP addresses
    proc.LocalAddrs = []net.IP{}
    addrs, err := net.InterfaceAddrs()
    if err == nil {
        for _, addr := range addrs {
            // a bit wtf'ish.. Don't know how to do this otherwise
            DEBUG("procaddrs", "Addr: %s", addr.String())
            ip, _, err := net.ParseCIDR(addr.String())
            if err == nil && ip != nil {
                proc.LocalAddrs = append(proc.LocalAddrs, ip)
            }
        }
    } else {
        ERR("InterfaceAddrs: %s", err)
    }
    INFO("Local IP addresses are: %s", proc.LocalAddrs)

    if proc.ReadFromProc {
        for pstr, procConfig := range config.Monitored {

            grepper := procConfig.Cmdline_grep
            if len(grepper) == 0 {
                grepper = pstr
            }

            p, err := NewProcess(proc, pstr, grepper, time.Tick(proc.RefreshPidsFreq))
            if err != nil {
                ERR("NewProcess: %s", err)
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
    DEBUG("procs", "In RefreshPids")
    for _ = range p.RefreshPidsTimer {
        DEBUG("procs", "In RefreshPids tick")
        var err error
        p.Pids, err = FindPidsByCmdlineGrep(p.proc.proc_prefix, p.Grepper)
        if err != nil {
            ERR("Error finding PID files for %s: %s", p.Name, err)
        }
        DEBUG("procs", "RefreshPids found pids %s for process %s", p.Pids, p.Name)

        p.proc.TestSignals <- true
    }
}

func FindPidsByCmdlineGrep(prefix string, process string) ([]int, error) {
    defer RECOVER("FindPidsByCmdlineGrep exception")
    pids := []int{}

    proc, err := os.Open(filepath.Join(prefix, "/proc"))
    if err != nil {
        return pids, MsgError("Open /proc: %s", err)
    }

    names, err := proc.Readdirnames(0)
    if err != nil {
        return pids, MsgError("Readdirnames: %s", err)
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

func (proc *ProcessesWatcher) FindProcessesTuple(tuple *IpPortTuple) (proc_tuple *CmdlineTuple) {
    proc_tuple = &CmdlineTuple{}

    if !proc.ReadFromProc {
        return
    }

    if proc.IsLocalIp(tuple.Src_ip) {
        DEBUG("procs", "Looking for port %d", tuple.Src_port)
        proc_tuple.Src = []byte(proc.FindProc(tuple.Src_port))
        if len(proc_tuple.Src) > 0 {
            DEBUG("procs", "Found device %s for port %d", proc_tuple.Src, tuple.Src_port)
        }
    }

    if proc.IsLocalIp(tuple.Dst_ip) {
        DEBUG("procs", "Looking for port %d", tuple.Dst_port)
        proc_tuple.Dst = []byte(proc.FindProc(tuple.Dst_port))
        if len(proc_tuple.Dst) > 0 {
            DEBUG("procs", "Found device %s for port %d", proc_tuple.Dst, tuple.Dst_port)
        }
    }

    return
}

func (proc *ProcessesWatcher) FindProc(port uint16) (procname string) {
    procname = ""
    defer RECOVER("FindProc exception")

    p, exists := proc.PortProcMap[port]
    if exists {
        return p.Proc.Name
    }

    if time.Now().Sub(proc.LastMapUpdate) > proc.MaxReadFreq {
        proc.UpdateMap()

        // try again
        p, exists := proc.PortProcMap[port]
        if exists {
            return p.Proc.Name
        }
    }

    return ""
}

func (proc *ProcessesWatcher) UpdateMap() {

    DEBUG("procs", "UpdateMap()")
    file, err := os.Open("/proc/net/tcp")
    if err != nil {
        ERR("Open: %s", err)
        return
    }
    socks, err := Parse_Proc_Net_Tcp(file)
    if err != nil {
        ERR("Parse_Proc_Net_Tcp: %s", err)
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
                ERR("FindSocketsOfPid: %s", err)
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

func (proc *ProcessesWatcher) UpdateMappingEntry(port uint16, pid int, p *Process) {
    entry := PortProcMapping{Port: port, Pid: pid, Proc: p}

    // Simply overwrite old entries for now.
    // We never expire entries from this map. Since there are 65k possible
    // ports, the size of the dict can be max 1.5 MB, which we consider
    // reasonable.
    proc.PortProcMap[port] = entry

    DEBUG("procsdetailed", "UpdateMappingEntry(): port=%d pid=%d", port, p.Name)
}

func FindSocketsOfPid(prefix string, pid int) (inodes []int64, err error) {

    dirname := filepath.Join(prefix, "/proc", strconv.Itoa(pid), "fd")
    procfs, err := os.Open(dirname)
    if err != nil {
        return []int64{}, MsgError("Open: %s", err)
    }
    names, err := procfs.Readdirnames(0)
    if err != nil {
        return []int64{}, MsgError("Readdirnames: %s", err)
    }

    for _, name := range names {
        link, err := os.Readlink(filepath.Join(dirname, name))
        if err != nil {
            DEBUG("procs", "Readlink %s: %s", name, err)
            continue
        }

        if strings.HasPrefix(link, "socket:[") {
            inode, err := strconv.ParseInt(link[8:len(link)-1], 10, 64)
            if err != nil {
                DEBUG("procs", "ParseInt: %s:", err)
                continue
            }

            inodes = append(inodes, int64(inode))
        }
    }

    return inodes, nil
}

func (proc *ProcessesWatcher) IsLocalIp(ip uint32) bool {
    Ip := net.IPv4(uint8(ip>>24), uint8(ip>>16), uint8(ip>>8), uint8(ip))

    if Ip.IsLoopback() {
        return true
    }

    for _, addr := range proc.LocalAddrs {
        if Ip.Equal(addr) {
            return true
        }
    }

    return false
}
