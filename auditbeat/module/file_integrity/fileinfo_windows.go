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

package file_integrity

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"golang.org/x/sys/windows"
)

// NewMetadata returns a new Metadata object. If an error is returned it is
// still possible for a non-nil Metadata object to be returned (possibly with
// less data populated).
func NewMetadata(path string, info os.FileInfo) (*Metadata, error) {
	attrs, ok := info.Sys().(*syscall.Win32FileAttributeData)
	if !ok {
		return nil, fmt.Errorf("unexpected fileinfo sys type %T for %v", info.Sys(), path)
	}

	var errs []error

	state := file.GetOSState(info)

	created := time.Unix(0, attrs.CreationTime.Nanoseconds()).UTC()
	accessed := time.Unix(0, attrs.LastAccessTime.Nanoseconds()).UTC()
	fileInfo := &Metadata{
		Attributes: attributesToStrings(attrs.FileAttributes),
		Inode:      state.IdxHi<<32 + state.IdxLo,
		Mode:       info.Mode(),
		Size:       uint64(info.Size()),
		MTime:      info.ModTime().UTC(),
		CTime:      created,
		Created:    &created,
		Accessed:   &accessed,
	}

	switch {
	case info.Mode().IsRegular():
		fileInfo.Type = FileType
	case info.IsDir():
		fileInfo.Type = DirType
	case info.Mode()&os.ModeSymlink > 0:
		fileInfo.Type = SymlinkType
	}

	var err error
	if fileInfo.SID, fileInfo.Owner, fileInfo.Group, err = getObjectSecurityInfo(path, info.IsDir()); err != nil {
		errs = append(errs, fmt.Errorf("getObjectSecurityInfo failed: %w", err))
	}

	if fileInfo.Origin, err = GetFileOrigin(path); err != nil {
		errs = append(errs, fmt.Errorf("GetFileOrigin failed: %w", err))
	}
	return fileInfo, errors.Join(errs...)
}

func getObjectSecurityInfo(path string, isDir bool) (ownerSID, ownerName, groupName string, err error) {
	var handle windows.Handle
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to convert path to UTF16: %w", err)
	}

	// Try multiple access levels, starting with least privilege
	accessLevels := []uint32{
		windows.READ_CONTROL,         // Minimum required for security info
		windows.GENERIC_READ,         // More permissive
		windows.FILE_READ_ATTRIBUTES, // Even more minimal
	}

	shareMode := uint32(windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE)
	creationDisposition := uint32(windows.OPEN_EXISTING)
	flags := uint32(windows.FILE_ATTRIBUTE_NORMAL)
	if isDir {
		flags |= windows.FILE_FLAG_BACKUP_SEMANTICS
	}

	// Try different access levels until one succeeds
	var lastErr error
	for _, access := range accessLevels {
		handle, lastErr = windows.CreateFile(pathPtr, access, shareMode, nil, creationDisposition, flags, 0)
		if lastErr == nil {
			break // Successfully opened with this access level
		}
	}

	if lastErr != nil {
		// If we can't open the file at all, try to get security info by path
		return getSecurityInfoByPath(path)
	}
	defer windows.CloseHandle(handle)

	// Try to get security information with graceful fallback
	return getSecurityInfoFromHandle(handle, path)
}

// getSecurityInfoByPath attempts to get security info without opening the file
func getSecurityInfoByPath(path string) (ownerSID, ownerName, groupName string, err error) {
	requestedInfo := windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION | windows.GROUP_SECURITY_INFORMATION)
	secInfo, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, requestedInfo)
	if err != nil {
		// Final fallback - try with just owner information
		requestedInfo = windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION)
		secInfo, err = windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, requestedInfo)
		if err != nil {
			return "", "", "", fmt.Errorf("GetNamedSecurityInfo failed for '%s': %w", path, err)
		}
	}

	return extractSecurityInfo(secInfo, path)
}

// getSecurityInfoFromHandle gets security info from an open file handle
func getSecurityInfoFromHandle(handle windows.Handle, path string) (ownerSID, ownerName, groupName string, err error) {
	requestedInfo := windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION | windows.GROUP_SECURITY_INFORMATION)
	secInfo, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, requestedInfo)
	if err != nil {
		// Fallback to just owner info if group access fails
		requestedInfo = windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION)
		secInfo, err = windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, requestedInfo)
		if err != nil {
			return "", "", "", fmt.Errorf("GetSecurityInfo failed for '%s': %w", path, err)
		}
	}

	return extractSecurityInfo(secInfo, path)
}

// extractSecurityInfo extracts owner and group information from security descriptor
func extractSecurityInfo(secInfo *windows.SECURITY_DESCRIPTOR, path string) (ownerSID, ownerName, groupName string, err error) {
	// Get owner information
	ownerSIDPtr, _, err := secInfo.Owner()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get owner for '%s': %w", path, err)
	}

	ownerSID = ownerSIDPtr.String()

	// Try to resolve owner name - don't fail if this doesn't work
	account, domain, use, err := ownerSIDPtr.LookupAccount("")
	if err == nil && account != "" {
		if domain != "" {
			ownerName = fmt.Sprintf(`%s\%s`, domain, account)
		} else {
			ownerName = account
		}

		// Check if the SID_NAME_USE value indicates a group type
		switch use {
		case windows.SidTypeGroup, windows.SidTypeWellKnownGroup, windows.SidTypeAlias:
			groupName = ownerName // If it's a group, use the same name
			return ownerSID, ownerName, groupName, nil
		}
	}

	// Try to get group information - this might fail on some file systems or with limited permissions
	groupSIDPtr, _, err := secInfo.Group()
	if err == nil {
		account, domain, _, err := groupSIDPtr.LookupAccount("")
		if err == nil && account != "" && account != "None" {
			if domain != "" {
				groupName = fmt.Sprintf(`%s\%s`, domain, account)
			} else {
				groupName = account
			}
		}
	}

	return ownerSID, ownerName, groupName, nil
}

// attributeMap maps the Windows file attribute constants to human-readable strings.
var attributeMap = map[uint32]string{
	windows.FILE_ATTRIBUTE_READONLY:      "read_only",
	windows.FILE_ATTRIBUTE_HIDDEN:        "hidden",
	windows.FILE_ATTRIBUTE_SYSTEM:        "system",
	windows.FILE_ATTRIBUTE_DIRECTORY:     "directory",
	windows.FILE_ATTRIBUTE_ARCHIVE:       "archive",
	windows.FILE_ATTRIBUTE_DEVICE:        "device",
	windows.FILE_ATTRIBUTE_NORMAL:        "normal",
	windows.FILE_ATTRIBUTE_TEMPORARY:     "temporary",
	windows.FILE_ATTRIBUTE_SPARSE_FILE:   "sparse_file",
	windows.FILE_ATTRIBUTE_REPARSE_POINT: "reparse_point",
	windows.FILE_ATTRIBUTE_COMPRESSED:    "compressed",
	windows.FILE_ATTRIBUTE_OFFLINE:       "offline",
	windows.FILE_ATTRIBUTE_ENCRYPTED:     "encrypted",
	windows.FILE_ATTRIBUTE_VIRTUAL:       "virtual",
}

func attributesToStrings(attributes uint32) []string {
	var result []string

	for flag, name := range attributeMap {
		// Check if the attribute bit is set in the input value.
		if attributes&flag != 0 {
			result = append(result, name)
		}
	}

	return result
}
