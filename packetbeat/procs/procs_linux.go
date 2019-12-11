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

// +build linux

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

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	"github.com/elastic/gosigar"
)

type socketInfo struct {
	srcIP, dstIP     net.IP
	srcPort, dstPort uint16

	uid   uint32
	inode uint64
}

var procFiles = map[applayer.Transport]struct {
	ipv4, ipv6 string
}{
	applayer.TransportUDP: {"/proc/net/udp", "/proc/net/udp6"},
	applayer.TransportTCP: {"/proc/net/tcp", "/proc/net/tcp6"},
}

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
	if err != nil {
		logp.Err("GetLocalPortToPIDMapping: parsing '%s': %s", sourceFiles.ipv6, err)
		return nil, err
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
			logp.Err("FindSocketsOfPid: %s", err)
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

func socketsFromProc(filename string, ipv6 bool) ([]*socketInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseProcNetProto(file, ipv6)
}

// Parses the /proc/net/(tcp|udp)6? file
func parseProcNetProto(input io.Reader, ipv6 bool) ([]*socketInfo, error) {
	buf := bufio.NewReader(input)

	sockets := []*socketInfo{}
	var err error
	var line []byte
	for err != io.EOF {
		line, err = buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			logp.Err("Error reading proc net file: %s", err)
			return nil, err
		}
		words := bytes.Fields(line)
		// Ignore empty lines and the header
		if len(words) == 0 || bytes.Equal(words[0], []byte("sl")) {
			continue
		}
		if len(words) < 10 {
			logp.Debug("procs", "Less than 10 words (%d) or starting with 'sl': %s", len(words), words)
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
