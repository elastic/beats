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
	"crypto/sha256"
	"encoding/hex"
	"strings"

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
	OpNotChanged
)

var operationNames = map[Operation]string{
	OpDone:       "done",
	OpCreate:     "create",
	OpWrite:      "write",
	OpDelete:     "delete",
	OpRename:     "rename",
	OpTruncate:   "truncate",
	OpArchived:   "archive",
	OpNotChanged: "not changed",
}

// Operation describes what happened to a file.
type Operation uint8

func (o Operation) String() string {
	name, ok := operationNames[o]
	if !ok {
		return ""
	}
	return name
}

// FingerprintID is the file-identity material derived from the fingerprint
// region bytes[offset:offset+length]. It is self-describing: a single value
// carries everything the watcher, the prospector and the registry key need,
// so no caller has to branch on a separate "is this growing?" flag.
type FingerprintID struct {
	// Raw is the hex-encoded fingerprint region read so far
	// (bytes[offset:offset+min(size,length)]). It is the matching material: a
	// growing file extends Raw, so a previous (shorter) Raw is a prefix of the
	// current one. The scanner sets Raw only in growing mode; static-mode and
	// retained-but-completed descriptors leave it empty (matching falls back
	// to the SHA-256 identity). Empty when no fingerprint was computed.
	Raw string
	// Complete is true once Raw covers the full configured length, i.e. the
	// file reached offset+length and Sum holds the final SHA-256.
	Complete bool
	// Sum is hex(sha256(bytes[offset:offset+length])), set only when Complete.
	// It is the durable registry identity — byte-identical to the value the
	// static (non-growing) fingerprint produces.
	Sum string
}

// Key returns the registry/identity key for this fingerprint, or "" when no
// fingerprint is available (the caller then falls back to the OS file state).
//
// A completed fingerprint keys on its SHA-256, identical to the static
// fingerprint identity so existing state is reused with no duplication. A
// still-growing fingerprint keys on a bounded hash of Raw: the raw header can
// be up to 2*length characters and the registry key is rewritten on every
// cursor checkpoint, so hashing keeps the memlog WAL small. The raw value is
// persisted in the entry instead, so prefix matching survives restarts.
func (f FingerprintID) Key() string {
	switch {
	case f.Complete:
		return f.Sum
	case f.Raw != "":
		sum := sha256.Sum256([]byte(f.Raw))
		return hex.EncodeToString(sum[:])
	default:
		return ""
	}
}

// Continues reports whether current represents the same file as f observed
// later with at least as much content: f's raw fingerprint material is a
// prefix of current's. This single relation drives growing-mode rename and
// threshold-crossing detection — current.Raw still carries the full header on
// the scan a file crosses the threshold, so the same prefix test bridges the
// transition to the SHA-256 identity. Returns false when f has no raw material
// (a completed entry whose Raw was dropped, or a static-mode entry).
func (f FingerprintID) Continues(current FingerprintID) bool {
	return f.Raw != "" && strings.HasPrefix(current.Raw, f.Raw)
}

// FileDescriptor represents full information about a file.
type FileDescriptor struct {
	// Filename is an original filename this descriptor was created from.
	// In case it was a symlink, this will be the filename of the symlink unlike
	// the filename from the `Info`.
	Filename string
	// Info is the result of file stat
	Info file.ExtendedFileInfo
	// Fingerprint is the file-identity material for the "fingerprint" identity.
	// It is the zero value when fingerprinting is disabled or produced nothing.
	Fingerprint FingerprintID
	// GZIP indicates if the file is compressed with GZIP.
	GZIP bool

	// bytesIngested is the number of bytes already ingested by the harvester
	// for this file
	bytesIngested int64
}

// SetBytesIngested allows for setting a size that is different than the one in Info
func (fd *FileDescriptor) SetBytesIngested(s int64) {
	fd.bytesIngested = s
}

// SizeOrBytesIngested returns the bytes ingested for the file or its size.
// If [SetBytesIngested] has been called with a value other
// than zero, the bytes ingested is returned, otherwise Info.Size() is returned.
func (fd FileDescriptor) SizeOrBytesIngested() int64 {
	if fd.bytesIngested != 0 {
		return fd.bytesIngested
	}

	return fd.Info.Size()
}

// FileID returns a unique in-memory identifier used by the scanner and watcher
// to recognise the same file across scans. If a fingerprint is computed it is
// used as the ID, otherwise a combination of the device ID and inode.
//
// Unlike Key (the persistent registry key), this identifier is never stored, so
// it does not need to be bounded: a still-growing file is identified by its raw
// fingerprint hex directly, avoiding a per-scan hash on the watcher hot path. A
// completed file uses its SHA-256, so the identity changes exactly once when the
// file crosses the threshold — SameFile bridges that transition via Continues.
func (fd FileDescriptor) FileID() string {
	switch {
	case fd.Fingerprint.Complete:
		return fd.Fingerprint.Sum
	case fd.Fingerprint.Raw != "":
		return fd.Fingerprint.Raw
	default:
		return fd.Info.GetOSState().Identifier()
	}
}

// SameFile returns true if descriptors point to the same file.
//
// Two matching paths are tried, in order:
//
//  1. Exact FileID match — the common case for files whose identity has not
//     changed between scans (and the only path used by the static fingerprint
//     and OS-state identities).
//  2. Growing-phase prefix match — the previous raw fingerprint material is a
//     prefix of the current one. This covers both below-threshold growth and
//     the one-time crossing to the SHA-256 identity (see FingerprintID.Continues).
func SameFile(prev, current *FileDescriptor) bool {
	if prev.FileID() == current.FileID() {
		return true
	}
	return prev.Fingerprint.Continues(current.Fingerprint)
}

// FSEvent returns information about file system changes.
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
	// SrcID is the identifier used to identify the harvester and the
	// entry in the registry
	SrcID string
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
	// NotifyChan returns the channel used to listen for
	// harvester closing notifications
	NotifyChan() chan HarvesterStatus
}
