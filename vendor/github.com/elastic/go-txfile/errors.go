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

package txfile

import (
	"fmt"

	"github.com/elastic/go-txfile/internal/strbld"
	"github.com/elastic/go-txfile/internal/vfs"
	"github.com/elastic/go-txfile/txerr"
)

// reason is used as package internal error type. It's used to guarantee all
// package level errors generated or returned by txfile are compatible to txerr.Error.
type reason interface {
	txerr.Error
}

// Error is the actual error type returned by all functions/methods within the
// txfile package.
// The Error is compatible to error and txerr.Error, but adds a few additional
// meta-data for applications to report and handle errors.
// Each single field in Error is optional. Fields can be accessed by methods only.
// As fields can being optional and Error being used to wrap other errors
// as well, txerr should be for inspecting errors.
type Error struct {
	op    string
	kind  error
	cause error

	ctx errorCtx
	msg string
}

// errorCtx stores additional metadata associated with an error and it's root cause.
// When adding an error cause, the context is merged, such that no two context
// variables with same contents will be reported twice.
type errorCtx struct {
	// database filename. Empty string if error is not related to a file
	file string

	// exact file offset an error was detected at
	offset int64
	isOff  bool // set if offset is valid

	// active transaction ID
	txid uint
	isTx bool // set if txid is valid

	// page number an error was detected for
	page   PageID
	isPage bool // set if the page id is valid
}

var _ reason = &Error{}

// Error formats the error message. The cause will not be included in the error
// string. Use fmt with %+v to create a formatted multiline error.
func (e *Error) Error() string { return txerr.Report(e, false) }

// Format adds support for fmt.Formatter to Error.
// The format patterns %v and %s print the top-level error message only
// (similar to `(*Error).Error()`). The format pattern "q" is similar to "%s",
// but adds double quotes before and after the message.
// Use %+v to create a multiline string containing the full trace of errors.
func (e *Error) Format(s fmt.State, c rune) { txerr.Format(e, s, c) }

// Op returns the operation the error occured at. Returns "" if the error value
// is used to wrap another error. Better use `txerr.GetOp(err)` to query an error value for
// the causing operation.
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

// ErrKind defines txfile error kinds(codes). ErrKind is compatible to error, so it can be used with `txerr.Is()`.
type ErrKind int

// internal txfile error kinds

//go:generate stringer -type=ErrKind -linecomment=true

const (
	NoError            ErrKind = iota // no error
	InternalError                     // internal error
	FileCreationFailed                // can not create file
	InitFailed                        // failed to initialize from file
	InvalidConfig                     // configuration error
	InvalidFileSize                   // invalid file size
	InvalidMetaPage                   // meta page invalid
	InvalidOp                         // invalid operation
	InvalidPageID                     // page id out of bounds
	InvalidParam                      // invalid parameter
	OutOfMemory                       // out of memory
	TxCommitFail                      // transaction failed during commit
	TxRollbackFail                    // transaction failed during rollback
	TxFailed                          // transaction failed
	TxFinished                        // finished transaction
	TxReadOnly                        // readonly transaction
	endOfErrKind                      // unknown error kind
)

// re-export file system error kinds (from internal/vfs)

const (
	PermissionError       = vfs.ErrPermission
	FileExists            = vfs.ErrExist
	FileDoesNotExist      = vfs.ErrNotExist
	FileClosed            = vfs.ErrClosed
	NoDiskSpace           = vfs.ErrNoSpace
	FDLimit               = vfs.ErrFDLimit
	CantResolvePath       = vfs.ErrResolvePath
	IOError               = vfs.ErrIO
	OSOtherError          = vfs.ErrOSOther
	OperationNotSupported = vfs.ErrNotSupported
	LockFailed            = vfs.ErrLockFailed
)

// Error returns a user readable error message.
func (k ErrKind) Error() string {
	if k > endOfErrKind {
		k = endOfErrKind
	}
	return k.String()
}

func (e *Error) of(kind ErrKind) *Error { e.kind = kind; return e }

func (e *Error) report(m string) *Error                     { e.msg = m; return e }
func (e *Error) reportf(m string, vs ...interface{}) *Error { return e.report(fmt.Sprintf(m, vs...)) }

// causedBy adds a cause to e and returns the modified e itself.
// The error contexts are merged (duplicates are removed from the cause), if
// the cause is `*Error`.
func (e *Error) causedBy(cause error) *Error {
	e.cause = cause
	other, ok := cause.(*Error)
	if !ok {
		return e
	}

	errCtx := &e.ctx
	causeCtx := &other.ctx
	if errCtx.file == causeCtx.file {
		causeCtx.file = ""
	}
	if errCtx.isTx && causeCtx.isTx && errCtx.txid == causeCtx.txid {
		causeCtx.isTx = false // delete common tx id from cause context
	}
	if errCtx.isPage && causeCtx.isPage && errCtx.page == causeCtx.page {
		causeCtx.isPage = false // delete common page id from cause context
	}
	if errCtx.isOff && causeCtx.isOff && errCtx.offset == causeCtx.offset {
		causeCtx.isOff = false // delete common page id from cause context
	}

	return e
}

func (ctx *errorCtx) String() string {
	buf := &strbld.Builder{}
	if ctx.file != "" {
		buf.Fmt("file='%s'", ctx.file)
	}
	if ctx.isTx {
		buf.Pad(" ")
		buf.Fmt("tx=%v", ctx.txid)
	}
	if ctx.isPage {
		buf.Pad(" ")
		buf.Fmt("page=%v", ctx.page)
	}
	if ctx.isOff {
		buf.Pad(" ")
		buf.Fmt("offset=%v", ctx.offset)
	}
	return buf.String()
}

func (ctx *errorCtx) SetPage(id PageID) {
	ctx.isPage, ctx.page = true, id
}

func (ctx *errorCtx) SetOffset(off int64) {
	ctx.isOff, ctx.offset = true, off
}

func errOp(op string) *Error {
	return &Error{op: op}
}

func errOf(kind ErrKind) *Error {
	return &Error{kind: kind}
}

func wrapErr(err error) *Error {
	return &Error{cause: err}
}

func raiseInvalidParam(msg string) reason {
	return &Error{kind: InvalidParam, msg: msg}
}

func raiseInvalidParamf(msg string, vs ...interface{}) reason {
	return raiseInvalidParam(fmt.Sprintf(msg, vs...))
}

func raiseOutOfBounds(id PageID) reason {
	return &Error{
		kind: InvalidPageID,
		ctx: errorCtx{
			isPage: true,
			page:   id,
		},
		msg: "out put bounds page id",
	}
}
