// +build windows

package file

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/input/file"
)

func Stat(path string) (*Metadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to stat path")
	}

	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil, errors.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	state := file.GetOSState(info)

	fileInfo := &Metadata{
		Inode: uint64(state.IdxHi<<32 + state.IdxLo),
		Mode:  info.Mode(),
		Size:  info.Size(),
		ATime: time.Unix(0, attrs.LastAccessTime.Nanoseconds()).UTC(),
		MTime: time.Unix(0, attrs.LastWriteTime.Nanoseconds()).UTC(),
		CTime: time.Unix(0, attrs.CreationTime.Nanoseconds()).UTC(),
	}

	switch {
	case info.IsDir():
		fileInfo.Type = "dir"
	case info.Mode().IsRegular():
		fileInfo.Type = "file"
	case info.Mode()&os.ModeSymlink > 0:
		fileInfo.Type = "symlink"
	}

	// fileOwner only works on files or symlinks to file because os.Open only
	// works on files. To open a dir we need to use CreateFile with the
	// FILE_FLAG_BACKUP_SEMANTICS flag.
	if !info.IsDir() {
		fileInfo.SID, fileInfo.Owner, err = fileOwner(path)
	}
	return fileInfo, err
}

func fileOwner(path string) (sid, owner string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to open file to get owner")
	}
	defer f.Close()

	var securityID *syscall.SID
	var securityDescriptor *SecurityDescriptor

	if err := GetSecurityInfo(syscall.Handle(f.Fd()), FileObject,
		OwnerSecurityInformation, &securityID, nil, nil, nil, &securityDescriptor); err != nil {
		return "", "", errors.Wrapf(err, "failed on GetSecurityInfo for %v", path)
	}
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(securityDescriptor)))

	// Covert SID to a string and lookup the username.
	var errs multierror.Errors
	sid, err = securityID.String()
	if err != nil {
		errs = append(errs, err)
	}

	account, domain, _, err := securityID.LookupAccount("")
	if err != nil {
		errs = append(errs, err)
	} else {
		owner = fmt.Sprintf(`%s\%s`, domain, account)
	}

	return sid, owner, errs.Err()
}
