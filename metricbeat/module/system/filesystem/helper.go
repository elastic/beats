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

// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"bufio"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"runtime"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/system"
	sigar "github.com/elastic/gosigar"
)

// Config stores the metricset-local config
type Config struct {
	IgnoreTypes []string `config:"filesystem.ignore_types"`
}

// FSStat contains filesystem metrics
type FSStat struct {
	sigar.FileSystemUsage
	DevName     string  `json:"device_name"`
	Mount       string  `json:"mount_point"`
	UsedPercent float64 `json:"used_p"`
	SysTypeName string  `json:"type"`
	ctime       time.Time
}

// GetFileSystemList retreves overall filesystem stats
func GetFileSystemList() ([]sigar.FileSystem, error) {
	fss := sigar.FileSystemList{}
	if err := fss.Get(); err != nil {
		return nil, err
	}

	if runtime.GOOS == "windows" {
		// No filtering on Windows
		return fss.List, nil
	}

	return filterFileSystemList(fss.List), nil
}

// filterFileSystemList filters mountpoints to avoid virtual filesystems
// and duplications
func filterFileSystemList(fsList []sigar.FileSystem) []sigar.FileSystem {
	var filtered []sigar.FileSystem
	devices := make(map[string]sigar.FileSystem)
	for _, fs := range fsList {
		// Ignore relative mount points, which are present for example
		// in /proc/mounts on Linux with network namespaces.
		if !filepath.IsAbs(fs.DirName) {
			debugf("Filtering filesystem with relative mountpoint %+v", fs)
			continue
		}

		// Don't do further checks in special devices
		if !filepath.IsAbs(fs.DevName) {
			filtered = append(filtered, fs)
			continue
		}

		// If the device name is a directory, this is a bind mount or nullfs,
		// don't count it as it'd be counting again its parent filesystem.
		devFileInfo, _ := os.Stat(fs.DevName)
		if devFileInfo != nil && devFileInfo.IsDir() {
			continue
		}

		// If a block device is mounted multiple times (e.g. with some bind mounts),
		// store it only once, and use the shorter mount point path.
		if seen, found := devices[fs.DevName]; found {
			if len(fs.DirName) < len(seen.DirName) {
				devices[fs.DevName] = fs
			}
			continue
		}

		devices[fs.DevName] = fs
	}

	for _, fs := range devices {
		filtered = append(filtered, fs)
	}

	return filtered
}

// GetFileSystemStat retreves stats for a single filesystem
func GetFileSystemStat(fs sigar.FileSystem) (*FSStat, error) {
	stat := sigar.FileSystemUsage{}
	if err := stat.Get(fs.DirName); err != nil {
		return nil, err
	}

	var t string
	if runtime.GOOS == "windows" {
		t = fs.TypeName
	} else {
		t = fs.SysTypeName
	}

	filesystem := FSStat{
		FileSystemUsage: stat,
		DevName:         fs.DevName,
		Mount:           fs.DirName,
		SysTypeName:     t,
	}

	return &filesystem, nil
}

// AddFileSystemUsedPercentage adds usage data to the filesystem struct
func AddFileSystemUsedPercentage(f *FSStat) {
	if f.Total == 0 {
		return
	}

	perc := float64(f.Used) / float64(f.Used+f.Avail)
	f.UsedPercent = common.Round(perc, common.DefaultDecimalPlacesCount)
}

// GetFilesystemEvent turns a stat struct into a MapStr
func GetFilesystemEvent(fsStat *FSStat) common.MapStr {
	return common.MapStr{
		"type":        fsStat.SysTypeName,
		"device_name": fsStat.DevName,
		"mount_point": fsStat.Mount,
		"total":       fsStat.Total,
		"free":        fsStat.Free,
		"available":   fsStat.Avail,
		"files":       fsStat.Files,
		"free_files":  fsStat.FreeFiles,
		"used": common.MapStr{
			"pct":   fsStat.UsedPercent,
			"bytes": fsStat.Used,
		},
	}
}

// Predicate is a function predicate for use with filesystems. It returns true
// if the argument matches the predicate.
type Predicate func(*sigar.FileSystem) bool

// Filter returns a filtered list of filesystems. The in parameter
// is used as the backing storage for the returned slice and is therefore
// modified in this operation.
func Filter(in []sigar.FileSystem, p Predicate) []sigar.FileSystem {
	out := in[:0]
	for _, fs := range in {
		if p(&fs) {
			out = append(out, fs)
		}
	}
	return out
}

// BuildTypeFilter returns a predicate that returns false if the given
// filesystem has a type that matches one of the ignoreType values.
func BuildTypeFilter(ignoreType ...string) Predicate {
	return func(fs *sigar.FileSystem) bool {
		for _, fsType := range ignoreType {
			// XXX (andrewkroh): SysTypeName appears to be used for non-Windows
			// and TypeName is used exclusively for Windows.
			if fs.SysTypeName == fsType || fs.TypeName == fsType {
				return false
			}
		}
		return true
	}
}

// DefaultIgnoredTypes tries to guess a sane list of filesystem types that
// could be ignored in the running system
func DefaultIgnoredTypes() (types []string) {
	// If /proc/filesystems exist, default ignored types are all marked
	// as nodev
	fsListFile := path.Join(*system.HostFS, "/proc/filesystems")
	if f, err := os.Open(fsListFile); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.Fields(scanner.Text())
			if len(line) == 2 && line[0] == "nodev" {
				types = append(types, line[1])
			}
		}
	}
	return
}
