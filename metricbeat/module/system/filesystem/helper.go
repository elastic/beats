// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"path/filepath"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/module/system"
	sigar "github.com/elastic/gosigar"
)

type FileSystemStat struct {
	sigar.FileSystemUsage
	DevName     string  `json:"device_name"`
	Mount       string  `json:"mount_point"`
	UsedPercent float64 `json:"used_p"`
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

	filesystem := FileSystemStat{
		FileSystemUsage: stat,
		DevName:         fs.DevName,
		Mount:           fs.DirName,
	}

	return &filesystem, nil
}

func AddFileSystemUsedPercentage(f *FileSystemStat) {
	if f.Total == 0 {
		return
	}

	perc := float64(f.Used) / float64(f.Total)
	f.UsedPercent = system.Round(perc)
}

func GetFilesystemEvent(fsStat *FileSystemStat) common.MapStr {
	return common.MapStr{
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
