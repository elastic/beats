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

package input_logfile

import (
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/libbeat/common/file"
)

const (
	OpDone Operation = iota
	OpCreate
	OpWrite
	OpDelete
	OpRename
	OpTruncate
	OpArchived
)

var operationNames = map[Operation]string{
	OpDone:     "done",
	OpCreate:   "create",
	OpWrite:    "write",
	OpDelete:   "delete",
	OpRename:   "rename",
	OpTruncate: "truncate",
	OpArchived: "archive",
}

// Operation describes what happened to a file.
type Operation uint8

func (o *Operation) String() string {
	name, ok := operationNames[*o]
	if !ok {
		return ""
	}
	return name
}

// FileDescriptor represents full information about a file.
type FileDescriptor struct {
	// Filename is an original filename this descriptor was created from.
	// In case it was a symlink, this will be the filename of the symlink unlike
	// the filename from the `Info`.
	Filename string
	// Info is the result of file stat
	Info file.ExtendedFileInfo
	// Fingerprint is a computed hash of the file header
	Fingerprint string
}

// FileID returns a unique file ID
// If fingerprint is computed it's used as the ID.
// Otherwise, a combination of the device ID and inode is used.
func (fd FileDescriptor) FileID() string {
	if fd.Fingerprint != "" {
		return fd.Fingerprint
	}
	return fd.Info.GetOSState().String()
}

// SameFile returns true if descriptors point to the same file.
func SameFile(a, b *FileDescriptor) bool {
	return a.FileID() == b.FileID()
}

// FSEvent returns inforamation about file system changes.
type FSEvent struct {
	// NewPath is the new path of the file.
	NewPath string
	// OldPath is the previous path to the file, is it was
	// deleted or renamed.
	OldPath string
	// Op is the file system event: create, write, rename, remove
	Op Operation
	// Descriptor describes the file in the event.
	Descriptor FileDescriptor
}

// FSScanner retrieves a list of files from the file system.
type FSScanner interface {
	// GetFiles returns the list of monitored files.
	// The keys of the map are the paths to the files and
	// the values are the file descriptors that contain all necessary information about the file.
	GetFiles() map[string]FileDescriptor
}

// FSWatcher returns file events of the monitored files.
type FSWatcher interface {
	FSScanner

	// Run is the event loop which watchers for changes
	// in the file system and returns events based on the data.
	Run(unison.Canceler)
	// Event returns the next event captured by FSWatcher.
	Event() FSEvent
}
