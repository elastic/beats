// +build !windows

package crawler

import (
	"github.com/elastic/filebeat/input"
	"os"
	"syscall"
)

// Check if the given file was renamed. If file is known but with different path,
// renamed will be set true and previous will be set to the previously known file path.
// Otherwise renamed will be false.
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
	for kf, ki := range p.missingFiles {
		ks := ki.Sys().(*syscall.Stat_t)
		if stat.Dev == ks.Dev && stat.Ino == ks.Ino {
			return kf
		}
	}

	// NOTE(ruflin): should instead an error be returned if not previous file?
	return ""
}

// getPreviousFile checks in the registrar if there is the newFile already exist with a different name
// In case an old file is found, the path to the file is returned
func (r *Registrar) getPreviousFile(newFilePath string, newFileInfo os.FileInfo) string {

	// As the state of the old file cannot be fetched anymore based on the path, IsSameFile does not work
	newState := newFileInfo.Sys().(*syscall.Stat_t)

	for oldFilePath, oldState := range r.State {

		// Skipping when path the same
		if oldFilePath == newFilePath {
			continue
		}

		// Compare Inode and device
		if newState.Dev == oldState.Device && newState.Ino == oldState.Inode {
			return oldFilePath
		}
	}

	return ""
}
