package file

import (
	"fmt"
	"os"
	"reflect"
	"syscall"

	"github.com/elastic/beats/libbeat/logp"
)

type StateOS struct {
	IdxHi uint64 `json:"idxhi,"`
	IdxLo uint64 `json:"idxlo,"`
	Vol   uint64 `json:"vol,"`
}

// GetOSState returns the platform specific StateOS
func GetOSState(info os.FileInfo) StateOS {

	// os.SameFile must be called to populate the id fields. Otherwise in case for example
	// os.Stat(file) is used to get the fileInfo, the ids are empty.
	// https://github.com/elastic/beats/filebeat/pull/53
	os.SameFile(info, info)

	// Gathering fileStat (which is fileInfo) through reflection as otherwise not accessible
	// See https://github.com/golang/go/blob/90c668d1afcb9a17ab9810bce9578eebade4db56/src/os/stat_windows.go#L33
	fileStat := reflect.ValueOf(info).Elem()

	// Get the three fields required to uniquely identify file und windows
	// More details can be found here: https://msdn.microsoft.com/en-us/library/aa363788(v=vs.85).aspx
	// Uint should already return uint64, but making sure this is the case
	// The required fiels can be found here: https://github.com/golang/go/blob/master/src/os/types_windows.go#L78
	fileState := StateOS{
		IdxHi: uint64(fileStat.FieldByName("idxhi").Uint()),
		IdxLo: uint64(fileStat.FieldByName("idxlo").Uint()),
		Vol:   uint64(fileStat.FieldByName("vol").Uint()),
	}

	return fileState
}

// IsSame file checks if the files are identical
func (fs StateOS) IsSame(state StateOS) bool {
	return fs.IdxHi == state.IdxHi && fs.IdxLo == state.IdxLo && fs.Vol == state.Vol
}

// SafeFileRotate safely rotates an existing file under path and replaces it with the tempfile
func SafeFileRotate(path, tempfile string) error {
	old := path + ".old"
	var e error

	// In Windows, one cannot rename a file if the destination already exists, at least
	// not with using the os.Rename function that Golang offers.
	// This tries to move the existing file into an old file first and only do the
	// move after that.
	if e = os.Remove(old); e != nil {
		logp.Debug("filecompare", "delete old: %v", e)
		// ignore error in case old doesn't exit yet
	}
	if e = os.Rename(path, old); e != nil {
		logp.Debug("filecompare", "rotate to old: %v", e)
		// ignore error in case path doesn't exist
	}

	if e = os.Rename(tempfile, path); e != nil {
		logp.Err("rotate: %v", e)
		return e
	}
	return nil
}

// ReadOpen opens a file for reading only
// As Windows blocks deleting a file when its open, some special params are passed here.
func ReadOpen(path string) (*os.File, error) {

	// Set all write flags
	// This indirectly calls syscall_windows::Open method https://github.com/golang/go/blob/7ebcf5eac7047b1eef2443eda1786672b5c70f51/src/syscall/syscall_windows.go#L251
	// As FILE_SHARE_DELETE cannot be passed to Open, os.CreateFile must be implemented directly

	// This is mostly the code from syscall_windows::Open. Only difference is passing the Delete flag
	// TODO: Open pull request to Golang so also Delete flag can be set
	if len(path) == 0 {
		return nil, fmt.Errorf("File '%s' not found. Error: %v", path, syscall.ERROR_FILE_NOT_FOUND)
	}

	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("Error converting to UTF16: %v", err)
	}

	var access uint32
	access = syscall.GENERIC_READ

	sharemode := uint32(syscall.FILE_SHARE_READ | syscall.FILE_SHARE_WRITE | syscall.FILE_SHARE_DELETE)

	var sa *syscall.SecurityAttributes

	var createmode uint32

	createmode = syscall.OPEN_EXISTING

	handle, err := syscall.CreateFile(pathp, access, sharemode, sa, createmode, syscall.FILE_ATTRIBUTE_NORMAL, 0)

	if err != nil {
		return nil, fmt.Errorf("Error creating file '%s': %v", pathp, err)
	}

	return os.NewFile(uintptr(handle), path), nil
}
