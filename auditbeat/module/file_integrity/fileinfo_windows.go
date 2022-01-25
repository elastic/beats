// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows
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

	"github.com/elastic/beats/v7/libbeat/common/file"
)

// NewMetadata returns a new Metadata object. If an error is returned it is
// still possible for a non-nil Metadata object to be returned (possibly with
// less data populated).
func NewMetadata(path string, info os.FileInfo) (*Metadata, error) {
	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil, errors.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	var errs multierror.Errors

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
		if fileInfo.SID, fileInfo.Owner, err = fileOwner(path); err != nil {
			errs = append(errs, errors.Wrap(err, "fileOwner failed"))
		}

	}
	if fileInfo.Origin, err = GetFileOrigin(path); err != nil {
		errs = append(errs, errors.Wrap(err, "GetFileOrigin failed"))
	}
	return fileInfo, errs.Err()
}

// fileOwner returns the SID and name (domain\user) of the file's owner.
func fileOwner(path string) (sid, owner string, err error) {
	var securityID *syscall.SID
	var securityDescriptor *SecurityDescriptor

	pathW, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return sid, owner, errors.Wrapf(err, "failed to convert path:'%s' to UTF16", path)
	}
	if err = GetNamedSecurityInfo(pathW, FileObject,
		OwnerSecurityInformation, &securityID, nil, nil, nil, &securityDescriptor); err != nil {
		return "", "", errors.Wrapf(err, "failed on GetSecurityInfo for %v", path)
	}
	defer syscall.LocalFree((syscall.Handle)(unsafe.Pointer(securityDescriptor)))

	// Convert SID to a string and lookup the username.
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
