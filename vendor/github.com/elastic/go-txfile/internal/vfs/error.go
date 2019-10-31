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

package vfs

import (
	"github.com/elastic/go-txfile/internal/strbld"
)

// Error is the common error type used by vfs implementations.
type Error struct {
	op   string
	kind error
	path string
	err  error
}

// Kind encodes an error code for use by applications.
// Implementations of vfs must unify errors, using the Error type and the error
// codes defined here.
type Kind int

//go:generate stringer -type=Kind -linecomment=true
//go:generate beatsfmt -w kind_string.go
const (
	ErrOSOther      Kind = iota // unknown OS error
	ErrPermission               // permission denied
	ErrExist                    // file already exists
	ErrNotExist                 // file does not exist
	ErrClosed                   // file already closed
	ErrNoSpace                  // no space or quota exhausted
	ErrFDLimit                  // process file desciptor limit reached
	ErrResolvePath              // cannot resolve path
	ErrIO                       // read/write IO error
	ErrNotSupported             // operation not supported
	ErrLockFailed               // file lock failed
	ErrUnlockFailed             // file unlock failed

	endOfErrKind // unknown error kind
)

// Error returns the error codes descriptive text.
func (k Kind) Error() string {
	if k < endOfErrKind {
		return k.String()
	}
	return "unknown"
}

// Err creates a new Error. All fields are optional.
func Err(op string, kind Kind, path string, err error) *Error {
	return &Error{op: op, kind: kind, path: path, err: err}
}

// Op reports the failed operation.
func (e *Error) Op() string { return e.op }

// Kind returns the error code for use by the applications error handling code.
func (e *Error) Kind() error { return e.kind }

// Path returns the path of the file an operation failed for.
func (e *Error) Path() string { return e.path }

// Cause returns the causing error, is there is one. Returns nil if the error
// is the root cause of an error.
func (e *Error) Cause() error { return e.err }

// Errors returns the error cause as a list. The Errors method is avaialble for
// compatiblity with other error packages and libraries consuming errors (e.g. zap or multierr).
func (e *Error) Errors() []error {
	if e.err == nil {
		return nil
	}
	return []error{e.err}
}

// Error builds the error message of the underlying error.
func (e *Error) Error() string {
	buf := &strbld.Builder{}
	putStr(buf, e.op)
	putStr(buf, e.path)
	putErr(buf, e.kind)
	putErr(buf, e.err)

	if buf.Len() == 0 {
		return "no error"
	}
	return buf.String()
}

func putStr(b *strbld.Builder, s string) {
	if s != "" {
		b.Pad(": ")
		b.WriteString(s)
	}
}

func putErr(b *strbld.Builder, err error) {
	if err != nil {
		putStr(b, err.Error())
	}
}
