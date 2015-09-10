// +build !windows

package crawler

import (
	"github.com/elastic/filebeat/input"
	"os"
	"syscall"
)

func (p *Prospector) isFileRenamed(file string, info os.FileInfo, missingfiles map[string]os.FileInfo) string {
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
	stat := info.Sys().(*syscall.Stat_t)

	for kf, ki := range p.prospectorinfo {
		if kf == file {
			continue
		}
		ks := ki.Fileinfo.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}

	// Now check the missingfiles
	for kf, ki := range missingfiles {
		ks := ki.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}
	return ""
}

func (p *Prospector) isFileRenamedResumelist(file string, info os.FileInfo, initial map[string]*input.FileState) string {
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
	stat := info.Sys().(*syscall.Stat_t)

	for kf, ki := range initial {
		if kf == file {
			continue
		}
		if stat.Dev == ki.Device && stat.Ino == ki.Inode {
			return kf
		}
	}

	return ""
}
