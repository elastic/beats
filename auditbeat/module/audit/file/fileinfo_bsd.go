// +build freebsd openbsd netbsd darwin

package file

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

func Stat(path string) (*Metadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to stat path")
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.Errorf("unexpected fileinfo sys type %T", info.Sys())
	}

	fileInfo := &Metadata{
		Inode: stat.Ino,
		UID:   stat.Uid,
		GID:   stat.Gid,
		Mode:  info.Mode(),
		Size:  info.Size(),
		ATime: time.Unix(0, stat.Atimespec.Nano()).UTC(),
		MTime: time.Unix(0, stat.Mtimespec.Nano()).UTC(),
		CTime: time.Unix(0, stat.Ctimespec.Nano()).UTC(),
	}

	switch {
	case info.IsDir():
		fileInfo.Type = "dir"
	case info.Mode().IsRegular():
		fileInfo.Type = "file"
	case info.Mode()&os.ModeSymlink > 0:
		fileInfo.Type = "symlink"
	}

	// Lookup UID and GID
	var errs multierror.Errors
	owner, err := user.LookupId(strconv.Itoa(int(fileInfo.UID)))
	if err != nil {
		errs = append(errs, err)
	} else {
		fileInfo.Owner = owner.Username
	}

	group, err := user.LookupGroupId(strconv.Itoa(int(fileInfo.GID)))
	if err != nil {
		errs = append(errs, err)
	} else {
		fileInfo.Group = group.Name
	}

	return fileInfo, errs.Err()
}
