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
	Resident  uint64            `json:"resident_bytes"`
	Virtual   uint64            `json:"virtual_bytes"`
	Metrics   map[string]uint64 `json:"raw,omitempty"` // Other memory related metrics.
}

type SeccompInfo struct {
	Mode       string `json:"mode"`
	NoNewPrivs *bool  `json:"no_new_privs,omitempty"` // Added in kernel 4.10.
}

type CapabilityInfo struct {
	Inheritable []string `json:"inheritable"`
	Permitted   []string `json:"permitted"`
	Effective   []string `json:"effective"`
	Bounding    []string `json:"bounding"`
	Ambient     []string `json:"ambient"`
}

type Capabilities interface {
	Capabilities() (*CapabilityInfo, error)
}

type Seccomp interface {
	Seccomp() (*SeccompInfo, error)
}
