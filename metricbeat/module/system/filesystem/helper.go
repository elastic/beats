// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"path/filepath"
	"time"

	"runtime"

	"github.com/elastic/beats/libbeat/common"
	sigar "github.com/elastic/gosigar"
)

type Config struct {
	IgnoreTypes []string `config:"filesystem.ignore_types"`
}

type FileSystemStat struct {
	sigar.FileSystemUsage
	DevName     string  `json:"device_name"`
	Mount       string  `json:"mount_point"`
	UsedPercent float64 `json:"used_p"`
	SysTypeName string  `json:"type"`
	ctime       time.Time
}

func GetFileSystemList() ([]sigar.FileSystem, error) {
	fss := sigar.FileSystemList{}
	if err := fss.Get(); err != nil {
		return nil, err
	}

	// Ignore relative mount points, which are present for example
	// in /proc/mounts on Linux with network namespaces.
	filtered := fss.List[:0]
	for _, fs := range fss.List {
		if filepath.IsAbs(fs.DirName) {
			filtered = append(filtered, fs)
			continue
		}
		debugf("Filtering filesystem with relative mountpoint %+v", fs)
	}
	fss.List = filtered

	return fss.List, nil
}

func GetFileSystemStat(fs sigar.FileSystem) (*FileSystemStat, error) {
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

	filesystem := FileSystemStat{
		FileSystemUsage: stat,
		DevName:         fs.DevName,
		Mount:           fs.DirName,
		SysTypeName:     t,
	}

	return &filesystem, nil
}

func AddFileSystemUsedPercentage(f *FileSystemStat) {
	if f.Total == 0 {
		return
	}

	perc := float64(f.Used) / float64(f.Used+f.Avail)
	f.UsedPercent = common.Round(perc, common.DefaultDecimalPlacesCount)
}

func GetFilesystemEvent(fsStat *FileSystemStat) common.MapStr {
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
