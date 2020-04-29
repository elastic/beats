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

package types

import "time"

// Process is the main wrapper for gathering information on a process
type Process interface {
	CPUTimer
	Info() (ProcessInfo, error)
	Memory() (MemoryInfo, error)
	User() (UserInfo, error)
	Parent() (Process, error)
	PID() int
}

// ProcessInfo contains basic stats about a process
type ProcessInfo struct {
	Name      string    `json:"name"`
	PID       int       `json:"pid"`
	PPID      int       `json:"ppid"`
	CWD       string    `json:"cwd"`
	Exe       string    `json:"exe"`
	Args      []string  `json:"args"`
	StartTime time.Time `json:"start_time"`
}

// UserInfo contains information about the UID and GID
// values of a process.
type UserInfo struct {
	// UID is the user ID.
	// On Linux and Darwin (macOS) this is the real user ID.
	// On Windows, this is the security identifier (SID) of the
	// user account of the process access token.
	UID string `json:"uid"`

	// On Linux and Darwin (macOS) this is the effective user ID.
	// On Windows, this is empty.
	EUID string `json:"euid"`

	// On Linux and Darwin (macOS) this is the saved user ID.
	// On Windows, this is empty.
	SUID string `json:"suid"`

	// GID is the primary group ID.
	// On Linux and Darwin (macOS) this is the real group ID.
	// On Windows, this is the security identifier (SID) of the
	// primary group of the process access token.
	GID string `json:"gid"`

	// On Linux and Darwin (macOS) this is the effective group ID.
	// On Windows, this is empty.
	EGID string `json:"egid"`

	// On Linux and Darwin (macOS) this is the saved group ID.
	// On Windows, this is empty.
	SGID string `json:"sgid"`
}

// Environment is the interface that wraps the Environment method.
// Environment returns variables for a process
type Environment interface {
	Environment() (map[string]string, error)
}

// OpenHandleEnumerator is the interface that wraps the OpenHandles method.
// OpenHandles lists the open file handles.
type OpenHandleEnumerator interface {
	OpenHandles() ([]string, error)
}

// OpenHandleCounter is the interface that wraps the OpenHandleCount method.
// OpenHandleCount returns the number of open file handles.
type OpenHandleCounter interface {
	OpenHandleCount() (int, error)
}

// CPUTimer is the interface that wraps the CPUTime method.
// CPUTime returns CPU time info
type CPUTimer interface {
	// CPUTime returns a CPUTimes structure for
	// the host or some process.
	//
	// The User and System fields are guaranteed
	// to be populated for all platforms, and
	// for both hosts and processes.
	CPUTime() (CPUTimes, error)
}

// CPUTimes contains CPU timing stats for a process
type CPUTimes struct {
	User    time.Duration `json:"user"`
	System  time.Duration `json:"system"`
	Idle    time.Duration `json:"idle,omitempty"`
	IOWait  time.Duration `json:"iowait,omitempty"`
	IRQ     time.Duration `json:"irq,omitempty"`
	Nice    time.Duration `json:"nice,omitempty"`
	SoftIRQ time.Duration `json:"soft_irq,omitempty"`
	Steal   time.Duration `json:"steal,omitempty"`
}

// Total returns the total CPU time
func (cpu CPUTimes) Total() time.Duration {
	return cpu.User + cpu.System + cpu.Idle + cpu.IOWait + cpu.IRQ + cpu.Nice +
		cpu.SoftIRQ + cpu.Steal
}

// MemoryInfo contains memory stats for a process
type MemoryInfo struct {
	Resident uint64            `json:"resident_bytes"`
	Virtual  uint64            `json:"virtual_bytes"`
	Metrics  map[string]uint64 `json:"raw,omitempty"` // Other memory related metrics.
}

// SeccompInfo contains seccomp info for a process
type SeccompInfo struct {
	Mode       string `json:"mode"`
	NoNewPrivs *bool  `json:"no_new_privs,omitempty"` // Added in kernel 4.10.
}

// CapabilityInfo contains capability set info.
type CapabilityInfo struct {
	Inheritable []string `json:"inheritable"`
	Permitted   []string `json:"permitted"`
	Effective   []string `json:"effective"`
	Bounding    []string `json:"bounding"`
	Ambient     []string `json:"ambient"`
}

// Capabilities is the interface that wraps the Capabilities method.
// Capabilities returns capabilities for a process
type Capabilities interface {
	Capabilities() (*CapabilityInfo, error)
}

// Seccomp is the interface that wraps the Seccomp method.
// Seccomp returns seccomp info on Linux
type Seccomp interface {
	Seccomp() (*SeccompInfo, error)
}
