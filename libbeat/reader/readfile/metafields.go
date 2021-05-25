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
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/reader"
)

// Reader produces lines by reading lines from an io.Reader
// through a decoder converting the reader it's encoding to utf-8.
type FileMetaReader struct {
	reader reader.Reader
	path   string
}

// New creates a new Encode reader from input reader by applying
// the given codec.
func NewFilemeta(r reader.Reader, path string) reader.Reader {
	return &FileMetaReader{r, path}
}

// Next reads the next line from it's initial io.Reader
// This converts a io.Reader to a reader.reader
func (r FileMetaReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()

	// if the message is empty, there is no need to enrich it with file metadata
	if message.IsEmpty() {
		return message, err
	}

	message.Fields.DeepUpdate(common.MapStr{
		"log": common.MapStr{
			"offset": message.Bytes,
			"path":   r.path,
		},
	})
	return message, err
}

func (r FileMetaReader) Close() error {
	return r.reader.Close()
}
