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

	"golang.org/x/sys/windows"

	"github.com/elastic/beats/v7/libbeat/common/file"
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
	var secInfo userInfo
	if secInfo, err = getObjectSecurityInfo(path, info.IsDir()); err == nil {
		fileInfo.SID = secInfo.sid
		fileInfo.Owner = secInfo.name
		fileInfo.Group = secInfo.groupName
		// For file.owner and file.group, use domain\name format since they don't have separate domain fields
		if secInfo.domain != "" {
			fileInfo.Owner = fmt.Sprintf("%s\\%s", secInfo.domain, secInfo.name)
		}
		if secInfo.groupDomain != "" {
			fileInfo.Group = fmt.Sprintf("%s\\%s", secInfo.groupDomain, secInfo.groupName)
		}
	} else {
		errs = append(errs, fmt.Errorf("getObjectSecurityInfo failed: %w", err))
	}

	if fileInfo.Origin, err = GetFileOrigin(path); err != nil {
		errs = append(errs, fmt.Errorf("GetFileOrigin failed: %w", err))
	}
	return fileInfo, errors.Join(errs...)
}

func getObjectSecurityInfo(path string, isDir bool) (info userInfo, err error) {
	var handle windows.Handle
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return userInfo{}, fmt.Errorf("failed to convert path to UTF16: %w", err)
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
func getSecurityInfoByPath(path string) (info userInfo, err error) {
	requestedInfo := windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION | windows.GROUP_SECURITY_INFORMATION)
	secInfo, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, requestedInfo)
	if err != nil {
		// Final fallback - try with just owner information
		requestedInfo = windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION)
		secInfo, err = windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, requestedInfo)
		if err != nil {
			return userInfo{}, fmt.Errorf("GetNamedSecurityInfo failed for '%s': %w", path, err)
		}
	}

	return extractSecurityInfo(secInfo, path)
}

// getSecurityInfoFromHandle gets security info from an open file handle
func getSecurityInfoFromHandle(handle windows.Handle, path string) (info userInfo, err error) {
	requestedInfo := windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION | windows.GROUP_SECURITY_INFORMATION)
	secInfo, err := windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, requestedInfo)
	if err != nil {
		// Fallback to just owner info if group access fails
		requestedInfo = windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION)
		secInfo, err = windows.GetSecurityInfo(handle, windows.SE_FILE_OBJECT, requestedInfo)
		if err != nil {
			return userInfo{}, fmt.Errorf("GetSecurityInfo failed for '%s': %w", path, err)
		}
	}

	return extractSecurityInfo(secInfo, path)
}

// extractSecurityInfo extracts owner and group information from security descriptor
func extractSecurityInfo(secInfo *windows.SECURITY_DESCRIPTOR, path string) (info userInfo, err error) {
	// Get owner information
	ownerSIDPtr, _, err := secInfo.Owner()
	if err != nil {
		return userInfo{}, fmt.Errorf("failed to get owner for '%s': %w", path, err)
	}

	info.sid = ownerSIDPtr.String()

	// Try to resolve owner name - don't fail if this doesn't work
	account, domain, use, err := ownerSIDPtr.LookupAccount("")
	if err == nil && account != "" {
		info.domain = domain
		info.name = account
		// Check if the SID_NAME_USE value indicates a group type
		switch use {
		case windows.SidTypeGroup, windows.SidTypeWellKnownGroup, windows.SidTypeAlias:
			// If it's a group, use the same info
			info.groupSID = info.sid
			info.groupDomain = info.domain
			info.groupName = info.name
			return info, nil
		}
	}

	// Try to get group information - this might fail on some file systems or with limited permissions
	groupSIDPtr, _, err := secInfo.Group()
	if err == nil {
		info.groupSID = groupSIDPtr.String()
		account, domain, _, err := groupSIDPtr.LookupAccount("")
		if err == nil && account != "" && account != "None" {
			info.groupDomain = domain
			info.groupName = account
		}
	}
	return info, nil
}

func attributesToStrings(attributes uint32) []string {
	var result []string
	if attributes&windows.FILE_ATTRIBUTE_READONLY != 0 {
		result = append(result, "read_only")
	}
	if attributes&windows.FILE_ATTRIBUTE_HIDDEN != 0 {
		result = append(result, "hidden")
	}
	if attributes&windows.FILE_ATTRIBUTE_SYSTEM != 0 {
		result = append(result, "system")
	}
	if attributes&windows.FILE_ATTRIBUTE_DIRECTORY != 0 {
		result = append(result, "directory")
	}
	if attributes&windows.FILE_ATTRIBUTE_ARCHIVE != 0 {
		result = append(result, "archive")
	}
	if attributes&windows.FILE_ATTRIBUTE_DEVICE != 0 {
		result = append(result, "device")
	}
	if attributes&windows.FILE_ATTRIBUTE_NORMAL != 0 {
		result = append(result, "normal")
	}
	if attributes&windows.FILE_ATTRIBUTE_TEMPORARY != 0 {
		result = append(result, "temporary")
	}
	if attributes&windows.FILE_ATTRIBUTE_SPARSE_FILE != 0 {
		result = append(result, "sparse_file")
	}
	if attributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		result = append(result, "reparse_point")
	}
	if attributes&windows.FILE_ATTRIBUTE_COMPRESSED != 0 {
		result = append(result, "compressed")
	}
	if attributes&windows.FILE_ATTRIBUTE_OFFLINE != 0 {
		result = append(result, "offline")
	}
	if attributes&windows.FILE_ATTRIBUTE_NOT_CONTENT_INDEXED != 0 {
		result = append(result, "not_content_indexed")
	}
	if attributes&windows.FILE_ATTRIBUTE_ENCRYPTED != 0 {
		result = append(result, "encrypted")
	}
	if attributes&windows.FILE_ATTRIBUTE_INTEGRITY_STREAM != 0 {
		result = append(result, "integrity_stream")
	}
	if attributes&windows.FILE_ATTRIBUTE_VIRTUAL != 0 {
		result = append(result, "virtual")
	}
	if attributes&windows.FILE_ATTRIBUTE_NO_SCRUB_DATA != 0 {
		result = append(result, "no_scrub_data")
	}
	if attributes&windows.FILE_ATTRIBUTE_RECALL_ON_OPEN != 0 {
		result = append(result, "recall_on_open")
	}
	if attributes&windows.FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS != 0 {
		result = append(result, "recall_on_data_access")
	}
	return result
}
