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

package types

import "time"

type Process interface {
	Info() (ProcessInfo, error)
}

type ProcessInfo struct {
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	CWD       string    `json:"cwd"`
	Exe       string    `json:"exe"`
	Args      []string  `json:"args"`
	StartTime time.Time `json:"start_time"`
}

type Environment interface {
	Environment() (map[string]string, error)
}

type FileDescriptor interface {
	FileDescriptors() ([]string, error)
	FileDescriptorCount() (int, error)
}

type CPUTimer interface {
	CPUTime() CPUTimes
}

type Memory interface {
	Memory() MemoryInfo
}

type CPUTimes struct {
	Timestamp time.Time     `json:"timestamp"` // Time at which samples were collected.
	User      time.Duration `json:"user"`
	System    time.Duration `json:"system"`
	Idle      time.Duration `json:"idle,omitempty"`
	IOWait    time.Duration `json:"iowait,omitempty"`
	IRQ       time.Duration `json:"irq,omitempty"`
	Nice      time.Duration `json:"nice,omitempty"`
	SoftIRQ   time.Duration `json:"soft_irq,omitempty"`
	Steal     time.Duration `json:"steal,omitempty"`
}

func (cpu CPUTimes) Total() time.Duration {
	return cpu.User + cpu.System + cpu.Idle + cpu.IOWait + cpu.IRQ + cpu.Nice +
		cpu.SoftIRQ + cpu.Steal
}

type MemoryInfo struct {
	Timestamp time.Time         `json:"timestamp"` // Time at which samples were collected.
	Resident  uint64            `json:"resident"`
	Virtual   uint64            `json:"virtual"`
	Metrics   map[string]uint64 `json:"raw,omitempty"` // Other memory related metrics.
}

type SeccompInfo struct {
	Mode          string   `json:"mode"`
	EffectiveCaps []string `json:"effective_capabilities"`
	NoNewPrivs    *bool    `json:"no_new_privs"`
}
