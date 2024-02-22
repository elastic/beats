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

//go:build linux || freebsd || openbsd || netbsd || darwin

package file_integrity

import (
	"bytes"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"

	"github.com/joeshaw/multierror"
	"github.com/pkg/xattr"
)

// NewMetadata returns a new Metadata object. If an error is returned it is
// still possible for a non-nil Metadata object to be returned (possibly with
// less data populated).
func NewMetadata(path string, info os.FileInfo) (*Metadata, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, fmt.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
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

	fillExtendedAttributes(fileInfo, path)

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

func fillExtendedAttributes(md *Metadata, path string) {
	var selinux []byte
	getExtendedAttributes(path, map[string]*[]byte{
		"security.selinux":        &selinux,
		"system.posix_acl_access": &md.POSIXACLAccess,
	})
	// The selinux attr may be null terminated. It would be cheaper
	// to use strings.TrimRight, but absent documentation saying
	// that there is only ever a final null terminator, take the
	// guaranteed correct path of terminating at the first found
	// null byte.
	selinux, _, _ = bytes.Cut(selinux, []byte{0})
	md.SELinux = string(selinux)
}

func getExtendedAttributes(path string, dst map[string]*[]byte) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	for n, d := range dst {
		att, err := xattr.FGet(f, n)
		if err != nil {
			continue
		}
		*d = att
	}
}
