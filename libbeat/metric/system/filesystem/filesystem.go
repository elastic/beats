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

//go:build darwin || freebsd || linux || openbsd || windows
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
)

//FSStat carries the metadata for a given filesystem
type FSStat struct {
	Directory string   `struct:"mount_point,omitempty"`
	Device    string   `struct:"device_name,omitempty"`
	Type      string   `struct:"type,omitempty"`
	Options   string   `struct:"options,omitempty"`
	Flags     opt.Uint `struct:"flags,omitempty"`
	// metrics
	Total     opt.Uint `struct:"total,omitempty"`
	Free      opt.Uint `struct:"free,omitempty"`
	Avail     opt.Uint `struct:"available,omitempty"`
	Used      UsedVals `struct:"used,omitempty"`
	Files     opt.Uint `struct:"files,omitempty"`
	FreeFiles opt.Uint `struct:"free_files,omitempty"`
}

// UsedVals wraps the `used` disk metrics
type UsedVals struct {
	Pct   opt.Float `struct:"pct,omitempty"`
	Bytes opt.Uint  `struct:"bytes,omitempty"`
}

// IsZero implements the IsZero interface for go-structform
func (u UsedVals) IsZero() bool {
	return u.Pct.IsZero() && u.Bytes.IsZero()
}

var debugf = logp.MakeDebug("libbeat.filesystem")

func getFSPath(hostfs resolve.Resolver) string {
	// Do a little work to make sure we don't break anything.
	// This code would previously just blindly just search for /etc/mtab
	// This wasn't available on certain containerized workflows,
	// So default to mtab's symlink of /proc/self/mounts
	// However, I'm a little skeptical of `self` inside containers,
	// so if hostfs is set, use /hostfs/proc/mounts
	if hostfs.IsSet() {
		return hostfs.ResolveHostFS("/proc/mounts")
	}
	return hostfs.ResolveHostFS("/proc/self/mounts")

}

// GetFilesystems returns a filesystem list filtered by the callback function
func GetFilesystems(hostfs resolve.Resolver, filter func(FSStat) bool) ([]FSStat, error) {
	fs := getFSPath(hostfs)

	if filter == nil {
		filter = buildDefaultFilters(hostfs)
	}

	//combine user-supplied and built-in filters
	filterFunc := func(fs FSStat) bool {
		return avoidFileSystem(fs) && filter(fs)
	}

	mounts, err := parseMounts(fs, filterFunc) //nolint: typecheck // I don't think the linter likes platform-specific code
	if err != nil {
		return nil, fmt.Errorf("error reading mounts: %w", err)
	}

	return filterDuplicates(mounts), nil

}

// Fill out computed stats after the platform-specific code fetches metrics from the OS
func (fs *FSStat) fillMetrics() {
	fs.Used.Bytes = fs.Total.SubtractOrNone(fs.Free)

	// I'm not sure why this does Used + avail instead of total, but I'm too afraid to change it
	percTotal := fs.Used.Bytes.ValueOr(0) + fs.Avail.ValueOr(0)
	if percTotal == 0 {
		return
	}

	perc := float64(fs.Used.Bytes.ValueOr(0)) / float64(percTotal)
	fs.Used.Pct = opt.FloatWith(common.Round(perc, common.DefaultDecimalPlacesCount))
}

// DefaultIgnoredTypes tries to guess a sane list of filesystem types that
// could be ignored in the running system
func DefaultIgnoredTypes(sys resolve.Resolver) []string {
	// If /proc/filesystems exist, default ignored types are all marked
	// as nodev
	types := []string{}
	fsListFile := sys.ResolveHostFS("/proc/filesystems")
	if f, err := os.Open(fsListFile); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.Fields(scanner.Text())
			if len(line) == 2 && line[0] == "nodev" {
				types = append(types, line[1])
			}
		}
	}
	return types
}

// BuildFilterWithList returns a filesystem filter with the given list of FS types
func BuildFilterWithList(ignored []string) func(FSStat) bool {
	return func(fs FSStat) bool {
		for _, fsType := range ignored {
			// XXX (andrewkroh): SystemType appears to be used for non-Windows
			// and Type is used exclusively for Windows.
			if fs.Type == fsType {
				return false
			}
		}
		return true
	}
}

func buildDefaultFilters(hostfs resolve.Resolver) func(FSStat) bool {
	ignoreType := DefaultIgnoredTypes(hostfs)
	return BuildFilterWithList(ignoreType)
}

// If a block device is mounted multiple times (e.g. with some bind mounts),
// store it only once, and use the shorter mount point path.
func filterDuplicates(fsList []FSStat) []FSStat {
	devices := make(map[string]FSStat)
	var filtered []FSStat

	for _, fs := range fsList {
		// Don't do any further checks on block devices
		if !filepath.IsAbs(fs.Device) {
			filtered = append(filtered, fs)
			continue
		}
		if seen, found := devices[fs.Device]; found {
			if len(fs.Directory) < len(seen.Directory) {
				devices[fs.Device] = fs
			}
			continue
		}
		devices[fs.Device] = fs
	}

	for _, fs := range devices {
		filtered = append(filtered, fs)
	}

	return filtered
}

func avoidFileSystem(fs FSStat) bool {
	// Ignore relative mount points, which are present for example
	// in /proc/mounts on Linux with network namespaces.
	if !filepath.IsAbs(fs.Directory) {
		debugf("Filtering filesystem with relative mountpoint %+v", fs)
		return false
	}

	// Don't do further checks in special devices
	if !filepath.IsAbs(fs.Device) {
		return true
	}

	// If the device name is a directory, this is a bind mount or nullfs,
	// don't count it as it'd be counting again its parent filesystem.
	devFileInfo, _ := os.Stat(fs.Device)
	if devFileInfo != nil && devFileInfo.IsDir() {
		return false
	}
	return true
}
