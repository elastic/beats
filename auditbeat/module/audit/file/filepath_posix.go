// +build linux freebsd openbsd netbsd darwin

package file

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

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
		return nil, errors.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	fileInfo := &Metadata{
		Inode: stat.Ino,
		UID:   stat.Uid,
		GID:   stat.Gid,
		Mode:  info.Mode(),
		Size:  info.Size(),
	}
	fileInfo.ATime, fileInfo.MTime, fileInfo.CTime = fileTimes(stat)

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
