// +build linux freebsd openbsd netbsd darwin

package file_integrity

import (
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
)

// NewMetadata returns a new Metadata object. If an error is returned it is
// still possible for a non-nil Metadata object to be returned (possibly with
// less data populated).
func NewMetadata(path string, info os.FileInfo) (*Metadata, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, errors.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	fileInfo := &Metadata{
		Inode:  stat.Ino,
		UID:    stat.Uid,
		GID:    stat.Gid,
		Mode:   info.Mode().Perm(),
		Size:   uint64(info.Size()),
		SetUID: info.Mode()&os.ModeSetuid != 0,
		SetGID: info.Mode()&os.ModeSetgid != 0,
	}
	_, fileInfo.MTime, fileInfo.CTime = fileTimes(stat)

	switch {
	case info.Mode().IsRegular():
		fileInfo.Type = FileType
	case info.IsDir():
		fileInfo.Type = DirType
	case info.Mode()&os.ModeSymlink > 0:
		fileInfo.Type = SymlinkType
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
	if fileInfo.Origin, err = GetFileOrigin(path); err != nil {
		errs = append(errs, err)
	}
	return fileInfo, errs.Err()
}
