// Copyright 2018 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package linux

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/prometheus/procfs"

	"github.com/elastic/go-sysinfo/types"
)

const userHz = 100

func (s linuxSystem) Processes() ([]types.Process, error) {
	procs, err := procfs.AllProcs()
	if err != nil {
		return nil, err
	}

	processes := make([]types.Process, 0, len(procs))
	for _, proc := range procs {
		processes = append(processes, &process{Proc: proc})
	}
	return processes, nil
}

func (s linuxSystem) Process(pid int) (types.Process, error) {
	proc, err := procfs.NewProc(pid)
	if err != nil {
		return nil, err
	}

	return &process{Proc: proc}, nil
}

type process struct {
	procfs.Proc
	info *types.ProcessInfo
}

func (p *process) CWD() (string, error) {
	// TODO: add CWD to procfs
	link := filepath.Join(procfs.DefaultMountPoint, strconv.Itoa(p.PID), "cwd")

	cwd, err := os.Readlink(link)
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

	bootTime, err := BootTime()
	if err != nil {
		return types.ProcessInfo{}, err
	}

	p.info = &types.ProcessInfo{
		Name:      stat.Comm,
		PID:       p.PID,
		PPID:      stat.PPID,
		CWD:       cwd,
		Exe:       exe,
		Args:      args,
		StartTime: bootTime.Add(ticksToDuration(stat.Starttime)),
	}

	return *p.info, nil
}

func (p *process) Memory() types.MemoryInfo {
	stat, err := p.NewStat()
	if err != nil {
		return types.MemoryInfo{}
	}

	return types.MemoryInfo{
		Timestamp: time.Now(),
		Resident:  uint64(stat.ResidentMemory()),
		Virtual:   uint64(stat.VirtualMemory()),
	}
}

func (p *process) CPUTime() types.CPUTimes {
	stat, err := p.NewStat()
	if err != nil {
		return types.CPUTimes{}
	}

	fmt.Println("UTime", stat.UTime, "STime", stat.STime)
	return types.CPUTimes{
		Timestamp: time.Now(),
		User:      ticksToDuration(uint64(stat.UTime)),
		System:    ticksToDuration(uint64(stat.STime)),
	}
}

func (p *process) FileDescriptors() ([]string, error) {
	return p.Proc.FileDescriptorTargets()
}

func (p *process) FileDescriptorCount() (int, error) {
	return p.Proc.FileDescriptorsLen()
}

func (p *process) Environment() (map[string]string, error) {
	// TODO: add Environment to procfs
	filename := filepath.Join(procfs.DefaultMountPoint, strconv.Itoa(p.PID), "environ")
	content, err := ioutil.ReadFile(filename)
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

func ticksToDuration(ticks uint64) time.Duration {
	seconds := float64(ticks) / float64(userHz) * float64(time.Second)
	return time.Duration(int64(seconds))
}
