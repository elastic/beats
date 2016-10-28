// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
	err := fss.Get()
	if err != nil {
		return nil, err
	}

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
	f.UsedPercent = system.Round(perc, .5, 4)
}

func CollectFileSystemStats(fss []sigar.FileSystem) []common.MapStr {
	events := make([]common.MapStr, 0, len(fss))
	for _, fs := range fss {
		fsStat, err := GetFileSystemStat(fs)
		if err != nil {
			logp.Debug("system", "Skip filesystem %d: %v", fsStat, err)
			continue
		}
		AddFileSystemUsedPercentage(fsStat)

		event := common.MapStr{
			"@timestamp": common.Time(time.Now()),
			"type":       "filesystem",
			"fs":         GetFilesystemEvent(fsStat),
		}
		events = append(events, event)
	}
	return events
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

func GetFileSystemStats() ([]common.MapStr, error) {
	fss, err := GetFileSystemList()
	if err != nil {
		logp.Warn("Getting filesystem list: %v", err)
		return nil, err
	}

	return CollectFileSystemStats(fss), nil
}
