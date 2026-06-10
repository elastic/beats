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

package reader

import (
	"io"
)

// Reader is the interface that wraps the basic Next method for
// getting a new message.
// Next returns the message being read or and error. EOF is returned
// if reader will not return any new message on subsequent calls.
type Reader interface {
	io.Closer
	Next() (Message, error)
}

// ContentRetainer is an optional capability for readers that keep a reference to
// a Message's Content beyond the Next() call that produced it — for example the
// multiline reader, which compares the previous line against the next. Readers
// that copy or fully transform Content within a single Next() do not implement
// it (or report that only an inner reader retains).
type ContentRetainer interface {
	// RetainsContent reports whether this reader, or any reader it wraps, holds
	// on to a Message's Content past the next Next() call.
	RetainsContent() bool
}

// RetainsContent reports whether r, or any reader it wraps, retains a Message's
// Content across Next() calls. Readers that don't implement ContentRetainer
// return false. It lets a caller decide whether the buffer backing Content can
// be reused across reads without corrupting an in-flight message.
func RetainsContent(r Reader) bool {
	c, ok := r.(ContentRetainer)
	return ok && c.RetainsContent()
}
