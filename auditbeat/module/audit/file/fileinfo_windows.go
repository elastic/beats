// +build windows

package file

import (
	"os"
	"reflect"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

func addFileAttributes(e *Event, maxFileSize int64) error {
	info, err := os.Lstat(e.Path)
	if err != nil {
		return errors.Wrap(err, "failed to stat file")
	}

	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return errors.Errorf("unexpected fileinfo sys type %T", info.Sys())
	}

	fileStat := reflect.ValueOf(info).Elem()
	idxhi := uint32(fileStat.FieldByName("idxhi").Uint())
	idxlo := uint32(fileStat.FieldByName("idxlo").Uint())

	e.Inode = uint64(idxhi<<32 + idxlo)
	e.Size = info.Size()
	// TODO: Add owner.
	e.ATime = time.Unix(0, attrs.LastAccessTime.Nanoseconds())
	e.MTime = time.Unix(0, attrs.LastWriteTime.Nanoseconds())
	e.CTime = time.Unix(0, attrs.CreationTime.Nanoseconds())

	if e.Size <= maxFileSize {
		md5sum, sha1sum, sha256sum, err := hashFile(e.Path)
		if err == nil {
			e.MD5 = md5sum
			e.SHA1 = sha1sum
			e.SHA256 = sha256sum
			e.Hashed = true
		}
	}

	return nil
}
