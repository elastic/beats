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

package pq

import (
	"fmt"

	"github.com/elastic/go-txfile"

	"github.com/elastic/go-txfile/internal/strbld"
	"github.com/elastic/go-txfile/txerr"
)

// ErrKind provides the pq related error kinds
type ErrKind int

// reason is used as package internal error type. It's used to guarantee all
// package level errors generated or returned by txfile are compatible to txerr.Error.
type reason interface {
	txerr.Error
}

// Error is the actual error type returned by all functions/methods within the
// pq package.  The Error is compatible to error and txerr.Error, but adds a
// few additional meta-data for applications to report and handle errors.
type Error struct {
	op    string
	kind  error
	cause error
	ctx   errorCtx
	msg   string
}

type errorCtx struct {
	// internal queue ID for correlating errors, changes between restarts.
	id queueID

	// page number an error was detected for
	page   txfile.PageID
	isPage bool // set if the page id is valid
}

//go:generate stringer -type=ErrKind -linecomment=true

const (
	NoError            ErrKind = iota // no error
	InitFailed                        // failed to initialize queue
	InvalidParam                      // invalid parameter
	InvalidPageSize                   // invalid page size
	InvalidConfig                     // invalid queue config
	QueueClosed                       // queue is already closed
	ReaderClosed                      // reader is already closed
	WriterClosed                      // writer is already closed
	NoQueueRoot                       // no queue root
	InvalidQueueRoot                  // queue root is invalid
	QueueVersion                      // unsupported queue version
	ACKEmptyQueue                     // invalid ack on empty queue
	ACKTooMany                        // too many events acked
	SeekFail                          // failed to seek to next page
	ReadFail                          // failed to read page
	InactiveTx                        // no active transaction
	UnexpectedActiveTx                // unexpected active transaction
)

// Error returns a user readable error message.
func (k ErrKind) Error() string {
	return k.String()
}

// Error returns the error message. The cause will not be included in the error
// string. Use fmt with %+v to create a formatted multiline error.
func (e *Error) Error() string { return txerr.Report(e, false) }

// Format adds support for fmt.Formatter to Error.
// The format patterns %v and %s print the top-level error message only
// (similar to `(*Error).Error()`). The format pattern "q" is similar to "%s",
// but adds double quotes before and after the message.
// Use %+v to create a multiline string containing the full trace of errors.
func (e *Error) Format(s fmt.State, c rune) { txerr.Format(e, s, c) }

// Op returns the operation the error occured at. Returns "" if the error value
// is used to wrap another error. Better use `txerr.GetOp(err)` to query an
// error value for the causing operation.
func (e *Error) Op() string { return e.op }

// Kind returns the error kind of the error. The kind should be used by
// applications to check if it is possible to recover from an error condition.
// Kind return nil if the error value does not define a kind. Better use
// `txerr.Is` or `txerr.GetKind` to query the error kind.
func (e *Error) Kind() error { return e.kind }

// Context returns a formatted string of the related meta-data as key/value
// pairs.
func (e *Error) Context() string { return e.ctx.String() }

// Message returns the user-focused error message.
func (e *Error) Message() string { return e.msg }

// Cause returns the causing error, if any.
func (e *Error) Cause() error { return e.cause }

// Errors is similar to `Cause()`, but returns a slice of errors. This way the
// error value can be consumed and formatted by zap (and propably other
// loggers).
func (e *Error) Errors() []error {
	if e.cause == nil {
		return nil
	}
	return []error{e.cause}
}

func (ctx *errorCtx) String() string {
	buf := &strbld.Builder{}
	if ctx.id != 0 {
		buf.Fmt("queueID=%v", ctx.id)
	}

	if ctx.isPage {
		buf.Pad(" ")
		buf.Fmt("page=%v", ctx.page)
	}
	return buf.String()
}

// IsQueueCorrupt checks if the error value indicates a corrupted queue, which
// can not be used anymore.
func IsQueueCorrupt(err error) bool {
	for _, kind := range []ErrKind{InvalidQueueRoot, SeekFail} {
		if txerr.Is(kind, err) {
			return true
		}
	}
	return false
}

func errOp(op string) *Error {
	return &Error{op: op}
}

func wrapErr(op string, cause error) *Error {
	return errOp(op).causedBy(cause)
}

func (e *Error) of(k ErrKind) *Error {
	e.kind = k
	return e
}

// causedBy adds a cause to e and returns the modified e itself.
// The error contexts are merged (duplicates are removed from the cause), if
// the cause is `*Error`.
func (e *Error) causedBy(cause error) *Error {
	e.cause = cause
	other, ok := cause.(*Error)
	if !ok {
		return e
	}

	// merge error and cause context such that the cause context only reports
	// fields that differ from the current context.

	errCtx := &e.ctx
	causeCtx := &other.ctx

	if errCtx.id == causeCtx.id {
		causeCtx.id = 0 // delete common queue id from cause context
	}
	if errCtx.isPage && causeCtx.isPage && errCtx.page == causeCtx.page {
		causeCtx.isPage = false // delete common page id from cause context
	}

	return e
}

func (e *Error) report(m string) *Error {
	e.msg = m
	return e
}
