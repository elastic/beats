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

package ecs

import (
	"time"
)

// A file is defined as a set of information that has been created on, or has
// existed on a filesystem.
// File objects can be associated with host events, network events, and/or file
// events (e.g., those produced by File Integrity Monitoring [FIM] products or
// services). File fields provide details about the affected file associated
// with the event or metric.
type File struct {
	// Name of the file including the extension, without the directory.
	Name string `ecs:"name"`

	// Array of file attributes.
	// Attributes names will vary by platform. Here's a non-exhaustive list of
	// values that are expected in this field: archive, compressed, directory,
	// encrypted, execute, hidden, read, readonly, system, write.
	Attributes []string `ecs:"attributes"`

	// Directory where the file is located. It should include the drive letter,
	// when appropriate.
	Directory string `ecs:"directory"`

	// Drive letter where the file is located. This field is only relevant on
	// Windows.
	// The value should be uppercase, and not include the colon.
	DriveLetter string `ecs:"drive_letter"`

	// Full path to the file, including the file name. It should include the
	// drive letter, when appropriate.
	Path string `ecs:"path"`

	// Target path for symlinks.
	TargetPath string `ecs:"target_path"`

	// File extension, excluding the leading dot.
	// Note that when the file name has multiple extensions (example.tar.gz),
	// only the last one should be captured ("gz", not "tar.gz").
	Extension string `ecs:"extension"`

	// File type (file, dir, or symlink).
	Type string `ecs:"type"`

	// Device that is the source of the file.
	Device string `ecs:"device"`

	// Inode representing the file in the filesystem.
	Inode string `ecs:"inode"`

	// The user ID (UID) or security identifier (SID) of the file owner.
	UID string `ecs:"uid"`

	// File owner's username.
	Owner string `ecs:"owner"`

	// Primary group ID (GID) of the file.
	Gid string `ecs:"gid"`

	// Primary group name of the file.
	Group string `ecs:"group"`

	// Mode of the file in octal representation.
	Mode string `ecs:"mode"`

	// File size in bytes.
	// Only relevant when `file.type` is "file".
	Size int64 `ecs:"size"`

	// Last time the file content was modified.
	Mtime time.Time `ecs:"mtime"`

	// Last time the file attributes or metadata changed.
	// Note that changes to the file content will update `mtime`. This implies
	// `ctime` will be adjusted at the same time, since `mtime` is an attribute
	// of the file.
	Ctime time.Time `ecs:"ctime"`

	// File creation time.
	// Note that not all filesystems store the creation time.
	Created time.Time `ecs:"created"`

	// Last time the file was accessed.
	// Note that not all filesystems keep track of access time.
	Accessed time.Time `ecs:"accessed"`

	// MIME type should identify the format of the file or stream of bytes
	// using
	// https://www.iana.org/assignments/media-types/media-types.xhtml[IANA
	// official types], where possible. When more than one type is applicable,
	// the most specific type should be used.
	MimeType string `ecs:"mime_type"`

	// A fork is additional data associated with a filesystem object.
	// On Linux, a resource fork is used to store additional data with a
	// filesystem object. A file always has at least one fork for the data
	// portion, and additional forks may exist.
	// On NTFS, this is analogous to an Alternate Data Stream (ADS), and the
	// default data stream for a file is just called $DATA. Zone.Identifier is
	// commonly used by Windows to track contents downloaded from the Internet.
	// An ADS is typically of the form:
	// `C:\path\to\filename.extension:some_fork_name`, and `some_fork_name` is
	// the value that should populate `fork_name`. `filename.extension` should
	// populate `file.name`, and `extension` should populate `file.extension`.
	// The full path, `file.path`, will include the fork name.
	ForkName string `ecs:"fork_name"`
}
