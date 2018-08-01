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
	"github.com/elastic/beats/filebeat/reader"
)

// Reader sets an upper limited on line length. Lines longer
// then the max configured line length will be snapped short.
type LimitReader struct {
	reader   reader.Reader
	maxBytes int
}

// New creates a new reader limiting the line length.
func NewLimitReader(r reader.Reader, maxBytes int) *LimitReader {
	return &LimitReader{reader: r, maxBytes: maxBytes}
}

// Next returns the next line.
func (r *LimitReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()
	if len(message.Content) > r.maxBytes {
		message.Content = message.Content[:r.maxBytes]
	}
	return message, err
}
