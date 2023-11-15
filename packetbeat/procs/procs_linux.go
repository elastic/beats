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

//go:build linux

package procs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-sysinfo/types"
	"github.com/elastic/gosigar"
)

// procName makes a best effort attempt to get a good name for the process. It
// uses /proc/<pid>/comm if it is less than 16 bytes long (TASK_COMM_LEN) and
// otherwise uses argv[0] if it is available, but falling back to the comm string
// if it is not.
func procName(info types.ProcessInfo) string {
	// We cannot know that a 16 byte string is not truncated,
	// so assume the worst and get the first argument if we
	// are at TASK_COMM_LEN and the arg is available.
	if len(info.Name) < 16 || len(info.Args) == 0 {
		return info.Name
	}
	return filepath.Base(info.Args[0])
}

var warnIPv6Once sync.Once

// GetLocalPortToPIDMapping returns the list of local port numbers and the PID
// that owns them.
func (proc *ProcessesWatcher) GetLocalPortToPIDMapping(transport applayer.Transport) (ports map[endpoint]int, err error) {
	sourceFiles, ok := procFiles[transport]
	if !ok {
		return nil, fmt.Errorf("unsupported transport protocol id: %d", transport)
	}
	var pids gosigar.ProcList
	if err = pids.Get(); err != nil {
		return nil, err
	}
	logp.Debug("procs", "getLocalPortsToPIDs()")
	ipv4socks, err := socketsFromProc(sourceFiles.ipv4, false)
	if err != nil {
		logp.Err("GetLocalPortToPIDMapping: parsing '%s': %s", sourceFiles.ipv4, err)
		return nil, err
	}

	ipv6socks, err := socketsFromProc(sourceFiles.ipv6, true)
	// Ignore the error when /proc/net/tcp6 doesn't exists (ipv6 disabled).
	if err != nil {
		if os.IsNotExist(err) {
			warnIPv6Once.Do(func() {
				logp.Warn("No IPv6 socket info reported by the kernel. Process monitor won't enrich IPv6 events")
			})
		} else {
			logp.Err("GetLocalPortToPIDMapping: parsing '%s': %s", sourceFiles.ipv6, err)
			return nil, err
		}
	}
	socksMap := map[uint64]*socketInfo{}
	for _, s := range ipv4socks {
		socksMap[s.inode] = s
	}
	for _, s := range ipv6socks {
		socksMap[s.inode] = s
	}

	ports = make(map[endpoint]int)
	for _, pid := range pids.List {
		inodes, err := findSocketsOfPid("", pid)
		if err != nil {
			if os.IsNotExist(err) {
				logp.Info("FindSocketsOfPid: %s", err)
			} else {
				logp.Err("FindSocketsOfPid: %s", err)
			}
			continue
		}

		for _, inode := range inodes {
			if sockInfo, exists := socksMap[inode]; exists {
				ports[endpoint{address: sockInfo.srcIP.String(), port: sockInfo.srcPort}] = pid
			}
		}
	}

	return ports, nil
}

var procFiles = map[applayer.Transport]struct {
	ipv4, ipv6 string
}{
	applayer.TransportUDP: {"/proc/net/udp", "/proc/net/udp6"},
	applayer.TransportTCP: {"/proc/net/tcp", "/proc/net/tcp6"},
}

func findSocketsOfPid(prefix string, pid int) (inodes []uint64, err error) {
	dirname := filepath.Join(prefix, "/proc", strconv.Itoa(pid), "fd")
	procfs, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	defer procfs.Close()
	names, err := procfs.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	for _, name := range names {
		link, err := os.Readlink(filepath.Join(dirname, name))
		if err != nil {
			logp.Debug("procs", err.Error())
			continue
		}

		if strings.HasPrefix(link, "socket:[") {
			inode, err := strconv.ParseInt(link[8:len(link)-1], 10, 64)
			if err != nil {
				logp.Debug("procs", err.Error())
				continue
			}

			inodes = append(inodes, uint64(inode))
		}
	}

	return inodes, nil
}

// socketInfo hold details for network sockets obtained from /proc/net.
type socketInfo struct {
	srcIP, dstIP     net.IP
	srcPort, dstPort uint16

	uid   uint32 // uid is the effective UID of the process that created the socket.
	inode uint64 // inode is the inode of the file corresponding to the socket.
}

// socketsFromProc returns the socket information held in the the /proc/net file
// at path.
func socketsFromProc(path string, ipv6 bool) ([]*socketInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseProcNetProto(file, ipv6)
}

// Parses the /proc/net/(tcp|udp)6? file
func parseProcNetProto(input io.Reader, ipv6 bool) ([]*socketInfo, error) {
	var (
		sockets []*socketInfo
		err     error
	)
	sc := bufio.NewScanner(input)
	for sc.Scan() {
		words := bytes.Fields(sc.Bytes())
		// Ignore empty lines and the header
		if len(words) == 0 || bytes.Equal(words[0], []byte("sl")) {
			continue
		}
		if len(words) < 10 {
			logp.Debug("procs", "Less than 10 words (%d) or starting with 'sl': %s", len(words), words)
			continue
		}

		var sock socketInfo
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
	err = sc.Err()
	if err != nil {
		logp.Err("Error reading proc net file: %s", err)
		return nil, err
	}
	return sockets, nil
}

func hexToIPPort(str []byte, ipv6 bool) (net.IP, uint16, error) {
	words := bytes.SplitN(str, []byte(":"), 2) // Use bytes.Cut when it becomes available.
	if len(words) < 2 {
		return nil, 0, errors.New("could not find port separator")
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

func hexToIP(word string, ipv6 bool) (net.IP, error) {
	if ipv6 {
		return hexToIpv6(word)
	}
	return hexToIpv4(word)
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
