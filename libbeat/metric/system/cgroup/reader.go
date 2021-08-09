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

package cgroup

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup/cgv1"
	"github.com/elastic/beats/v7/libbeat/metric/system/cgroup/cgv2"
)

// StatsV1 contains metrics and limits from each of the cgroup subsystems.
type StatsV1 struct {
	ID            string                       `json:"id,omitempty"`   // ID of the cgroup.
	Path          string                       `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	CPU           *cgv1.CPUSubsystem           `json:"cpu"`
	CPUAccounting *cgv1.CPUAccountingSubsystem `json:"cpuacct"`
	Memory        *cgv1.MemorySubsystem        `json:"memory"`
	BlockIO       *cgv1.BlockIOSubsystem       `json:"blkio"`
}

// StatsV2 contains metrics and limits from each of the cgroup subsystems.
type StatsV2 struct {
	ID     string `json:"id,omitempty"`   // ID of the cgroup.
	Path   string `json:"path,omitempty"` // Path to the cgroup relative to the cgroup subsystem's mountpoint.
	CPU    *cgv2.CPUSubsystem
	Memory *cgv2.MemorySubsystem
	IO     *cgv2.IOSubsystem
}

// CgroupsVersion is a version tag that defines what version of cgroups is attached to a process
type CgroupsVersion int

// CgroupsV1 indicates that a process is cgroupsv1
const CgroupsV1 CgroupsVersion = 1

// CgroupsV2 indicates that a process is cgroupsv2
const CgroupsV2 CgroupsVersion = 2

type mount struct {
	subsystem  string // Subsystem name (e.g. cpuacct).
	mountpoint string // Mountpoint of the subsystem (e.g. /cgroup/cpuacct).
	path       string // Relative path to the cgroup (e.g. /docker/<id>).
	id         string // ID of the cgroup.
	fullPath   string // Absolute path to the cgroup. It's the mountpoint joined with the path.
}

// Reader reads cgroup metrics and limits.
type Reader struct {
	// Mountpoint of the root filesystem. Defaults to / if not set. This can be
	// useful for example if you mount / as /rootfs inside of a container.
	rootfsMountpoint         string
	ignoreRootCgroups        bool // Ignore a cgroup when its path is "/".
	cgroupsHierarchyOverride string
	cgroupMountpoints        Mountpoints // Mountpoints for each subsystem (e.g. cpu, cpuacct, memory, blkio).
}

// ReaderOptions holds options for NewReaderOptions.
type ReaderOptions struct {
	// RootfsMountpoint holds the mountpoint of the root filesystem.
	//
	// If unspecified, "/" is assumed.
	RootfsMountpoint string

	// IgnoreRootCgroups ignores cgroup subsystem with the path "/".
	IgnoreRootCgroups bool

	// CgroupsHierarchyOverride is an optional path override for cgroup
	// subsystem paths. If non-empty, this will be used instead of the
	// paths specified in /proc/<pid>/cgroup.
	//
	// This should be set to "/" when running within a Docker container,
	// where the paths in /proc/<pid>/cgroup do not correspond to any
	// paths under /sys/fs/cgroup.
	CgroupsHierarchyOverride string
}

// NewReader creates and returns a new Reader.
func NewReader(rootfsMountpoint string, ignoreRootCgroups bool) (*Reader, error) {
	return NewReaderOptions(ReaderOptions{
		RootfsMountpoint:  rootfsMountpoint,
		IgnoreRootCgroups: ignoreRootCgroups,
	})
}

// NewReaderOptions creates and returns a new Reader with the given options.
func NewReaderOptions(opts ReaderOptions) (*Reader, error) {
	if opts.RootfsMountpoint == "" {
		opts.RootfsMountpoint = "/"
	}

	// Determine what subsystems are supported by the kernel.
	subsystems, err := SupportedSubsystems(opts.RootfsMountpoint)
	if err != nil {
		return nil, errors.Wrap(err, "error finding subsystems")
	}

	// Locate the mountpoints of those subsystems.
	mountpoints, err := SubsystemMountpoints(opts.RootfsMountpoint, subsystems)
	if err != nil {
		return nil, errors.Wrap(err, "error finding mountpoints")
	}

	return &Reader{
		rootfsMountpoint:         opts.RootfsMountpoint,
		ignoreRootCgroups:        opts.IgnoreRootCgroups,
		cgroupsHierarchyOverride: opts.CgroupsHierarchyOverride,
		cgroupMountpoints:        mountpoints,
	}, nil
}

// CgroupsVersion reports if the given PID is attached to a V1 or V2 controller
func (r *Reader) CgroupsVersion(pid int) (CgroupsVersion, error) {
	cgPath := filepath.Join(r.rootfsMountpoint, "/proc/", fmt.Sprintf("%d", pid), "cgroup")
	cgstring, err := ioutil.ReadFile(cgPath)
	if err != nil {
		return CgroupsV1, errors.Wrapf(err, "error reading %s", cgPath)
	}
	//V2 cgroups always begin with 0::/
	if strings.Contains(string(cgstring), "0::/") {
		return CgroupsV2, nil
	}
	return CgroupsV1, nil
}

// GetStatsForPid is a generic method that returns a CGStats interface for V1 and V2
// cgroup statistics. For applications that require raw metrics, use GetV*StatsForProcess()
func (r *Reader) GetStatsForPid(pid int) (CGStats, error) {
	v, err := r.CgroupsVersion(pid)
	if err != nil {
		return nil, errors.Wrapf(err, "error finding cgroup version for pid %d", pid)
	}
	if v == CgroupsV1 {
		return r.GetV1StatsForProcess(pid)
	}
	return r.GetV2StatsForProcess(pid)
}

// GetV1StatsForProcess returns cgroup metrics and limits associated with a process.
func (r *Reader) GetV1StatsForProcess(pid int) (*StatsV1, error) {
	// Read /proc/[pid]/cgroup to get the paths to the cgroup metrics.
	paths, err := r.ProcessCgroupPaths(pid)
	if err != nil {
		return nil, err
	}
	stats := StatsV1{}
	stats.Path, stats.ID = getCommonCgroupMetadata(paths)
	for conName, cgPath := range paths {
		if r.ignoreRootCgroups && cgPath.ControllerPath == "/" {
			continue
		}
		err := getStatsV1(cgPath, conName, &stats)
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching stats for controller %s", conName)
		}
	}

	// Return nil if no metrics were collected.
	if stats.BlockIO == nil && stats.CPU == nil && stats.CPUAccounting == nil && stats.Memory == nil {
		return nil, nil
	}

	return &stats, nil
}

// GetV2StatsForProcess returns cgroup metrics and limits associated with a process.
func (r *Reader) GetV2StatsForProcess(pid int) (*StatsV2, error) {
	// Read /proc/[pid]/cgroup to get the paths to the cgroup metrics.
	paths, err := r.ProcessCgroupPaths(pid)
	if err != nil {
		return nil, err
	}
	stats := StatsV2{}
	stats.ID, stats.Path = getCommonCgroupMetadata(paths)
	for conName, cgPath := range paths {
		if r.ignoreRootCgroups && cgPath.ControllerPath == "/" {
			continue
		}
		err := getStatsV2(cgPath, conName, &stats)
		if err != nil {
			return nil, errors.Wrapf(err, "error fetching stats for controller %s", conName)
		}
	}
	return &stats, nil
}

// ProcessCgroupPaths is a wrapper around Reader.ProcessCgroupPaths for libraries that only need the slimmer functionality from
// the gosigar cgroups code. This does not have the same function signature, and consumers still need to distinguish between v1 and v2 cgroups.
func ProcessCgroupPaths(hostfs string, pid int) (map[string]ControllerPath, error) {
	reader, err := NewReader(hostfs, false)
	if err != nil {
		return nil, errors.Wrap(err, "error creating cgroups reader")
	}
	return reader.ProcessCgroupPaths(pid)
}

func getStatsV2(path ControllerPath, name string, stats *StatsV2) error {
	id := filepath.Base(path.ControllerPath)

	switch name {
	case "cpu":
		stats.CPU = &cgv2.CPUSubsystem{}
		err := stats.CPU.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching CPU stats")
		}
		stats.CPU.ID = id
		stats.CPU.Path = path.ControllerPath
	case "memory":
		stats.Memory = &cgv2.MemorySubsystem{}
		err := stats.Memory.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching Memory stats")
		}
		stats.Memory.ID = id
		stats.Memory.Path = path.ControllerPath
	case "io":
		stats.IO = &cgv2.IOSubsystem{}
		err := stats.IO.Get(path.FullPath, true)
		if err != nil {
			return errors.Wrap(err, "error fetching IO stats")
		}
		stats.IO.ID = id
		stats.IO.Path = path.ControllerPath
	}

	return nil
}

func getStatsV1(path ControllerPath, name string, stats *StatsV1) error {

	id := filepath.Base(path.ControllerPath)

	switch name {
	case "blkio":
		stats.BlockIO = &cgv1.BlockIOSubsystem{}
		err := stats.BlockIO.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching BlockIO stats")
		}
		stats.BlockIO.ID = id
		stats.BlockIO.Path = path.ControllerPath
	case "cpu":
		stats.CPU = &cgv1.CPUSubsystem{}
		err := stats.CPU.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching cpu stats")
		}
		stats.CPU.ID = id
		stats.CPU.Path = path.ControllerPath
	case "cpuacct":
		stats.CPUAccounting = &cgv1.CPUAccountingSubsystem{}
		err := stats.CPUAccounting.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching cpuacct stats")
		}
		stats.CPUAccounting.ID = id
		stats.CPUAccounting.Path = path.ControllerPath
	case "memory":
		stats.Memory = &cgv1.MemorySubsystem{}
		err := stats.Memory.Get(path.FullPath)
		if err != nil {
			return errors.Wrap(err, "error fetching memory stats")
		}
		stats.Memory.ID = id
		stats.Memory.Path = path.ControllerPath
	}

	return nil
}

// getCommonCgroupMetadata returns Metadata containing the cgroup path and ID
// iff all subsystems share a common path and ID. This is common for
// containerized processes. If there is no common path and ID then the returned
// values are empty strings.
func getCommonCgroupMetadata(mounts map[string]ControllerPath) (string, string) {
	var path string
	for _, m := range mounts {
		if path == "" {
			path = m.ControllerPath
		} else if path != m.ControllerPath {
			// All paths are not the same.
			return "", ""
		}
	}

	return path, filepath.Base(path)
}
