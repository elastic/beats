package cgroup

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	// ErrInvalidFormat indicates a malformed key/value pair on a line.
	ErrInvalidFormat = errors.New("error invalid key/value format")
)

// Parses a cgroup param and returns the key name and value.
func parseCgroupParamKeyValue(t string) (string, uint64, error) {
	parts := strings.Fields(t)
	if len(parts) != 2 {
		return "", 0, ErrInvalidFormat
	}

	value, err := parseUint([]byte(parts[1]))
	if err != nil {
		return "", 0, fmt.Errorf("unable to convert param value (%q) to uint64: %v", parts[1], err)
	}

	return parts[0], value, nil
}

// parseUintFromFile reads a single uint value from a file.
func parseUintFromFile(path ...string) (uint64, error) {
	value, err := ioutil.ReadFile(filepath.Join(path...))
	if err != nil {
		return 0, err
	}

	return parseUint(value)
}

// parseUint reads a single uint value. It will trip any whitespace before
// attempting to parse string. If the value is negative it will return 0.
func parseUint(value []byte) (uint64, error) {
	strValue := string(bytes.TrimSpace(value))
	uintValue, err := strconv.ParseUint(strValue, 10, 64)
	if err != nil {
		// Munge negative values to 0.
		intValue, intErr := strconv.ParseInt(strValue, 10, 64)
		if intErr == nil && intValue < 0 {
			return 0, nil
		} else if intErr != nil && intErr.(*strconv.NumError).Err == strconv.ErrRange && intValue < 0 {
			return 0, nil
		}

		return 0, err
	}

	return uintValue, nil
}

// SupportedSubsystems returns the subsystems that are supported by the
// kernel. The returned map contains a entry for each subsystem.
func SupportedSubsystems(rootfsMountpoint string) (map[string]struct{}, error) {
	if rootfsMountpoint == "" {
		rootfsMountpoint = "/"
	}

	cgroups, err := os.Open(filepath.Join(rootfsMountpoint, "proc", "cgroups"))
	if err != nil {
		return nil, err
	}

	subsystemSet := map[string]struct{}{}
	sc := bufio.NewScanner(cgroups)
	for sc.Scan() {
		line := sc.Text()

		// Ignore the header.
		if len(line) > 0 && line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		subsystem := fields[0]
		subsystemSet[subsystem] = struct{}{}
	}

	return subsystemSet, nil
}

// SubsystemMountpoints returns the mountpoints for each of the given subsystems.
// The returned map contains the subsystem name as a key and the value is the
// mountpoint.
func SubsystemMountpoints(rootfsMountpoint string, subsystems map[string]struct{}) (map[string]string, error) {
	if rootfsMountpoint == "" {
		rootfsMountpoint = "/"
	}

	mountinfo, err := os.Open(filepath.Join(rootfsMountpoint, "proc", "self", "mountinfo"))
	if err != nil {
		return nil, err
	}

	mounts := map[string]string{}
	sc := bufio.NewScanner(mountinfo)
	for sc.Scan() {
		// https://www.kernel.org/doc/Documentation/filesystems/proc.txt
		// Example:
		// 25 21 0:20 / /cgroup/cpu rw,relatime - cgroup cgroup rw,cpu
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) != 10 || fields[6] != "-" {
			return nil, fmt.Errorf("invalid mountinfo data")
		}

		if fields[7] != "cgroup" {
			continue
		}

		mountpoint := fields[4]
		opts := strings.Split(fields[9], ",")
		for _, opt := range opts {
			// XXX(akroh): May need to handle options prepended with 'name='.
			// Test if option is a subsystem name.
			if _, found := subsystems[opt]; found {
				// Add the subsystem mount if it does not already exist.
				if _, exists := mounts[opt]; !exists {
					mounts[opt] = mountpoint
				}
			}
		}
	}

	return mounts, nil
}

// ProcessCgroupPaths returns the cgroups to which a process belongs and the
// pathname of the cgroup relative to the mountpoint of the subsystem.
func ProcessCgroupPaths(rootfsMountpoint string, pid int) (map[string]string, error) {
	if rootfsMountpoint == "" {
		rootfsMountpoint = "/"
	}

	cgroup, err := os.Open(filepath.Join(rootfsMountpoint, "proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return nil, err
	}

	paths := map[string]string{}
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
		subsystems := strings.Split(fields[1], ",")
		for _, subsystem := range subsystems {
			paths[subsystem] = path
		}
	}

	return paths, nil
}
