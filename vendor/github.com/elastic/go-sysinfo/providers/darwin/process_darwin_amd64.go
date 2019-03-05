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

// +build darwin,amd64,cgo

package darwin

// #cgo LDFLAGS:-lproc
// #include <sys/sysctl.h>
// #include <libproc.h>
import "C"

import (
	"bytes"
	"encoding/binary"
	"os"
	"strconv"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

//go:generate sh -c "go tool cgo -godefs defs_darwin.go > ztypes_darwin_amd64.go"

func (s darwinSystem) Processes() ([]types.Process, error) {
	n, err := C.proc_listallpids(nil, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting process count from proc_listallpids (n = %v)", n)
	} else if n <= 0 {
		return nil, errors.Errorf("proc_listallpids returned %v", n)
	}

	var pid C.int
	bufsize := n * C.int(unsafe.Sizeof(pid))
	buf := make([]byte, bufsize)
	n, err = C.proc_listallpids(unsafe.Pointer(&buf[0]), bufsize)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting processes from proc_listallpids (n = %v)", n)
	} else if n <= 0 {
		return nil, errors.Errorf("proc_listallpids returned %v", n)
	}

	bbuf := bytes.NewBuffer(buf)
	processes := make([]types.Process, 0, n)
	for i := 0; i < int(n); i++ {
		err = binary.Read(bbuf, binary.LittleEndian, &pid)
		if err != nil {
			return nil, errors.Wrap(err, "error reading binary list of PIDs")
		}

		if pid == 0 {
			continue
		}

		processes = append(processes, &process{pid: int(pid)})
	}
	return processes, nil
}

func (s darwinSystem) Process(pid int) (types.Process, error) {
	p := process{pid: pid}

	return &p, nil
}

func (s darwinSystem) Self() (types.Process, error) {
	return s.Process(os.Getpid())
}

type process struct {
	pid  int
	cwd  string
	exe  string
	args []string
	env  map[string]string
}

func (p *process) PID() int {
	return p.pid
}

func (p *process) Info() (types.ProcessInfo, error) {
	var task procTaskAllInfo
	if err := getProcTaskAllInfo(p.pid, &task); err != nil {
		return types.ProcessInfo{}, err
	}

	var vnode procVnodePathInfo
	if err := getProcVnodePathInfo(p.pid, &vnode); err != nil {
		return types.ProcessInfo{}, err
	}

	if err := kern_procargs(p.pid, p); err != nil {
		return types.ProcessInfo{}, err
	}

	return types.ProcessInfo{
		Name: int8SliceToString(task.Pbsd.Pbi_name[:]),
		PID:  p.pid,
		PPID: int(task.Pbsd.Pbi_ppid),
		CWD:  int8SliceToString(vnode.Cdir.Path[:]),
		Exe:  p.exe,
		Args: p.args,
		StartTime: time.Unix(int64(task.Pbsd.Pbi_start_tvsec),
			int64(task.Pbsd.Pbi_start_tvusec)*int64(time.Microsecond)),
	}, nil
}

func (p *process) User() (types.UserInfo, error) {
	var task procTaskAllInfo
	if err := getProcTaskAllInfo(p.pid, &task); err != nil {
		return types.UserInfo{}, err
	}

	return types.UserInfo{
		UID:  strconv.Itoa(int(task.Pbsd.Pbi_ruid)),
		EUID: strconv.Itoa(int(task.Pbsd.Pbi_uid)),
		SUID: strconv.Itoa(int(task.Pbsd.Pbi_svuid)),
		GID:  strconv.Itoa(int(task.Pbsd.Pbi_rgid)),
		EGID: strconv.Itoa(int(task.Pbsd.Pbi_gid)),
		SGID: strconv.Itoa(int(task.Pbsd.Pbi_svgid)),
	}, nil
}

func (p *process) Environment() (map[string]string, error) {
	return p.env, nil
}

func (p *process) CPUTime() (types.CPUTimes, error) {
	var task procTaskAllInfo
	if err := getProcTaskAllInfo(p.pid, &task); err != nil {
		return types.CPUTimes{}, err
	}
	return types.CPUTimes{
		User:   time.Duration(task.Ptinfo.Total_user),
		System: time.Duration(task.Ptinfo.Total_system),
	}, nil
}

func (p *process) Memory() (types.MemoryInfo, error) {
	var task procTaskAllInfo
	if err := getProcTaskAllInfo(p.pid, &task); err != nil {
		return types.MemoryInfo{}, err
	}
	return types.MemoryInfo{
		Virtual:  task.Ptinfo.Virtual_size,
		Resident: task.Ptinfo.Resident_size,
		Metrics: map[string]uint64{
			"page_ins":    uint64(task.Ptinfo.Pageins),
			"page_faults": uint64(task.Ptinfo.Faults),
		},
	}, nil
}

func getProcTaskAllInfo(pid int, info *procTaskAllInfo) error {
	size := C.int(unsafe.Sizeof(*info))
	ptr := unsafe.Pointer(info)

	n, err := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKALLINFO, 0, ptr, size)
	if err != nil {
		return err
	} else if n != size {
		return errors.New("failed to read process info with proc_pidinfo")
	}

	return nil
}

func getProcVnodePathInfo(pid int, info *procVnodePathInfo) error {
	size := C.int(unsafe.Sizeof(*info))
	ptr := unsafe.Pointer(info)

	n := C.proc_pidinfo(C.int(pid), C.PROC_PIDVNODEPATHINFO, 0, ptr, size)
	if n != size {
		return errors.New("failed to read vnode info with proc_pidinfo")
	}

	return nil
}

var nullTerminator = []byte{0}

// wrapper around sysctl KERN_PROCARGS2
// callbacks params are optional,
// up to the caller as to which pieces of data they want
func kern_procargs(pid int, p *process) error {
	mib := []C.int{C.CTL_KERN, C.KERN_PROCARGS2, C.int(pid)}
	var data []byte
	if err := sysctl(mib, &data); err != nil {
		return nil
	}
	buf := bytes.NewBuffer(data)

	// argc
	var argc int32
	if err := binary.Read(buf, binary.LittleEndian, &argc); err != nil {
		return err
	}

	// exe
	lines := bytes.Split(buf.Bytes(), nullTerminator)
	p.exe = string(lines[0])
	lines = lines[1:]

	// skip nulls
	for len(lines) > 0 {
		if len(lines[0]) == 0 {
			lines = lines[1:]
			continue
		}
		break
	}

	// args
	for i := 0; i < int(argc); i++ {
		p.args = append(p.args, string(lines[0]))
		lines = lines[1:]
	}

	// env vars
	env := make(map[string]string, len(lines))
	for _, l := range lines {
		if len(l) == 0 {
			break
		}
		parts := bytes.SplitN(l, []byte{'='}, 2)
		if len(parts) != 2 {
			return errors.New("failed to parse")
		}
		key := string(parts[0])
		value := string(parts[1])
		env[key] = value
	}
	p.env = env

	return nil
}

func int8SliceToString(s []int8) string {
	buf := bytes.NewBuffer(make([]byte, len(s)))
	buf.Reset()

	for _, b := range s {
		if b == 0 {
			break
		}
		buf.WriteByte(byte(b))
	}
	return buf.String()
}
