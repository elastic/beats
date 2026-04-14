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

package readfile

import (
	"fmt"
	"maps"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// FileMetaReader enriches every message with per-file metadata.
// OS metadata strings (device_id, inode, etc.) are cached after the
// first call to Next because they are constant for the file's lifetime.
type FileMetaReader struct {
	reader       reader.Reader
	path         string
	fi           file.ExtendedFileInfo
	includeOwner bool
	includeGroup bool
	fingerprint  string
	offset       int64
	cachedMeta   mapstr.M // lazily populated on first Next()
}

// New creates a new Encode reader from input reader by applying
// the given codec.
func NewFilemeta(r reader.Reader, path string, fi file.ExtendedFileInfo, includeOwner bool, includeGroup bool, fingerprint string, offset int64) reader.Reader {
	return &FileMetaReader{
		reader:       r,
		path:         path,
		fi:           fi,
		includeOwner: includeOwner,
		includeGroup: includeGroup,
		fingerprint:  fingerprint,
		offset:       offset,
	}
}

// Next reads the next line from it's initial io.Reader
// This converts a io.Reader to a reader.reader
func (r *FileMetaReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()

	// if the message is empty, there is no need to enrich it with file metadata
	if message.IsEmpty() {
		r.offset += int64(message.Bytes)
		return message, err
	}

	// On first call, compute and cache the per-file OS metadata (device_id,
	// inode, etc.) since they are constant for this file's lifetime.
	// Build into a local variable so a failed setFileSystemMetadata does not
	// leave r.cachedMeta in a partial state.
	if r.cachedMeta == nil {
		m := make(mapstr.M, 4)
		m["path"] = r.path
		if err := setFileSystemMetadata(r.fi, m, r.includeOwner, r.includeGroup); err != nil {
			return message, fmt.Errorf("failed to set file system metadata: %w", err)
		}
		if r.fingerprint != "" {
			m["fingerprint"] = r.fingerprint
		}
		r.cachedMeta = m
	}

	// Copy cached fields into a fresh map for this event.
	fileMap := make(mapstr.M, len(r.cachedMeta))
	maps.Copy(fileMap, r.cachedMeta)
	message.Fields["log"] = mapstr.M{
		"offset": r.offset,
		"file":   fileMap,
	}
	r.offset += int64(message.Bytes)

	return message, err
}

func (r *FileMetaReader) Close() error {
	return r.reader.Close()
}
