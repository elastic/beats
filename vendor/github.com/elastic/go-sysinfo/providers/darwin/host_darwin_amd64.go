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

import (
	"os"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/internal/registry"
	"github.com/elastic/go-sysinfo/providers/shared"
	"github.com/elastic/go-sysinfo/types"
)

func init() {
	registry.Register(darwinSystem{})
}

type darwinSystem struct{}

func (s darwinSystem) Host() (types.Host, error) {
	return newHost()
}

type host struct {
	info types.HostInfo
}

func (h *host) Info() types.HostInfo {
	return h.info
}

func (h *host) CPUTime() (*types.CPUTimes, error) {
	cpu, err := getHostCPULoadInfo()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get host CPU usage")
	}

	ticksPerSecond := time.Duration(getClockTicks())

	return &types.CPUTimes{
		Timestamp: time.Now(),
		User:      time.Duration(cpu.User) * time.Second / ticksPerSecond,
		System:    time.Duration(cpu.System) * time.Second / ticksPerSecond,
		Idle:      time.Duration(cpu.Idle) * time.Second / ticksPerSecond,
		Nice:      time.Duration(cpu.Nice) * time.Second / ticksPerSecond,
	}, nil
}

func (h *host) Memory() (*types.HostMemoryInfo, error) {
	mem := &types.HostMemoryInfo{Timestamp: time.Now()}

	// Total physical memory.
	if err := sysctlByName("hw.memsize", &mem.Total); err != nil {
		return nil, errors.Wrap(err, "failed to get total physical memory")
	}

	// Page size for computing byte totals.
	pageSizeBytes, err := getPageSize()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get page size")
	}

	// Virtual Memory Statistics
	vmStat, err := getHostVMInfo64()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get virtual memory statistics")
	}

	// Swap
	swap, err := getSwapUsage()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get swap usage")
	}

	inactiveBytes := uint64(vmStat.Inactive_count) * pageSizeBytes
	purgeableBytes := uint64(vmStat.Purgeable_count) * pageSizeBytes
	mem.Metrics = map[string]uint64{
		"active_bytes":         uint64(vmStat.Active_count) * pageSizeBytes,
		"compressed_bytes":     uint64(vmStat.Compressor_page_count) * pageSizeBytes,
		"compressions_bytes":   uint64(vmStat.Compressions) * pageSizeBytes, // Cumulative compressions.
		"copy_on_write_faults": vmStat.Cow_faults,
		"decompressions_bytes": uint64(vmStat.Decompressions) * pageSizeBytes,      // Cumulative decompressions.
		"external_bytes":       uint64(vmStat.External_page_count) * pageSizeBytes, // File Cache / File-backed pages
		"inactive_bytes":       inactiveBytes,
		"internal_bytes":       uint64(vmStat.Internal_page_count) * pageSizeBytes, // App Memory / Anonymous
		"page_ins_bytes":       uint64(vmStat.Pageins) * pageSizeBytes,
		"page_outs_bytes":      uint64(vmStat.Pageouts) * pageSizeBytes,
		"purgeable_bytes":      purgeableBytes,
		"purged_bytes":         uint64(vmStat.Purges) * pageSizeBytes,
		"reactivated_bytes":    uint64(vmStat.Reactivations) * pageSizeBytes,
		"speculative_bytes":    uint64(vmStat.Speculative_count) * pageSizeBytes,
		"swap_ins_bytes":       uint64(vmStat.Swapins) * pageSizeBytes,
		"swap_outs_bytes":      uint64(vmStat.Swapouts) * pageSizeBytes,
		"throttled_bytes":      uint64(vmStat.Throttled_count) * pageSizeBytes,
		"translation_faults":   vmStat.Faults,
		"uncompressed_bytes":   uint64(vmStat.Total_uncompressed_pages_in_compressor) * pageSizeBytes,
		"wired_bytes":          uint64(vmStat.Wire_count) * pageSizeBytes,
		"zero_filled_bytes":    uint64(vmStat.Zero_fill_count) * pageSizeBytes,
	}

	// From Activity Monitor: Memory Used = App Memory (internal) + Wired + Compressed
	// https://support.apple.com/en-us/HT201538
	mem.Used = uint64(vmStat.Internal_page_count+vmStat.Wire_count+vmStat.Compressor_page_count) * pageSizeBytes
	mem.Free = uint64(vmStat.Free_count) * pageSizeBytes
	mem.Available = mem.Free + inactiveBytes + purgeableBytes
	mem.VirtualTotal = swap.Total
	mem.VirtualUsed = swap.Used
	mem.VirtualFree = swap.Available

	return mem, nil
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

func (r *reader) time(h *host) {
	h.info.Timezone, h.info.TimezoneOffsetSec = time.Now().Zone()
}

func (r *reader) uniqueID(h *host) {
	// TODO: call gethostuuid(uuid [16]byte, timespec)
}
