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

package procs

import (
	"fmt"
	"time"
)

// Backend values for ProcsConfig.Backend.
const (
	// BackendProcfs resolves ports to processes by polling the OS
	// socket table (/proc on Linux, IP Helper on Windows). This is
	// the default. Lookups for sockets that already closed fail.
	BackendProcfs = "procfs"

	// BackendKernelTracing tracks socket and process lifecycle with
	// quark (eBPF), so short-lived connections can be resolved even
	// after their socket closed. Linux amd64/arm64 only; initialization
	// fails on other systems or when eBPF is unavailable.
	BackendKernelTracing = "kernel_tracing"

	// BackendAuto tries kernel_tracing and silently falls back to
	// procfs when it is unavailable.
	BackendAuto = "auto"
)

type ProcsConfig struct {
	Enabled         bool          `config:"enabled"`
	Backend         string        `config:"backend"`
	MaxProcReadFreq time.Duration `config:"max_proc_read_freq"`
	Monitored       []ProcConfig  `config:"monitored"`
	RefreshPidsFreq time.Duration `config:"refresh_pids_freq"`

	// KernelTracingGraceTime is how long the kernel_tracing backend
	// keeps closed connections and exited processes resolvable. It is
	// not user-configurable: the beater derives it from the flows
	// configuration (see GraceTimeForFlows). Zero selects
	// DefaultKernelTracingGraceTime.
	KernelTracingGraceTime time.Duration `config:",ignore"`
}

const (
	// DefaultKernelTracingGraceTime is the kernel_tracing grace time
	// used when flows are disabled. Protocol plugins look processes up
	// as soon as a transaction completes, so only a short window past
	// the socket close is needed.
	DefaultKernelTracingGraceTime = 10 * time.Second

	// kernelTracingGraceMargin is added to the flow reporting delay to
	// absorb scheduling jitter in the flows worker.
	kernelTracingGraceMargin = 10 * time.Second
)

// GraceTimeForFlows returns the kernel_tracing grace time needed when
// flows are enabled with the given timeout and reporting period. A
// flow is reported up to timeout+period after its last packet, and the
// backend must still remember the connection at that point.
func GraceTimeForFlows(timeout, period time.Duration) time.Duration {
	return max(timeout, 0) + max(period, 0) + kernelTracingGraceMargin
}

type ProcConfig struct {
	Process     string `config:"process"`
	CmdlineGrep string `config:"cmdline_grep"`
}

// Validate validates the ProcsConfig. It is called by go-ucfg on Unpack.
func (c *ProcsConfig) Validate() error {
	switch c.Backend {
	case "", BackendProcfs, BackendKernelTracing, BackendAuto:
		return nil
	}
	return fmt.Errorf("invalid procs backend %q: must be one of %s, %s or %s",
		c.Backend, BackendProcfs, BackendKernelTracing, BackendAuto)
}
