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

// mountinfo represents a subset of the fields containing /proc/[pid]/mountinfo.
type mountinfo struct {
	mountpoint     string
	filesystemType string
	superOptions   []string
}

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
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		mount, err := parseMountinfoLine(line)
		if err != nil {
			return nil, err
		}

		if mount.filesystemType != "cgroup" {
			continue
		}

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
