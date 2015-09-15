// +build !windows

package crawler

import (
	"os"
	"syscall"
)

// Checks if the given file was renamed. Returns the previous file on success, otherwise ""
func (p *Prospector) isFileRenamed(file string, info os.FileInfo) string {
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
	stat := info.Sys().(*syscall.Stat_t)

	for kf, ki := range p.prospectorList {
		if kf == file {
			continue
		}
		ks := ki.Fileinfo.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}

	// Now check the missingfiles
	for kf, ki := range *p.missingFiles {
		ks := ki.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}

	// NOTE(ruflin): should instead an error be returned if not previous file?
	return ""
}

func (c *Crawler) isFileRenamedResumelist(file string, info os.FileInfo) string {
	// NOTE(driskell): What about using golang's func os.SameFile(fi1, fi2 FileInfo) bool instead?
	stat := info.Sys().(*syscall.Stat_t)

	for kf, ki := range c.Files {
		if kf == file {
			continue
		}
		if stat.Dev == ki.Device && stat.Ino == ki.Inode {
			return kf
		}
	}

	return ""
}
