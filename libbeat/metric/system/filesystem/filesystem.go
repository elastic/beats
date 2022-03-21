//go:build darwin || freebsd || linux || openbsd || windows
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/libbeat/opt"
	"github.com/pkg/errors"
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

	mounts, err := parseMounts(fs, filterFunc)
	if err != nil {
		return nil, errors.Wrap(err, "error reading mounts")
	}

	return filterDuplicates(mounts), nil

}

// DefaultIgnoredTypes tries to guess a sane list of filesystem types that
// could be ignored in the running system
func DefaultIgnoredTypes(sys resolve.Resolver) (types []string) {
	// If /proc/filesystems exist, default ignored types are all marked
	// as nodev
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
	return
}

func buildDefaultFilters(hostfs resolve.Resolver) func(FSStat) bool {
	ignoreType := DefaultIgnoredTypes(hostfs)
	return func(fs FSStat) bool {
		for _, fsType := range ignoreType {
			// XXX (andrewkroh): SystemType appears to be used for non-Windows
			// and Type is used exclusively for Windows.
			if fs.Type == fsType {
				return false
			}
		}
		return true
	}
}

// If a block device is mounted multiple times (e.g. with some bind mounts),
// store it only once, and use the shorter mount point path.
func filterDuplicates(fsList []FSStat) []FSStat {
	devices := make(map[string]FSStat)
	var filtered []FSStat

	for _, fs := range fsList {
		if seen, found := devices[fs.Device]; found {
			if len(fs.Directory) < len(seen.Directory) {
				devices[fs.Device] = fs
			}
			continue
		} else {
			devices[fs.Device] = fs
		}
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
