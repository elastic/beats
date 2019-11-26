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

package statestore

import (
	"errors"
	"fmt"
	"strings"
)

// error codes

var (
	// ErrClosed error code indicates that the operation can not
	// be executed, because the store has been closed already.
	ErrClosed = errors.New("store is closed")
)

// Error provides a common error type used by the statestore package.  It
// reports the failed operation, custom message and the root cause if
// available.
type Error struct {
	op      string
	code    error
	message string
	cause   error
}

// Op returns the name of the operation that failed.
func (e *Error) Op() string {
	return e.op
}

// Code returns a sentinal error that can be used for checking the error type.
func (e *Error) Code() error {
	return e.code
}

// Unwrap returns the cause if available.
func (e *Error) Unwrap() error {
	return e.cause
}

// Error builds the complete error string.
func (e *Error) Error() string {
	var buf strings.Builder

	pad := func() {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
	}

	padOpt := func(err error) {
		if err != nil {
			pad()
			fmt.Fprintf(&buf, "%+v", err)
		}
	}

	if e.op != "" {
		buf.WriteString(e.op)
	}
	padOpt(e.code)
	if e.message != "" {
		pad()
		buf.WriteString(e.message)
	}
	padOpt(e.cause)
	return buf.String()
}

func raiseClosed(op string) *Error {
	return &Error{
		op:   op,
		code: ErrClosed,
	}
}
