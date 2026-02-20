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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// cgroupCntainerCache is a performance helper used for
// cases where we're in a container and we need to fetch our cgroup
// path from the host system. We want to cache these results, since traversing
// /hostfs/sys/fs/cgroup is a bit intensive.
// This value is also unlikely to change more than once.
// see guessContainerCgroupPath() below for more context
type cgroupContainerCache struct {
	mut    sync.Mutex
	cgPath string
}

func (cgc *cgroupContainerCache) get() string {
	cgc.mut.Lock()
	defer cgc.mut.Unlock()
	return cgc.cgPath
}

func (cgc *cgroupContainerCache) set(update string) {
	cgc.mut.Lock()
	defer cgc.mut.Unlock()
	cgc.cgPath = update
}

var cgroupContainerPath *cgroupContainerCache

func init() {
	cgroupContainerPath = &cgroupContainerCache{cgPath: ""}
}

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
	V1Mounts               map[string]string
	V2Loc                  string
	ContainerizedRootMount string
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

// wrapper that allows us to bypass isCgroupNSPrivate() for testing
var cgroupNSStateFetch = isCgroupNSPrivate

// Flatten combines the V1 and V2 cgroups in cases where we don't need a map with keys
func (pl PathList) Flatten() []ControllerPath {
	list := make([]ControllerPath, 0, len(pl.V1)+len(pl.V2))
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

	var separatorIndex int
	for i, value := range fields {
		if value == "-" {
			separatorIndex = i
			break
		}
	}
	if fields[separatorIndex] != "-" {
		return mount, fmt.Errorf("invalid mountinfo line, separator ('-') not "+
			"found in line='%s'", line)
	}

	if len(fields)-separatorIndex-1 < 3 {
		return mount, fmt.Errorf("invalid mountinfo line, expected at least "+
			"3 fields after separator but got %d from line='%s'",
			len(fields)-separatorIndex-1, line)
	}

	fields = fields[separatorIndex+1:]
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
func SubsystemMountpoints(rootfs resolve.Resolver, subsystems map[string]struct{}, logger *logp.Logger) (Mountpoints, error) {
	// TODO: will we run into mount namespace issues if we use /proc/self/mountinfo?
	mountinfo, err := os.Open(rootfs.ResolveHostFS("/proc/self/mountinfo"))
	if err != nil {
		return Mountpoints{}, err
	}
	defer mountinfo.Close()

	mounts := map[string]string{}
	mountInfo := Mountpoints{}
	sc := bufio.NewScanner(mountinfo)
	possibleV2Paths := []string{}
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
			possibleV2Paths = append(possibleV2Paths, mount.mountpoint)
		}

	}

	mountInfo.V2Loc = getProperV2Paths(rootfs, possibleV2Paths, logger)
	mountInfo.V1Mounts = mounts

	// we only care about a contanerized root path if we're trying to monitor a host system
	// from inside a container
	// This logic helps us proper fetch the cgroup path when we're running inside a container
	// with a private namespace
	if mountInfo.V2Loc != "" && rootfs.IsSet() && cgroupNSStateFetch(logger) {
		mountInfo.ContainerizedRootMount, err = guessContainerCgroupPath(mountInfo.V2Loc, os.Getpid())
		// treat this as a non-fatal error. If we end up needing this value, the lookups will fail down the line
		if err != nil {
			logger.Debugf("Non-fatal error fetching cgroup path inside container: %v", err)
		}
	}

	return mountInfo, sc.Err()
}

// isCgroupNSHost returns true if we're running inside a container with a
// private cgroup namespace. Will return true if we're in a public namespace, or there's an error
// Note that this function only makes sense *inside* a container. Outside it will probably always return false.
func isCgroupNSPrivate(logger *logp.Logger) bool {
	// we don't care about hostfs here, since we're just concerned about
	// detecting the environment we're running under.
	raw, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		logger.Debugf("error reading /proc/self/cgroup to detect docker namespace settings: %w", err)
		return false
	}
	// if we have a path of just "/" that means we're in our own private namespace
	// if it's something else, we're probably in a host namespace
	segments := strings.Split(strings.TrimSpace(string(raw)), ":")
	return segments[len(segments)-1] == "/"

}

// tries to find the cgroup path for the currently-running container,
// assuming we are running in a container.
// see https://docs.docker.com/config/containers/runmetrics/#find-the-cgroup-for-a-given-container
// We need to know the root cgroup we're running under, as
// for monitoring a v2 system with a private namespace, we'll get relative paths
// for the cgroup of a pid, see https://github.com/elastic/elastic-agent-system-metrics/issues/139
// This will only work on v2 cgroups, I haven't run into this on a system with cgroups v1 yet;
// not sure if docker namespacing behaves the same.
func guessContainerCgroupPath(v2Loc string, OurPid int) (string, error) {
	// check the cache first
	if cachePath := cgroupContainerPath.get(); cachePath != "" {
		// check the validity of the cache
		rawFile, err := os.ReadFile(filepath.Join(v2Loc, cachePath, "cgroup.procs"))
		// if we get a read error, assume the cache is invalid, move on
		if err == nil {
			if foundMatchingPidInProcsFile(OurPid, string(rawFile)) {
				return cachePath, nil
			}
		}
	}
	// pattern:
	// if in a private cgroup namespace,
	// traverse over the root cgroup path, look for *.procs files
	// go through all of the *.procs files until we have one that contains our pid
	// that path is our cgroup

	foundCgroupPath := ""
	err := filepath.WalkDir(v2Loc, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), "procs") {
			pidfile, err := os.ReadFile(path) //nolint: nilerr // we can get lots of weird permissions errors here, so don't fail on an error
			if err != nil {
				return nil //nolint: nilerr // we can get lots of weird permissions errors here, so don't fail on an error
			}
			if foundMatchingPidInProcsFile(OurPid, string(pidfile)) {
				foundCgroupPath = path
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error traversing paths to find cgroup: %w", err)
	}

	if foundCgroupPath == "" {
		return "", nil
	}
	// strip to cgroup path
	cgroupDir := filepath.Dir(foundCgroupPath)
	relativePath := strings.TrimPrefix(cgroupDir, v2Loc)
	cgroupContainerPath.set(relativePath)
	return relativePath, nil
}

// foundMatchingPidInProcsFile is a helper for guessContainerCgroupPath
// that tells us if we have a matching process in a cgroup.procs file
func foundMatchingPidInProcsFile(ourPid int, fileData string) bool {
	for rawPid := range strings.SplitSeq(fileData, "\n") {
		if len(rawPid) == 0 {
			continue
		}
		pidInt, err := strconv.ParseInt(strings.TrimSpace(rawPid), 10, 64)
		if err != nil {
			return false
		}
		if pidInt == int64(ourPid) {
			return true
		}
	}

	return false
}

// when we're reading from a host mountinfo path from inside a container
// (i.e) `/hostfs/proc/self/mountinfo`, we can get a set of cgroup2 mountpoints like this:
// 1718 1686 0:26 / /hostfs/sys/fs/cgroup rw,nosuid,nodev,noexec,relatime master:4 - cgroup2 cgroup2 rw,seclabel
// 1771 1770 0:26 / /hostfs/var/lib/docker/overlay2/1b570230fa3ec3679e354b0c219757c739f91d774ebc02174106488606549da0/merged/sys/fs/cgroup ro,nosuid,nodev,noexec,relatime - cgroup2 cgroup rw,seclabel
// That latter mountpoint, just a link to the overlayfs, is almost guaranteed to throw a permissions error
// try to sort out the mountpoints, and use the correct one
func getProperV2Paths(rootfs resolve.Resolver, possibleV2Paths []string, logger *logp.Logger) string {
	if len(possibleV2Paths) > 1 {
		// try to sort out anything that looks like a docker fs
		filteredPaths := []string{}
		for _, path := range possibleV2Paths {
			if strings.Contains(path, "overlay2") {
				continue
			}
			filteredPaths = append(filteredPaths, path)
		}
		// if we have no correct paths, give up and use the last one
		// the "last one" ideom preserves behavior before we got more clever with looking for the V2 paths
		if len(filteredPaths) == 0 {
			usePath := possibleV2Paths[len(possibleV2Paths)-1]
			logger.Debugf("could not find correct cgroupv2 path, reverting to path that may produce errors: %s", usePath)
			return usePath
		}

		// if we're using an alternate hostfs, assume we want to monitor the host system, from inside a container
		// and use that path
		if rootfs.IsSet() {
			root := rootfs.ResolveHostFS("")
			hostFSPaths := []string{}
			for _, path := range filteredPaths {
				if strings.Contains(path, root) {
					hostFSPaths = append(hostFSPaths, path)
				}
			}
			// return the last path
			if len(hostFSPaths) > 0 {
				return hostFSPaths[len(hostFSPaths)-1]
			} else {
				usePath := filteredPaths[len(filteredPaths)-1]
				logger.Debugf("An alternate hostfs was specified, but could not find any cgroup mountpoints that contain a hostfs. Using: %s", usePath)
				return usePath
			}
		} else {
			// if no hosfs is set, just use the last element
			return filteredPaths[len(filteredPaths)-1]
		}

	} else if len(possibleV2Paths) == 1 {
		return possibleV2Paths[0]
	}

	return ""
}

// ProcessCgroupPaths returns the cgroups to which a process belongs and the
// pathname of the cgroup relative to the mountpoint of the subsystem.
func (r *Reader) ProcessCgroupPaths(pid int) (PathList, error) {
	cgroupPath := filepath.Join("proc", strconv.Itoa(pid), "cgroup")
	cgroup, err := os.Open(r.rootfsMountpoint.ResolveHostFS(cgroupPath))
	if err != nil {
		return PathList{}, err //return a blank error so other events can use any file not found errors
	}
	defer cgroup.Close()

	version, err := r.CgroupsVersion(pid)
	if err != nil {
		return PathList{}, fmt.Errorf("error finding cgroup version for pid %d: %w", pid, err)
	}

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

		//on newer docker versions (1.41+?), docker will do  namespacing with cgroups
		// such that we'll get a cgroup path like `0::/../../user.slice/user-1000.slice/session-520.scope`
		// `man 7 cgroups` says the following about the path field in the `cgroup` file (emphasis mine):
		//
		// This field contains the pathname of the control group
		// in the hierarchy to which the process belongs.  This
		// pathname is **relative to the mount point of the hierarchy**.
		//
		// However, when we try to append something like `/../..` to another path, we obviously blow things up.
		// we need to use the absolute path of the container cgroup
		if cgroupNSStateFetch(r.logger) && r.rootfsMountpoint.IsSet() {
			if r.cgroupMountpoints.ContainerizedRootMount == "" {
				r.logger.Debugf("cgroup for process %d contains a relative cgroup path (%s), but we were not able to find a root cgroup. Cgroup monitoring for this PID may be incomplete",
					pid, path)
			} else {
				r.logger.Debugf("using root mount %s and path %s", r.cgroupMountpoints.ContainerizedRootMount, path)
				path = filepath.Join(r.cgroupMountpoints.ContainerizedRootMount, path)
			}
		}

		// cgroup V2
		// cgroup v2 controllers will always start with this string
		if strings.HasPrefix(line, "0::/") {
			// if you're running inside a container
			// that's operating with a hybrid cgroups config,
			// the containerized process won't see the V2 mount
			// inside /proc/self/mountinfo if docker is using cgroups V1
			// For this very annoying edge case, revert to the hostfs flag
			// If it's not set, warn the user that they've hit this.

			// we skip reading paths in case there are cgroups V1 controllers, we are at the cgroup V2 root and the cgroup V2 mount is not available
			// instead of returning an error because we don't want to break V1 metric collection for misconfigured hybrid systems that have only
			// a cgroup V2 root but don't have any other controllers. This case happens when cgroup V2 FS is mounted at a special location but not used
			if version == CgroupsV1 && line == "0::/" && r.cgroupMountpoints.V2Loc == "" {
				continue
			}

			controllerPath := filepath.Join(r.cgroupMountpoints.V2Loc, path)
			if r.cgroupMountpoints.V2Loc == "" && !r.rootfsMountpoint.IsSet() {
				r.logger.Debugf(`PID %d contains a cgroups V2 path (%s) but no V2 mountpoint was found.
This may be because metricbeat is running inside a container on a hybrid system.
To monitor cgroups V2 processess in this way, mount the unified (V2) hierarchy inside
the container as /sys/fs/cgroup/unified and start the system module with the hostfs setting.`, pid, line)
				continue
			} else if r.cgroupMountpoints.V2Loc == "" && r.rootfsMountpoint.IsSet() {
				controllerPath = r.rootfsMountpoint.ResolveHostFS(filepath.Join("/sys/fs/cgroup/unified", path))
			}

			// Check if there is an entry for controllerPath already cached.
			r.v2ControllerPathCache.Lock()
			cacheEntry, ok := r.v2ControllerPathCache.cache[controllerPath]
			if ok {
				// If the cached entry for controllerPath is not older than 5 minutes,
				// return the cached entry.
				if time.Since(cacheEntry.added) < 5*time.Minute {
					cPaths.V2 = cacheEntry.pathList.V2
					r.v2ControllerPathCache.Unlock()
					continue
				}

				// Consider the existing entry for controllerPath invalid, as it is
				// older than 5 minutes.
				delete(r.v2ControllerPathCache.cache, controllerPath)
			}
			r.v2ControllerPathCache.Unlock()

			cgpaths, err := os.ReadDir(controllerPath)
			if err != nil {
				return cPaths, fmt.Errorf("error fetching cgroupV2 controllers for cgroup location '%s' and path line '%s': %w", r.cgroupMountpoints.V2Loc, line, err)
			}
			// In order to produce the same kind of data for cgroups V1 and V2 controllers,
			// We iterate over the group, and look for controllers, since the V2 unified system doesn't list them under the PID
			for _, singlePath := range cgpaths {
				if strings.Contains(singlePath.Name(), "stat") {
					controllerName := strings.TrimSuffix(singlePath.Name(), ".stat")
					cPaths.V2[controllerName] = ControllerPath{ControllerPath: path, FullPath: controllerPath, IsV2: true}
				}
			}
			r.v2ControllerPathCache.Lock()
			r.v2ControllerPathCache.cache[controllerPath] = pathListWithTime{
				added:    time.Now(),
				pathList: cPaths,
			}
			r.v2ControllerPathCache.Unlock()
			// cgroup v1
		} else {
			subsystems := strings.SplitSeq(fields[1], ",")
			for subsystem := range subsystems {
				fullPath := filepath.Join(r.cgroupMountpoints.V1Mounts[subsystem], path)
				cPaths.V1[subsystem] = ControllerPath{ControllerPath: path, FullPath: fullPath, IsV2: false}
			}
		}
	}

	if sc.Err() != nil {
		return cPaths, fmt.Errorf("error scanning cgroup file: %w", sc.Err())
	}

	return cPaths, nil
}
