// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
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
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

//go:generate sh -c "go tool cgo -godefs defs_darwin.go > ztypes_darwin_amd64.go"

func (s darwinSystem) Processes() ([]types.Process, error) {
	return nil, nil
}

func (s darwinSystem) Process(pid int) (types.Process, error) {
	p := process{pid: pid}

	return &p, nil
}

func (s darwinSystem) Self() (types.Process, error) {
	return s.Process(os.Getpid())
}

type process struct {
	pid   int
	cwd   string
	exe   string
	args  []string
	env   map[string]string
	task  procTaskAllInfo
	vnode procVnodePathInfo
}

func (p *process) Info() (types.ProcessInfo, error) {
	if err := getProcTaskAllInfo(p.pid, &p.task); err != nil {
		return types.ProcessInfo{}, err
	}

	if err := getProcVnodePathInfo(p.pid, &p.vnode); err != nil {
		return types.ProcessInfo{}, err
	}

	if err := kern_procargs(p.pid, p); err != nil {
		return types.ProcessInfo{}, err
	}

	return types.ProcessInfo{
		Name: int8SliceToString(p.task.Pbsd.Pbi_name[:]),
		PID:  p.pid,
		PPID: int(p.task.Pbsd.Pbi_ppid),
		CWD:  int8SliceToString(p.vnode.Cdir.Path[:]),
		Exe:  p.exe,
		Args: p.args,
		StartTime: time.Unix(int64(p.task.Pbsd.Pbi_start_tvsec),
			int64(p.task.Pbsd.Pbi_start_tvusec)*int64(time.Microsecond)),
	}, nil
}

func (p *process) Environment() (map[string]string, error) {
	return p.env, nil
}

func (p *process) CPUTime() types.CPUTimes {
	return types.CPUTimes{
		Timestamp: time.Now(),
		User:      time.Duration(p.task.Ptinfo.Total_user),
		System:    time.Duration(p.task.Ptinfo.Total_system),
	}
}

func (p *process) Memory() types.MemoryInfo {
	return types.MemoryInfo{
		Timestamp: time.Now(),
		Virtual:   p.task.Ptinfo.Virtual_size,
		Resident:  p.task.Ptinfo.Resident_size,
		Metrics: map[string]uint64{
			"page_ins":    uint64(p.task.Ptinfo.Pageins),
			"page_faults": uint64(p.task.Ptinfo.Faults),
		},
	}
}

func getProcTaskAllInfo(pid int, info *procTaskAllInfo) error {
	size := C.int(unsafe.Sizeof(*info))
	ptr := unsafe.Pointer(info)

	n := C.proc_pidinfo(C.int(pid), C.PROC_PIDTASKALLINFO, 0, ptr, size)
	if n != size {
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
