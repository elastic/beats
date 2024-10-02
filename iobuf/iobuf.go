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

package iobuf

import (
	"bytes"
	"io"
)

// ReadAll reads all data from r and returns it as a byte slice.
// A successful call returns err == nil, not err == EOF. It does not
// treat an EOF as an error to be reported.
//
// This function is similar to io.ReadAll, but uses a bytes.Buffer to
// accumulate the data, which has a more efficient growing algorithm and
// uses io.WriterTo if r implements it.
func ReadAll(r io.Reader) ([]byte, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r)
	return buf.Bytes(), err
}
