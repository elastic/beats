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
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/elastic-agent-libs/logp"
)

var (
	// ErrCgroupsMissing indicates the /proc/cgroups was not found. This means
	// that cgroups were disabled at compile time (CONFIG_CGROUPS=n) or that
	// an invalid rootfs path was given.
	ErrCgroupsMissing = errors.New("cgroups not found or unsupported by OS")
)

// mountinfo represents a subset of the fields containing /proc/[pid]/mountinfo.
type mountinfo struct {
	mountpoint     string
	filesystemType string
	superOptions   []string
}

// Mountpoints organizes info about V1 and V2 cgroup mountpoints
// V2 uses a "unified" hierarchy, so we have less to keep track of
type Mountpoints struct {
	V1Mounts map[string]string
	V2Loc    string
}

// ControllerPath wraps the controller path
type ControllerPath struct {
	ControllerPath string
	FullPath       string
	IsV2           bool
}

// PathList contains the V1 and V2 controller paths in a process
// Separate the V1 and V2 cgroups so we don't have hybrid cgroups fighting for one namespace
type PathList struct {
	V1 map[string]ControllerPath
	V2 map[string]ControllerPath
}

// Flatten combines the V1 and V2 cgroups in cases where we don't need a map with keys
func (pl PathList) Flatten() []ControllerPath {
	list := []ControllerPath{}
	for _, v1 := range pl.V1 {
		list = append(list, v1)
	}
	for _, v2 := range pl.V2 {
		list = append(list, v2)
	}

	return list
}

// parseMountinfoLine parses a line from the /proc/[pid]/mountinfo file on
// Linux. The format of the line is specified in section 3.5 of
// https://www.kernel.org/doc/Documentation/filesystems/proc.txt.
func parseMountinfoLine(line string) (mountinfo, error) {
	mount := mountinfo{}

	fields := strings.Fields(line)
	if len(fields) < 10 {
		return mount, fmt.Errorf("invalid mountinfo line, expected at least "+
			"10 fields but got %d from line='%s'", len(fields), line)
	}

	mount.mountpoint = fields[4]

	var seperatorIndex int
	for i, value := range fields {
		if value == "-" {
			seperatorIndex = i
			break
		}
	}
	if fields[seperatorIndex] != "-" {
		return mount, fmt.Errorf("invalid mountinfo line, separator ('-') not "+
			"found in line='%s'", line)
	}

	if len(fields)-seperatorIndex-1 < 3 {
		return mount, fmt.Errorf("invalid mountinfo line, expected at least "+
			"3 fields after seperator but got %d from line='%s'",
			len(fields)-seperatorIndex-1, line)
	}

	fields = fields[seperatorIndex+1:]
	mount.filesystemType = fields[0]
	mount.superOptions = strings.Split(fields[2], ",")
	return mount, nil
}

// SupportedSubsystems returns the subsystems that are supported by the
// kernel. The returned map contains a entry for each subsystem.
func SupportedSubsystems(rootfs resolve.Resolver) (map[string]struct{}, error) {
	cgroups, err := os.Open(rootfs.ResolveHostFS("/proc/cgroups"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCgroupsMissing
		}
		return nil, err
	}
	defer cgroups.Close()

	subsystemSet := map[string]struct{}{}
	sc := bufio.NewScanner(cgroups)
	for sc.Scan() {
		line := sc.Text()

		// Ignore the header.
		if len(line) > 0 && line[0] == '#' {
			continue
		}

		// Parse the cgroup subsystems.
		// Format:  subsys_name    hierarchy      num_cgroups    enabled
		// Example: cpuset         4              1              1
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		// Check the enabled flag.
		if len(fields) > 3 {
			enabled := fields[3]
			if enabled == "0" {
				// Ignore cgroup subsystems that are disabled (via the
				// cgroup_disable kernel command-line boot parameter).
				continue
			}
		}

		subsystem := fields[0]
		subsystemSet[subsystem] = struct{}{}
	}

	return subsystemSet, sc.Err()
}

// SubsystemMountpoints returns the mountpoints for each of the given subsystems.
// The returned map contains the subsystem name as a key and the value is the
// mountpoint.
func SubsystemMountpoints(rootfs resolve.Resolver, subsystems map[string]struct{}) (Mountpoints, error) {

	mountinfo, err := os.Open(rootfs.ResolveHostFS("/proc/self/mountinfo"))
	if err != nil {
		return Mountpoints{}, err
	}
	defer mountinfo.Close()

	mounts := map[string]string{}
	mountInfo := Mountpoints{}
	sc := bufio.NewScanner(mountinfo)
	for sc.Scan() {
		// https://www.kernel.org/doc/Documentation/filesystems/proc.txt
		// Example:
		// 25 21 0:20 / /cgroup/cpu rw,relatime - cgroup cgroup rw,cpu
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		mount, err := parseMountinfoLine(line)
		if err != nil {
			return Mountpoints{}, err
		}

		// if the mountpoint from the subsystem has a different root than ours, it probably belongs to something else.
		if !strings.HasPrefix(mount.mountpoint, rootfs.ResolveHostFS("")) {
			continue
		}

		// cgroupv1 option
		if mount.filesystemType == "cgroup" {
			for _, opt := range mount.superOptions {
				// Sometimes the subsystem name is written like "name=blkio".
				fields := strings.SplitN(opt, "=", 2)
				if len(fields) > 1 {
					opt = fields[1]
				}

				// Test if option is a subsystem name.
				if _, found := subsystems[opt]; found {
					// Add the subsystem mount if it does not already exist.
					if _, exists := mounts[opt]; !exists {
						mounts[opt] = mount.mountpoint
					}
				}
			}
		}

		// V2 option
		if mount.filesystemType == "cgroup2" {
			mountInfo.V2Loc = mount.mountpoint
		}

	}

	mountInfo.V1Mounts = mounts

	return mountInfo, sc.Err()
}

// ProcessCgroupPaths returns the cgroups to which a process belongs and the
// pathname of the cgroup relative to the mountpoint of the subsystem.
func (r Reader) ProcessCgroupPaths(pid int) (PathList, error) {
	cgroupPath := filepath.Join("proc", strconv.Itoa(pid), "cgroup")
	cgroup, err := os.Open(r.rootfsMountpoint.ResolveHostFS(cgroupPath))
	if err != nil {
		return PathList{}, err //return a blank error so other events can use any file not found errors
	}
	defer cgroup.Close()

	cPaths := PathList{V1: map[string]ControllerPath{}, V2: map[string]ControllerPath{}}
	sc := bufio.NewScanner(cgroup)
	for sc.Scan() {
		// http://man7.org/linux/man-pages/man7/cgroups.7.html
		// Format: hierarchy-ID:subsystem-list:cgroup-path
		// Example:
		// 2:cpu:/docker/b29faf21b7eff959f64b4192c34d5d67a707fe8561e9eaa608cb27693fba4242
		line := sc.Text()

		fields := strings.Split(line, ":")
		if len(fields) != 3 {
			continue
		}

		path := fields[2]
		if r.cgroupsHierarchyOverride != "" {
			path = r.cgroupsHierarchyOverride
		}
		// cgroup V2
		// cgroup v2 controllers will always start with this string
		if strings.Contains(line, "0::/") {
			// if you're running inside a container
			// that's operating with a hybrid cgroups config,
			// the containerized process won't see the V2 mount
			// inside /proc/self/mountinfo if docker is using cgroups V1
			// For this very annoying edge case, revert to the hostfs flag
			// If it's not set, warn the user that they've hit this.
			controllerPath := filepath.Join(r.cgroupMountpoints.V2Loc, path)
			if r.cgroupMountpoints.V2Loc == "" && !r.rootfsMountpoint.IsSet() {
				logp.L().Debugf(`PID %d contains a cgroups V2 path (%s) but no V2 mountpoint was found.
This may be because metricbeat is running inside a container on a hybrid system.
To monitor cgroups V2 processess in this way, mount the unified (V2) hierarchy inside
the container as /sys/fs/cgroup/unified and start the system module with the hostfs setting.`, pid, line)
				continue
			} else if r.cgroupMountpoints.V2Loc == "" && r.rootfsMountpoint.IsSet() {
				controllerPath = r.rootfsMountpoint.ResolveHostFS(filepath.Join("/sys/fs/cgroup/unified", path))
			}

			cgpaths, err := ioutil.ReadDir(controllerPath)
			if err != nil {
				return cPaths, errors.Wrapf(err, "error fetching cgroupV2 controllers for cgroup location '%s' and path line '%s'", r.cgroupMountpoints.V2Loc, line)
			}
			// In order to produce the same kind of data for cgroups V1 and V2 controllers,
			// We iterate over the group, and look for controllers, since the V2 unified system doesn't list them under the PID
			for _, singlePath := range cgpaths {
				if strings.Contains(singlePath.Name(), "stat") {
					controllerName := strings.TrimSuffix(singlePath.Name(), ".stat")
					cPaths.V2[controllerName] = ControllerPath{ControllerPath: path, FullPath: controllerPath, IsV2: true}
				}
			}
			// cgroup v1
		} else {
			subsystems := strings.Split(fields[1], ",")
			for _, subsystem := range subsystems {
				fullPath := filepath.Join(r.cgroupMountpoints.V1Mounts[subsystem], path)
				cPaths.V1[subsystem] = ControllerPath{ControllerPath: path, FullPath: fullPath, IsV2: false}
			}
		}
	}

	if sc.Err() != nil {
		return cPaths, errors.Wrap(sc.Err(), "error scanning cgroup file")
	}

	return cPaths, nil
}
