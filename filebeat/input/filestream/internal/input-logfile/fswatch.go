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
	"os"

	"github.com/elastic/go-concert/unison"
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

// FSEvent returns inforamation about file system changes.
type FSEvent struct {
	// NewPath is the new path of the file.
	NewPath string
	// OldPath is the previous path to the file, is it was
	// deleted or renamed.
	OldPath string
	// Op is the file system event: create, write, rename, remove
	Op Operation
	// Info describes the file in the event.
	Info os.FileInfo
}

// FSScanner retrieves a list of files from the file system.
type FSScanner interface {
	// GetFiles returns the list of monitored files.
	// The keys of the map are the paths to the files and
	// the values are the FileInfos describing the file.
	GetFiles() map[string]os.FileInfo
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
