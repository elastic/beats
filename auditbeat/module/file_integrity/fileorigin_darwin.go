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

// +build darwin

package file_integrity

/*
#include <stdlib.h>
#include <sys/xattr.h>
*/
import "C"

import (
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"howett.net/plist"
)

var (
	kMDItemWhereFroms = C.CString("com.apple.metadata:kMDItemWhereFroms")

	ignoredErrno = map[syscall.Errno]bool{
		syscall.ENOATTR: true, // Attribute not found
		syscall.ENOTSUP: true, // Extended Attributes not supported by the filesystem
		syscall.EISDIR:  true, // Not applicable to kMDItemWhereFroms
		syscall.ENOTDIR: true, // Not applicable to kMDItemWhereFroms
	}
)

// GetFileOrigin fetches the kMDItemWhereFroms metadata for the given path. This
// is special metadata in the filesystem that encodes information of an external
// origin of this file. It is always encoded as a list of strings, with
// different meanings depending on the origin:
//
// For files downloaded from a web browser, the first string is the URL for
// the source document. The second URL (optional), is the web address where the
// download link was followed:
// [ "https://cdn.kernel.org/pub/linux/kernel/v4.x/ChangeLog-4.13.16", "https://www.kernel.org/" ]
//
// For files or directories transferred via Airdrop, the origin is one string
// with the name of the computer that sent the file:
// [ "Adrian's MacBook Pro" ]
//
// For files attached to e-mails (using Mail app), three strings are
// returned: Sender address, subject and e-mail identifier:
// [ "Adrian Serrano \u003cadrian@elastic.co\u003e",
//   "Sagrada Familia tickets",
//   "message:%3CCAMZw10FD4fktC9qdJgLjwW=a8LM4gbJ44jFcaK8.BOWg1t4OwQ@elastic.co%3E"
// ],
//
// For all other files the result is an empty (nil) list.
func GetFileOrigin(path string) ([]string, error) {
	// Allocate a zero-terminated string representation of path. Must be freed
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	// Query length kMDItemWhereFroms extended-attribute
	attrSize, err := C.getxattr(cPath, kMDItemWhereFroms, nil, 0, 0, 0)
	if attrSize == -1 {
		return nil, errors.Wrap(filterErrno(err), "getxattr: query attribute length failed")
	}
	if attrSize == 0 {
		return nil, nil
	}

	// Read the kMDItemWhereFroms attribute
	data := make([]byte, attrSize)
	newSize, err := C.getxattr(cPath, kMDItemWhereFroms, unsafe.Pointer(&data[0]), C.size_t(attrSize), 0, 0)
	if newSize == -1 {
		return nil, errors.Wrap(filterErrno(err), "getxattr failed")
	}
	if newSize != attrSize {
		return nil, errors.New("getxattr: attribute changed while reading")
	}

	// Decode plist format. A list of strings is expected
	var urls []string
	if _, err = plist.Unmarshal(data, &urls); err != nil {
		return nil, errors.Wrap(err, "plist unmarshal failed")
	}

	// The returned list seems to be padded with empty strings when some of
	// the fields are missing (i.e. no context URL). Get rid of trailing empty
	// strings:
	n := len(urls)
	for n > 0 && len(urls[n-1]) == 0 {
		n--
	}
	return urls[:n], nil
}

func filterErrno(err error) error {
	if err == nil {
		return nil
	}
	if errno, ok := err.(syscall.Errno); ok {
		if _, ok = ignoredErrno[errno]; ok {
			return nil
		}
	}
	return err
}
