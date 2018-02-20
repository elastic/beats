// +build windows

package file_integrity

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common/file"
)

// NewMetadata returns a new Metadata object. If an error is returned it is
// still possible for a non-nil Metadata object to be returned (possibly with
// less data populated).
func NewMetadata(path string, info os.FileInfo) (*Metadata, error) {
	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil, errors.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	state := file.GetOSState(info)

	fileInfo := &Metadata{
		Inode: uint64(state.IdxHi<<32 + state.IdxLo),
		Mode:  info.Mode(),
		Size:  uint64(info.Size()),
		MTime: time.Unix(0, attrs.LastWriteTime.Nanoseconds()).UTC(),
		CTime: time.Unix(0, attrs.CreationTime.Nanoseconds()).UTC(),
	}

	switch {
	case info.Mode().IsRegular():
		fileInfo.Type = FileType
	case info.IsDir():
		fileInfo.Type = DirType
	case info.Mode()&os.ModeSymlink > 0:
		fileInfo.Type = SymlinkType
	}

	// fileOwner only works on files or symlinks to file because os.Open only
	// works on files. To open a dir we need to use CreateFile with the
	// FILE_FLAG_BACKUP_SEMANTICS flag.
	var err error
	if !info.IsDir() {
		fileInfo.SID, fileInfo.Owner, err = fileOwner(path)
	}
	fileInfo.Origin, err = GetFileOrigin(path)
	return fileInfo, err
}

// fileOwner returns the SID and name (domain\user) of the file's owner.
func fileOwner(path string) (sid, owner string, err error) {
	f, err := file.ReadOpen(path)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to open file to get owner")
	}
	defer f.Close()

	var securityID *syscall.SID
	var securityDescriptor *SecurityDescriptor

	if err = GetSecurityInfo(syscall.Handle(f.Fd()), FileObject,
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
