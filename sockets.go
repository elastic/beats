package main

import (
    "bufio"
    "bytes"
    "encoding/csv"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "time"

    "github.com/tsg/fsnotify"
)

type SocketInfo struct {
    Src_ip, Dst_ip     uint32
    Src_port, Dst_port uint16

    Uid   uint16
    Inode int64
}

func hex_to_ip_port(str []byte) (uint32, uint16, error) {
    words := bytes.Split(str, []byte(":"))
    if len(words) < 2 {
        return 0, 0, MsgError("Didn't find ':' as a separator")
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

// Parses the /proc/net/tcp file
func Parse_Proc_Net_Tcp(input io.Reader) ([]*SocketInfo, error) {
    buf := bufio.NewReader(input)

    sockets := []*SocketInfo{}
    var err error = nil
    var line []byte
    for err != io.EOF {
        line, err = buf.ReadBytes('\n')
        if err != nil && err != io.EOF {
            ERR("Error reading /proc/net/tcp: %s", err)
            return nil, err
        }

        words := bytes.Fields(line)
        if len(words) < 10 || bytes.Equal(words[0], []byte("sl")) {
            //DEBUG("Less then 10 words (%d) or starting with 'sl': %s", len(words), words)
            continue
        }

        var sock SocketInfo
        var err_ error

        sock.Src_ip, sock.Src_port, err_ = hex_to_ip_port(words[1])
        if err_ != nil {
            DEBUG("sockets", "Error parsing IP and port: %s", err_)
            continue
        }

        sock.Dst_ip, sock.Dst_port, err_ = hex_to_ip_port(words[2])
        if err_ != nil {
            DEBUG("sockets", "Error parsing IP and port: %s", err_)
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

// Interface to help with testing
type Directory interface {
    Readdirnames(n int) (names []string, err error)
}

func find_Sockets_Of_Process(dir Directory,
    readlink func(name string) (string, error)) (inodes []int, err error) {

    names, err := dir.Readdirnames(0)
    if err != nil {
        return []int{}, MsgError("Readdirnames: %s", err)
    }

    for _, name := range names {
        link, err := readlink(name)
        if err != nil {
            //DEBUG("readlink: %s:", err)
            // Most likely the process has gone away. Break out from here.
            break
        }

        if strings.HasPrefix(link, "socket:[") {
            inode, err := strconv.ParseInt(link[8:len(link)-1], 10, 32)
            if err != nil {
                DEBUG("sockets", "ParseInt: %s:", err)
                continue
            }

            inodes = append(inodes, int(inode))
        }
    }

    return inodes, nil
}

// Given the pid of a process, find the inodes of all the open
// sockets of that process.
func Find_Sockets_Of_Process(pid int) (inodes []int, err error) {
    dir, err := os.Open(fmt.Sprintf("/proc/%d/fd", pid))
    if err != nil {
        return []int{}, MsgError("Open: %s", err)
    }

    return find_Sockets_Of_Process(dir, func(name string) (string, error) {
        return os.Readlink(fmt.Sprintf("/proc/%d/fd/%s", pid, name))
    })
}

type InodePidMap map[int]int

func Generate_InodePidMap() (inode_pid_map InodePidMap, err error) {

    inode_pid_map = InodePidMap{}

    proc, err := os.Open("/proc")
    if err != nil {
        return inode_pid_map, MsgError("Open /proc: %s", err)
    }

    names, err := proc.Readdirnames(0)
    if err != nil {
        return inode_pid_map, MsgError("Readdirnames: %s", err)
    }

    for _, name := range names {
        pid, err := strconv.Atoi(name)
        if err != nil {
            continue
        }

        inodes, err := Find_Sockets_Of_Process(pid)
        if err != nil {
            continue
        }

        for _, inode := range inodes {
            inode_pid_map[inode] = pid
        }
    }

    return
}

type PortPidMap map[uint16]int
type PortCmdlineMap map[uint16]string

func Generate_PortPidMap() (port_pid_map PortPidMap, err error) {
    port_pid_map = PortPidMap{}

    file, err := os.Open("/proc/net/tcp")
    if err != nil {
        return port_pid_map, MsgError("Open: %s", err)
    }

    sockets, err := Parse_Proc_Net_Tcp(file)
    if err != nil {
        return port_pid_map, MsgError("Parse_Proc_Net_Tcp: %s", err)
    }

    inode_pid_map, err := Generate_InodePidMap()
    if err != nil {
        return port_pid_map, MsgError("Generate_InodePidMap: %s", err)
    }

    for _, sock := range sockets {
        pid, exists := inode_pid_map[int(sock.Inode)]
        if exists {
            port_pid_map[sock.Src_port] = pid
        }
    }

    return
}

func Find_CommandLine_By_Port(port_pid_map PortPidMap, port uint16) (cmdline []byte, err error) {

    pid, exists := port_pid_map[port]
    if !exists {
        return []byte{}, MsgError("Port not found")
    }

    file, err := os.Open(fmt.Sprintf("/proc/%d/cmdline", pid))
    if err != nil {
        return []byte{}, MsgError("Open: %s", err)
    }

    cmdline = make([]byte, 500)
    read, err := file.Read(cmdline)
    if err != nil {
        return []byte{}, MsgError("Read: %s", err)
    }

    // Replace all NULL characters with spaces
    for i := 0; i < read; i++ {
        if cmdline[i] == 0 {
            cmdline[i] = byte(' ')
        }
    }

    return cmdline, nil
}

func FindPidsByCmdline(prefix string, processes []string) ([]int, error) {
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

        for _, process := range processes {
            if strings.Index(string(cmdline), process) >= 0 {
                pids = append(pids, pid)
            }
        }
    }

    return pids, nil
}

func ReadSocketLink(path string) (int, error) {

    link, err := os.Readlink(path)
    if err != nil {
        //DEBUG("sockets", "readlink: %s:", err)
        return 0, err
    }

    if !strings.HasPrefix(link, "socket:[") {
        return 0, MsgError("Not a socket FD")
    }

    inode, err := strconv.ParseInt(link[8:len(link)-1], 10, 32)
    if err != nil {
        DEBUG("sockets", "ParseInt: %s:", err)
        return 0, err
    }

    return int(inode), nil
}

// Passed through the channel when a new socket shows up
// in the monitored processes.
type NewSocketCreated struct {
    Inode int
    Pid   int
}

func WatchForSockets(prefix string, pid int, sockChan chan NewSocketCreated) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        ERR("NewWatcher: %s", err)
        return
    }

    dir := filepath.Join(prefix, "/proc", strconv.Itoa(pid), "fd")

    go func() {
        for {
            select {
            case ev := <-watcher.Event:
                DEBUG("sockets", "Got event: %s", ev)
                if ev.IsCreate() {
                    inode, err := ReadSocketLink(ev.Name)
                    if err != nil {
                        DEBUG("sockets", "Reading link: %s", err)
                        continue
                    }
                    sockChan <- NewSocketCreated{Pid: pid, Inode: inode}
                }
            case err := <-watcher.Error:
                WARN("Watch error: %s", err)
            }
        }
    }()

    err = watcher.Watch(dir)
    if err != nil {
        ERR("fswatch: %s", err)
    }

    DEBUG("sockets", "Watching %s", dir)
}

type SocketsType struct {
    port_pid_map     *PortPidMap
    port_cmdline_map PortCmdlineMap
    lock             sync.Mutex
    NewSockChan      chan NewSocketCreated

    // config
    Processes       []string
    ReadFromProc    bool
    ReadFromCsvFile bool
    SleepTime       time.Duration
}

var Sockets SocketsType

func PrintNewSockets(sockChan chan NewSocketCreated) {
    for sockFd := range sockChan {
        DEBUG("sockets", "Pid %d has new socket with inode %d", sockFd.Pid, sockFd.Inode)
    }
}

func (sockets *SocketsType) SetupProcessMonitoring() {
    pids, err := FindPidsByCmdline("", sockets.Processes)
    if err != nil {
        ERR("Couldn't get the PIDs to monitor: %s", err)
        return
    }

    for _, pid := range pids {
        WatchForSockets("", pid, sockets.NewSockChan)
    }

    go PrintNewSockets(sockets.NewSockChan)
}

func (sockets *SocketsType) BuildPortsMap() {

    for /*ever*/ {
        port_pid_map, err := Generate_PortPidMap()
        if err != nil {
            WARN("Generate_PortPidMap: %s", err)
        } else {
            sockets.lock.Lock()
            sockets.port_pid_map = &port_pid_map
            sockets.lock.Unlock()
        }

        time.Sleep(sockets.SleepTime)
    }
}

func (sockets *SocketsType) ReadFromCsv(csv_file string) {
    file, err := os.Open(csv_file)
    if err != nil {
        ERR("Error opening file %s: %s", csv_file, err)
        return
    }

    sockets.port_cmdline_map = PortCmdlineMap{}

    reader := csv.NewReader(file)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            ERR("Broken CSV file %s: %s", csv_file, err)
            return
        }

        if len(record) != 2 {
            ERR("Expected two columns in the CSV file")
            return
        }

        port, err := strconv.ParseUint(record[0], 10, 16)
        if err != nil {
            ERR("Non integer port: %s", port)
            return
        }

        sockets.port_cmdline_map[uint16(port)] = record[1]

    }

    DEBUG("sockets", "port_cmdline_map: %s", sockets.port_cmdline_map)
}

func (sockets *SocketsType) Find_Cmdlines(tuple *IpPortTuple) (cmdline_tuple *CmdlineTuple) {
    cmdline_tuple = &CmdlineTuple{}

    if sockets.ReadFromProc {
        sockets.lock.Lock()
        port_pid_map := sockets.port_pid_map
        sockets.lock.Unlock()

        if port_pid_map == nil {
            return
        }

        cmdline_tuple.Src, _ = Find_CommandLine_By_Port(*port_pid_map, tuple.Src_port)
        cmdline_tuple.Dst, _ = Find_CommandLine_By_Port(*port_pid_map, tuple.Dst_port)

    } else if sockets.ReadFromCsvFile {

        cmdline_tuple.Src = []byte(sockets.port_cmdline_map[tuple.Src_port])
        cmdline_tuple.Dst = []byte(sockets.port_cmdline_map[tuple.Dst_port])
    }

    /*
       if len(cmdline_tuple.Src) > 0 {
           DEBUG("Found command line %s for port %d", cmdline_tuple.Src, tuple.Src_port)
       }

       if len(cmdline_tuple.Dst) > 0 {
           DEBUG("Found command line %s for port %d", cmdline_tuple.Dst, tuple.Dst_port)
       }
    */

    return
}
