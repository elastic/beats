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

package memlog

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// diskstore manages the on-disk state of the memlog store.
type diskstore struct {
	log *logp.Logger

	// store configuration
	checkpointPred CheckpointPredicate
	fileMode       os.FileMode
	bufferSize     int

	// on disk file tracking information
	home        string         // home path of the store
	logFilePath string         // current log file
	dataFiles   []dataFileInfo // set of data files found

	// txid is the sequential counter that tracks
	// all updates to the store. The txid is added to operation being logged
	// used as name for the data files.
	txid uint64

	// log file access. The log file is updated using an in memory write buffer.
	logFile *os.File
	logBuf  *bufio.Writer

	// internal state and metrics
	logFileSize      uint64
	logEntries       uint
	logInvalid       bool
	logNeedsTruncate bool
}

// dataFileInfo is used to track and sort on disk data files.
// We should have only one data file on disk, but in case delete operations
// have failed or not finished dataFileInfo is used to detect the ordering.
type dataFileInfo struct {
	path string
	txid uint64
}

// storeEntry is used to write entries to the checkpoint file only.
type storeEntry struct {
	Key    string        `struct:"_key"`
	Fields common.MapStr `struct:",inline"`
}

// storeMeta is read from the meta file.
type storeMeta struct {
	Version string `struct:"version"`
}

// logAction is prepended to each operation logged to the update file.
// It contains the update ID, a sequential counter to track correctness,
// and the action name.
type logAction struct {
	Op string `json:"op"`
	ID uint64 `json:"id"`
}

const (
	logFileName  = "log.json"
	metaFileName = "meta.json"

	storeVersion = "1"

	keyField = "_key"
)

// newDiskStore initializes the disk store stucture only. The store must have
// been opened already.  It tries to open the update log file for append
// operations. If opening the update log file fails, it is marked as
// 'corrupted', triggering a checkpoint operation on the first update to the store.
func newDiskStore(
	log *logp.Logger,
	home string,
	dataFiles []dataFileInfo,
	txid uint64,
	mode os.FileMode,
	entries uint,
	logInvalid bool,
	bufferSize uint,
	checkpointPred CheckpointPredicate,
) *diskstore {
	s := &diskstore{
		log:              log.With("path", home),
		home:             home,
		logFilePath:      filepath.Join(home, logFileName),
		dataFiles:        dataFiles,
		txid:             txid,
		fileMode:         mode,
		bufferSize:       int(bufferSize),
		logFile:          nil,
		logBuf:           nil,
		logEntries:       entries,
		logInvalid:       logInvalid,
		logNeedsTruncate: false, // only truncate on next checkpoint
		checkpointPred:   checkpointPred,
	}

	_ = s.tryOpenLog()
	return s
}

// tryOpenLog access the update log. The log file is truncated if a checkpoint operation has been
// executed last.
// The log file is marked as invalid if opening it failed. This will trigger a checkpoint operation
// and another call to tryOpenLog in the future.
func (s *diskstore) tryOpenLog() error {
	panic("TODO: implement me")
}

// mustCheckpoint returns true if the store is required to execute a checkpoint
// operation, either by predicate or by some internal state detecting a problem
// with the log file.
func (s *diskstore) mustCheckpoint() bool {
	return s.logInvalid || s.checkpointPred(s.logFileSize)
}

func (s *diskstore) Close() error {
	panic("TODO: implement me")
}

// log operation adds another entry to the update log file.
// The log file is marked as invalid if the write fails. This will trigger a
// checkpoint operation in the future.
func (s *diskstore) LogOperation(op op) error {
	panic("TODO: implement me")
}

// WriteCheckpoint serializes all state into a json file. The file contains an
// array with all states known to the memory storage.
// WriteCheckpoint first serializes all state to a temporary file, and finally
// moves the temporary data file into the correct location. No files
// are overwritten or replaced. Instead the change sequence number is used for
// the filename, and older data files will be deleted after success.
//
// The active marker file is overwritten after all updates did succeed. The
// marker file contains the filename of the current valid data-file.
// NOTE: due to limitation on some Operating system or file systems, the active
//       marker is not a symlink, but an actual file.
func (s *diskstore) WriteCheckpoint(state map[string]entry) error {
	panic("TODO: implement me")
}
