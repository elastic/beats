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

package file_integrity

import (
	"github.com/h2non/filetype"

	"github.com/menderesk/beats/v7/libbeat/common/file"
)

const (
	// Size for mime detection, office file
	// detection requires ~8kb to detect properly
	headerSize = 8192
)

// getMimeType does a best-effort to get the file type, if no
// filetype can be determined, it just returns an empty
// string
func getMimeType(path string) string {
	f, err := file.ReadOpen(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	head := make([]byte, headerSize)
	n, err := f.Read(head)
	if err != nil {
		return ""
	}

	kind, err := filetype.Match(head[:n])
	if err != nil {
		return ""
	}
	return kind.MIME.Value
}
