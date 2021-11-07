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

package aix

/*
#cgo LDFLAGS: -L/usr/lib -lperfstat

#include <libperfstat.h>
#include <procinfo.h>
#include <sys/proc.h>

*/
import "C"

import (
	"os"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/internal/registry"
	"github.com/elastic/go-sysinfo/providers/shared"
	"github.com/elastic/go-sysinfo/types"
)

//go:generate sh -c "go tool cgo -godefs defs_aix.go | sed 's/*byte/uint64/g' > ztypes_aix_ppc64.go"
// As cgo will return some psinfo's fields with *byte, binary.Read will refuse this type.

func init() {
	registry.Register(aixSystem{})
}

type aixSystem struct{}

// Host returns a new AIX host.
func (aixSystem) Host() (types.Host, error) {
	return newHost()
}

type host struct {
	info types.HostInfo
}

// Architecture returns the architecture of the host
func Architecture() (string, error) {
	return "ppc", nil
}

// Info returns the host details.
func (h *host) Info() types.HostInfo {
	return h.info
}

// Info returns the current CPU usage of the host.
func (*host) CPUTime() (types.CPUTimes, error) {
	clock := uint64(C.sysconf(C._SC_CLK_TCK))
	tick2nsec := func(val uint64) uint64 {
		return val * 1e9 / clock
	}

	cpudata := C.perfstat_cpu_total_t{}

	if _, err := C.perfstat_cpu_total(nil, &cpudata, C.sizeof_perfstat_cpu_total_t, 1); err != nil {
		return types.CPUTimes{}, errors.Wrap(err, "error while callin perfstat_cpu_total")
	}

	return types.CPUTimes{
		User:   time.Duration(tick2nsec(uint64(cpudata.user))),
		System: time.Duration(tick2nsec(uint64(cpudata.sys))),
		Idle:   time.Duration(tick2nsec(uint64(cpudata.idle))),
		IOWait: time.Duration(tick2nsec(uint64(cpudata.wait))),
	}, nil
}

// Memory returns the current memory usage of the host.
func (*host) Memory() (*types.HostMemoryInfo, error) {
	var mem types.HostMemoryInfo

	pagesize := uint64(os.Getpagesize())

	meminfo := C.perfstat_memory_total_t{}
	_, err := C.perfstat_memory_total(nil, &meminfo, C.sizeof_perfstat_memory_total_t, 1)
	if err != nil {
		return nil, errors.Wrap(err, "perfstat_memory_total failed")
	}

	mem.Total = uint64(meminfo.real_total) * pagesize
	mem.Free = uint64(meminfo.real_free) * pagesize
	mem.Used = uint64(meminfo.real_inuse) * pagesize

	// There is no real equivalent to memory available in AIX.
	mem.Available = mem.Free

	mem.VirtualTotal = uint64(meminfo.virt_total) * pagesize
	mem.VirtualFree = mem.Free + uint64(meminfo.pgsp_free)*pagesize
	mem.VirtualUsed = mem.VirtualTotal - mem.VirtualFree

	return &mem, nil
}

func newHost() (*host, error) {
	h := &host{}
	r := &reader{}
	r.architecture(h)
	r.bootTime(h)
	r.hostname(h)
	r.network(h)
	r.kernelVersion(h)
	r.os(h)
	r.time(h)
	r.uniqueID(h)
	return h, r.Err()
}

type reader struct {
	errs []error
}

func (r *reader) addErr(err error) bool {
	if err != nil {
		if errors.Cause(err) != types.ErrNotImplemented {
			r.errs = append(r.errs, err)
		}
		return true
	}
	return false
}

func (r *reader) Err() error {
	if len(r.errs) > 0 {
		return &multierror.MultiError{Errors: r.errs}
	}
	return nil
}

func (r *reader) architecture(h *host) {
	v, err := Architecture()
	if r.addErr(err) {
		return
	}
	h.info.Architecture = v
}

func (r *reader) bootTime(h *host) {
	v, err := BootTime()
	if r.addErr(err) {
		return
	}
	h.info.BootTime = v
}

func (r *reader) hostname(h *host) {
	v, err := os.Hostname()
	if r.addErr(err) {
		return
	}
	h.info.Hostname = v
}

func (r *reader) network(h *host) {
	ips, macs, err := shared.Network()
	if r.addErr(err) {
		return
	}
	h.info.IPs = ips
	h.info.MACs = macs
}

func (r *reader) kernelVersion(h *host) {
	v, err := KernelVersion()
	if r.addErr(err) {
		return
	}
	h.info.KernelVersion = v
}

func (r *reader) os(h *host) {
	v, err := OperatingSystem()
	if r.addErr(err) {
		return
	}
	h.info.OS = v
}

func (*reader) time(h *host) {
	h.info.Timezone, h.info.TimezoneOffsetSec = time.Now().Zone()
}

func (r *reader) uniqueID(h *host) {
	v, err := MachineID()
	if r.addErr(err) {
		return
	}
	h.info.UniqueID = v
}
