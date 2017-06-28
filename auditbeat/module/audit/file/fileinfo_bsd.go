// +build freebsd openbsd netbsd darwin

package file

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

func addFileAttributes(e *Event, maxFileSize int64) error {
	info, err := os.Lstat(e.Path)
	if err != nil {
		return errors.Wrap(err, "failed to stat file")
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.Errorf("unexpected fileinfo sys type %T", info.Sys())
	}

	e.Inode = stat.Ino
	e.UID = int32(stat.Uid)
	e.GID = int32(stat.Gid)
	e.Mode = fmt.Sprintf("%#o", uint32(info.Mode()))
	e.Size = info.Size()
	e.ATime = time.Unix(0, stat.Atimespec.Nano())
	e.MTime = time.Unix(0, stat.Mtimespec.Nano())
	e.CTime = time.Unix(0, stat.Ctimespec.Nano())

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
