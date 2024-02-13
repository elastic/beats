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

//go:build linux

package kprobes

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Used to make the mount functions thread safe
var mountMutex sync.Mutex

// mount contains information for a specific mounted filesystem.
//
//	Path           	- Absolute path where the directory is mounted
//	FilesystemType 	- Type of the mounted filesystem, e.g. "ext4"
//	Device         	- Device for filesystem (empty string if we cannot find one)
//	DeviceMajor   	- Device major number of the filesystem.  This is set even if
//					  Device isn't, since all filesystems have a device
//					  number assigned by the kernel, even pseudo-filesystems.
//	DeviceMinor   	- Device minor number of the filesystem.  This is set even if
//					  Device isn't, since all filesystems have a device
//					  number assigned by the kernel, even pseudo-filesystems.
//	Subtree        	- The mounted subtree of the filesystem.  This is usually
//					  "/", meaning that the entire filesystem is mounted, but
//					  it can differ for bind mounts.
//	ReadOnly       	- True if this is a read-only mount
type mount struct {
	Path           string
	FilesystemType string
	DeviceMajor    uint32
	DeviceMinor    uint32
	Subtree        string
	ReadOnly       bool
}

// mountPoints allows mounts to be sorted by Path length.
type mountPoints []*mount

func (p mountPoints) Len() int      { return len(p) }
func (p mountPoints) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p mountPoints) Less(i, j int) bool {
	if len(p[i].Path) == len(p[j].Path) {
		return p[i].Path > p[j].Path
	}

	return len(p[i].Path) > len(p[j].Path)
}

// getMountByPath returns the mount point that matches the given path.
//
// The path parameter specifies the path to search for a matching mount point.
// It should not be empty.
//
// The function returns a pointer to a mount struct if a matching mount point is found,
// otherwise it returns nil.
func (p mountPoints) getMountByPath(path string) *mount {
	if path == "" {
		return nil
	}

	// Remove trailing slash if it not root /
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	for _, mount := range p {
		mountPath := mount.Path
		if strings.HasPrefix(path, mountPath) {
			return mount
		}
	}

	return nil
}

// Unescape octal-encoded escape sequences in a string from the mountinfo file.
// The kernel encodes the ' ', '\t', '\n', and '\\' bytes this way.  This
// function exactly inverts what the kernel does, including by preserving
// invalid UTF-8.
func unescapeString(str string) string {
	var sb strings.Builder
	for i := 0; i < len(str); i++ {
		b := str[i]
		if b == '\\' && i+3 < len(str) {
			if parsed, err := strconv.ParseInt(str[i+1:i+4], 8, 8); err == nil {
				b = uint8(parsed)
				i += 3
			}
		}
		sb.WriteByte(b)
	}
	return sb.String()
}

// Parse one line of /proc/self/mountinfo.
//
// The line contains the following space-separated fields:
//
//	[0] mount ID
//	[1] parent ID
//	[2] major:minor
//	[3] root
//	[4] mount point
//	[5] mount options
//	[6...n-1] optional field(s)
//	[n] separator
//	[n+1] filesystem type
//	[n+2] mount source
//	[n+3] super options
//
// For more details, see https://www.kernel.org/doc/Documentation/filesystems/proc.txt
func parseMountInfoLine(line string) (*mount, error) {
	fields := strings.Split(line, " ")
	if len(fields) < 10 {
		return nil, nil
	}

	// Count the optional fields.  In case new fields are appended later,
	// don't simply assume that n == len(fields) - 4.
	n := 6
	for fields[n] != "-" {
		n++
		if n >= len(fields) {
			return nil, nil
		}
	}
	if n+3 >= len(fields) {
		return nil, nil
	}

	mnt := &mount{}
	var err error
	mnt.DeviceMajor, mnt.DeviceMinor, err = newDeviceMajorMinorFromString(fields[2])
	if err != nil {
		return nil, err
	}
	mnt.Subtree = unescapeString(fields[3])
	mnt.Path = unescapeString(fields[4])
	for _, opt := range strings.Split(fields[5], ",") {
		if opt == "ro" {
			mnt.ReadOnly = true
		}
	}
	mnt.FilesystemType = unescapeString(fields[n+1])
	return mnt, nil
}

// readMountInfo reads mount information from the given input reader and returns
// a list of mount points and an error. Each mount point is represented by a mount
// struct containing information about the mount.
func readMountInfo(r io.Reader) (mountPoints, error) {
	seenMountsByPath := make(map[string]*mount)
	var mPoints mountPoints //nolint:prealloc //can't be preallocated as the number of lines is unknown before scan

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		mnt, err := parseMountInfoLine(line)
		if err != nil {
			return nil, err
		}

		if mnt == nil {
			continue
		}

		_, exists := seenMountsByPath[mnt.Path]
		if exists {
			// duplicate mountpoint entries have been observed for
			// /proc/sys/fs/binfmt_misc
			continue
		}

		mPoints = append(mPoints, mnt)
		// Note this overrides the info if we have seen the mountpoint
		// earlier in the file. This is correct behavior because the
		// mountpoints are listed in mount order.
		seenMountsByPath[mnt.Path] = mnt
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	sort.Sort(mPoints)

	return mPoints, nil
}

// getAllMountPoints populates the mount mappings by parsing /proc/self/mountinfo.
func getAllMountPoints() (mountPoints, error) {
	mountMutex.Lock()
	defer mountMutex.Unlock()

	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readMountInfo(file)
}

// newDeviceMajorMinorFromString generates a new device major and minor numbers from a given string.
func newDeviceMajorMinorFromString(str string) (uint32, uint32, error) {
	var major, minor uint32
	if count, _ := fmt.Sscanf(str, "%d:%d", &major, &minor); count != 2 {
		return 0, 0, fmt.Errorf("invalid device number string %q", str)
	}
	return major, minor, nil
}
