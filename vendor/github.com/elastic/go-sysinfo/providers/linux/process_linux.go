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

package linux

import (
	"bytes"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/procfs"

	"github.com/elastic/go-sysinfo/types"
)

const userHz = 100

func (s linuxSystem) Processes() ([]types.Process, error) {
	procs, err := s.procFS.AllProcs()
	if err != nil {
		return nil, err
	}

	processes := make([]types.Process, 0, len(procs))
	for _, proc := range procs {
		processes = append(processes, &process{Proc: proc, fs: s.procFS})
	}
	return processes, nil
}

func (s linuxSystem) Process(pid int) (types.Process, error) {
	proc, err := s.procFS.NewProc(pid)
	if err != nil {
		return nil, err
	}

	return &process{Proc: proc, fs: s.procFS}, nil
}

func (s linuxSystem) Self() (types.Process, error) {
	proc, err := s.procFS.Self()
	if err != nil {
		return nil, err
	}

	return &process{Proc: proc, fs: s.procFS}, nil
}

type process struct {
	procfs.Proc
	fs   procFS
	info *types.ProcessInfo
}

func (p *process) PID() int {
	return p.Proc.PID
}

func (p *process) Parent() (types.Process, error) {
	info, err := p.Info()
	if err != nil {
		return nil, err
	}

	proc, err := p.fs.NewProc(info.PPID)
	if err != nil {
		return nil, err
	}

	return &process{Proc: proc, fs: p.fs}, nil
}

func (p *process) path(pa ...string) string {
	return p.fs.path(append([]string{strconv.Itoa(p.PID())}, pa...)...)
}

func (p *process) CWD() (string, error) {
	// TODO: add CWD to procfs
	cwd, err := os.Readlink(p.path("cwd"))
	if os.IsNotExist(err) {
		return "", nil
	}

	return cwd, err
}

func (p *process) Info() (types.ProcessInfo, error) {
	if p.info != nil {
		return *p.info, nil
	}

	stat, err := p.NewStat()
	if err != nil {
		return types.ProcessInfo{}, err
	}

	exe, err := p.Executable()
	if err != nil {
		return types.ProcessInfo{}, err
	}

	args, err := p.CmdLine()
	if err != nil {
		return types.ProcessInfo{}, err
	}

	cwd, err := p.CWD()
	if err != nil {
		return types.ProcessInfo{}, err
	}

	bootTime, err := bootTime(p.fs.FS)
	if err != nil {
		return types.ProcessInfo{}, err
	}

	p.info = &types.ProcessInfo{
		Name:      stat.Comm,
		PID:       p.PID(),
		PPID:      stat.PPID,
		CWD:       cwd,
		Exe:       exe,
		Args:      args,
		StartTime: bootTime.Add(ticksToDuration(stat.Starttime)),
	}

	return *p.info, nil
}

func (p *process) Memory() (types.MemoryInfo, error) {
	stat, err := p.NewStat()
	if err != nil {
		return types.MemoryInfo{}, err
	}

	return types.MemoryInfo{
		Resident: uint64(stat.ResidentMemory()),
		Virtual:  uint64(stat.VirtualMemory()),
	}, nil
}

func (p *process) CPUTime() (types.CPUTimes, error) {
	stat, err := p.NewStat()
	if err != nil {
		return types.CPUTimes{}, err
	}

	return types.CPUTimes{
		User:   ticksToDuration(uint64(stat.UTime)),
		System: ticksToDuration(uint64(stat.STime)),
	}, nil
}

// OpenHandles returns the list of open file descriptors of the process.
func (p *process) OpenHandles() ([]string, error) {
	return p.Proc.FileDescriptorTargets()
}

// OpenHandles returns the number of open file descriptors of the process.
func (p *process) OpenHandleCount() (int, error) {
	return p.Proc.FileDescriptorsLen()
}

func (p *process) Environment() (map[string]string, error) {
	// TODO: add Environment to procfs
	content, err := ioutil.ReadFile(p.path("environ"))
	if err != nil {
		return nil, err
	}

	env := map[string]string{}
	pairs := bytes.Split(content, []byte{0})
	for _, kv := range pairs {
		parts := bytes.SplitN(kv, []byte{'='}, 2)
		if len(parts) != 2 {
			continue
		}

		key := string(bytes.TrimSpace(parts[0]))
		if key == "" {
			continue
		}

		env[key] = string(parts[1])
	}

	return env, nil
}

func (p *process) Seccomp() (*types.SeccompInfo, error) {
	content, err := ioutil.ReadFile(p.path("status"))
	if err != nil {
		return nil, err
	}

	return readSeccompFields(content)
}

func (p *process) Capabilities() (*types.CapabilityInfo, error) {
	content, err := ioutil.ReadFile(p.path("status"))
	if err != nil {
		return nil, err
	}

	return readCapabilities(content)
}

func (p *process) User() (types.UserInfo, error) {
	content, err := ioutil.ReadFile(p.path("status"))
	if err != nil {
		return types.UserInfo{}, err
	}

	var user types.UserInfo
	err = parseKeyValue(content, ":", func(key, value []byte) error {
		// See proc(5) for the format of /proc/[pid]/status
		switch string(key) {
		case "Uid":
			ids := strings.Split(string(value), "\t")
			if len(ids) >= 3 {
				user.UID = ids[0]
				user.EUID = ids[1]
				user.SUID = ids[2]
			}
		case "Gid":
			ids := strings.Split(string(value), "\t")
			if len(ids) >= 3 {
				user.GID = ids[0]
				user.EGID = ids[1]
				user.SGID = ids[2]
			}
		}
		return nil
	})

	return user, nil
}

func ticksToDuration(ticks uint64) time.Duration {
	seconds := float64(ticks) / float64(userHz) * float64(time.Second)
	return time.Duration(int64(seconds))
}
