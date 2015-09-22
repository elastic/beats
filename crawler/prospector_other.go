// +build !windows

package crawler

import (
	"os"
	"syscall"
)

// Check if the given file was renamed. If file is known but with different path,
// renamed will be set true and previous will be set to the previously known file path.
// Otherwise renamed will be false.
func (p *Prospector) getPreviousFile(file string, info os.FileInfo) string {
	// TODO: To implement this properly the file state of the previous file is required.
	// For more details see how crawler implements it
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
	for kf, ki := range p.missingFiles {
		ks := ki.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}

	// NOTE(ruflin): should instead an error be returned if not previous file?
	return ""
}
