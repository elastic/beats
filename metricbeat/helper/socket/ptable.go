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

package socket

import (
	"strconv"
	"strings"
	"unsafe"

	"github.com/joeshaw/multierror"
	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)

// process tools

// Proc contains static process information.
type Proc struct {
	PID        int
	Command    string
	Executable string
	CmdLine    string
	Args       []string
}

// ProcTable contains all of the active processes (if the current user is root).
type ProcTable struct {
	fs         procfs.FS
	procs      map[int]*Proc
	inodes     map[uint32]*Proc
	privileged bool
}

// NewProcTable returns a new ProcTable that reads data from the /proc
// directory by default. An alternative proc filesystem mountpoint can be
// specified through the mountpoint parameter.
func NewProcTable(mountpoint string) (*ProcTable, error) {
	if mountpoint == "" {
		mountpoint = procfs.DefaultMountPoint
	}

	fs, err := procfs.NewFS(mountpoint)
	if err != nil {
		return nil, err
	}

	p := &ProcTable{fs: fs, privileged: isPrivileged(mountpoint)}
	p.Refresh()
	return p, nil
}

// Privileged returns true if the process has enough permissions to read
// sockets of all users
func (t *ProcTable) Privileged() bool {
	return t.privileged
}

// Refresh updates the process table with new processes and removes processes
// that have exited. It collects the PID, command, and socket inode information.
// If running as non-root, only information from the current process will be
// collected.
func (t *ProcTable) Refresh() error {
	var err error
	var procs []procfs.Proc
	if t.privileged {
		procs, err = t.fs.AllProcs()
		if err != nil {
			return err
		}
	} else {
		proc, err := t.fs.Self()
		if err != nil {
			return err
		}
		procs = append(procs, proc)
	}

	var errs multierror.Errors
	inodes := map[uint32]*Proc{}
	cachedProcs := make(map[int]*Proc, len(procs))
	for _, p := range procs {
		proc := t.procs[p.PID]

		// Cache miss.
		if proc == nil {
			proc = &Proc{PID: p.PID}

			if proc.Executable, err = p.Executable(); err != nil {
				errs = append(errs, err)
			}
			if proc.Command, err = p.Comm(); err != nil {
				errs = append(errs, err)
			}
			if cmdline, err := p.CmdLine(); err != nil {
				errs = append(errs, err)
			} else {
				proc.Args = cmdline
				proc.CmdLine = strings.Join(cmdline, " ")
			}
		}
		cachedProcs[proc.PID] = proc

		// Always update map socket inode to Proc.
		socketInodes, err := socketInodes(&p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, inode := range socketInodes {
			inodes[inode] = proc
		}
	}

	t.procs = cachedProcs
	t.inodes = inodes
	return errs.Err()
}

func socketInodes(p *procfs.Proc) ([]uint32, error) {
	fds, err := p.FileDescriptorTargets()
	if err != nil {
		return nil, err
	}

	var inodes []uint32
	for _, fd := range fds {
		if strings.HasPrefix(fd, "socket:[") {
			inode, err := strconv.ParseInt(fd[8:len(fd)-1], 10, 64)
			if err != nil {
				continue
			}

			inodes = append(inodes, uint32(inode))
		}
	}

	return inodes, nil
}

// ProcessBySocketInode returns the Proc associated with the given socket
// inode.
func (t *ProcTable) ProcessBySocketInode(inode uint32) *Proc {
	return t.inodes[inode]
}

const requiredCapabilities = uint64(0x0000000000080004) // ptrace & dac_read_search

// isPrivileged checks if this process has privileges to read sockets
// of all users
func isPrivileged(mountpoint string) bool {
	capabilities := getCapabilities()
	return (capabilities.effective & requiredCapabilities) > 0
}

type capData64 struct {
	effective   uint64
	permitted   uint64
	inheritable uint64
}

type capData32 struct {
	effective   uint32
	permitted   uint32
	inheritable uint32
}

func uint32to64(a, b uint32) uint64 {
	return uint64(a)<<32 | uint64(b)
}

const (
	capabilityVersion1 = 0x19980330 // Version 1, 32-bit capabilities
	capabilityVersion3 = 0x20080522 // Version 3 (replaced V2), 64-bit capabilities
)

func getCapabilities() capData64 {
	header := struct {
		version uint32
		pid     int32
	}{
		version: capabilityVersion3,
		pid:     0, // Self
	}

	// Check compatibility with version 3
	_, _, e := unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), 0, 0)
	if e != 0 {
		header.version = capabilityVersion1
	}

	var data [2]capData32
	_, _, e = unix.Syscall(unix.SYS_CAPGET, uintptr(unsafe.Pointer(&header)), uintptr(unsafe.Pointer(&data)), 0)
	if e != 0 {
		// If this fails, there are invalid arguments
		panic(unix.ErrnoName(e))
	}

	var data64 capData64
	data64.effective = uint32to64(data[1].effective, data[0].effective)
	data64.permitted = uint32to64(data[1].permitted, data[0].permitted)
	data64.inheritable = uint32to64(data[1].inheritable, data[0].inheritable)
	return data64
}
