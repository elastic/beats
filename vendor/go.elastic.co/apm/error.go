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

package apm // import "go.elastic.co/apm"

import (
	"crypto/rand"
	"fmt"
	"net"
	"os"
	"reflect"
	"runtime"
	"syscall"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/internal/pkgerrorsutil"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/stacktrace"
)

const (
	// maxErrorGraphSize is the maximum number of errors
	// to report in an error tree. Once this number of
	// nodes is reached, we will stop recursing through
	// error causes.
	maxErrorTreeNodes = 50
)

// Recovered creates an Error with t.NewError(err), where
// err is either v (if v implements error), or otherwise
// fmt.Errorf("%v", v). The value v is expected to have
// come from a panic.
func (t *Tracer) Recovered(v interface{}) *Error {
	var e *Error
	switch v := v.(type) {
	case error:
		e = t.NewError(v)
	default:
		e = t.NewError(fmt.Errorf("%v", v))
	}
	return e
}

// NewError returns a new Error with details taken from err.
// NewError will panic if called with a nil error.
//
// The exception message will be set to err.Error().
// The exception module and type will be set to the package
// and type name of the cause of the error, respectively,
// where the cause has the same definition as given by
// github.com/pkg/errors.
//
// If err implements
//   type interface {
//       StackTrace() github.com/pkg/errors.StackTrace
//   }
// or
//   type interface {
//       StackTrace() []stacktrace.Frame
//   }
// then one of those will be used to set the error
// stacktrace. Otherwise, NewError will take a stacktrace.
//
// If err implements
//   type interface {Type() string}
// then that will be used to set the error type.
//
// If err implements
//   type interface {Code() string}
// or
//   type interface {Code() float64}
// then one of those will be used to set the error code.
func (t *Tracer) NewError(err error) *Error {
	if err == nil {
		panic("NewError must be called with a non-nil error")
	}
	e := t.newError()
	e.cause = err
	e.err = err.Error()
	if e.recording {
		rand.Read(e.ID[:]) // ignore error, can't do anything about it
		initException(&e.exception, err, e.stackTraceLimit)
		if len(e.exception.stacktrace) == 0 {
			e.SetStacktrace(2)
		}
	}
	return e
}

// NewErrorLog returns a new Error for the given ErrorLogRecord.
//
// The resulting Error's stacktrace will not be set. Call the
// SetStacktrace method to set it, if desired.
//
// If r.Message is empty, "[EMPTY]" will be used.
func (t *Tracer) NewErrorLog(r ErrorLogRecord) *Error {
	e := t.newError()
	e.cause = r.Error
	e.err = e.log.Message
	if e.recording {
		e.log = ErrorLogRecord{
			Message:       truncateString(r.Message),
			MessageFormat: truncateString(r.MessageFormat),
			Level:         truncateString(r.Level),
			LoggerName:    truncateString(r.LoggerName),
		}
		if e.log.Message == "" {
			e.log.Message = "[EMPTY]"
		}
		rand.Read(e.ID[:]) // ignore error, can't do anything about it
		if r.Error != nil {
			initException(&e.exception, r.Error, e.stackTraceLimit)
		}
	}
	return e
}

// newError returns a new Error associated with the Tracer.
func (t *Tracer) newError() *Error {
	e, _ := t.errorDataPool.Get().(*ErrorData)
	if e == nil {
		e = &ErrorData{
			tracer: t,
			Context: Context{
				captureBodyMask: CaptureBodyErrors,
			},
		}
	}

	instrumentationConfig := t.instrumentationConfig()
	e.recording = instrumentationConfig.recording
	if e.recording {
		e.Timestamp = time.Now()
		e.Context.captureHeaders = instrumentationConfig.captureHeaders
		e.Context.sanitizedFieldNames = instrumentationConfig.sanitizedFieldNames
		e.stackTraceLimit = instrumentationConfig.stackTraceLimit
	}

	return &Error{ErrorData: e}
}

// Error describes an error occurring in the monitored service.
type Error struct {
	// ErrorData holds the error data. This field is set to nil when
	// the error's Send method is called.
	*ErrorData

	// cause holds the original error.
	//
	// It is accessible via the Cause method:
	// https://godoc.org/github.com/pkg/errors#Cause
	cause error

	// string holds original error string
	err string
}

// ErrorData holds the details for an error, and is embedded inside Error.
// When the error is sent, its ErrorData field will be set to nil.
type ErrorData struct {
	tracer             *Tracer
	recording          bool
	stackTraceLimit    int
	exception          exceptionData
	log                ErrorLogRecord
	logStacktrace      []stacktrace.Frame
	transactionSampled bool
	transactionType    string

	// ID is the unique identifier of the error. This is set by
	// the various error constructors, and is exposed only so
	// the error ID can be logged or displayed to the user.
	ID ErrorID

	// TraceID is the unique identifier of the trace in which
	// this error occurred. If the error is not associated with
	// a trace, this will be the zero value.
	TraceID TraceID

	// TransactionID is the unique identifier of the transaction
	// in which this error occurred. If the error is not associated
	// with a transaction, this will be the zero value.
	TransactionID SpanID

	// ParentID is the unique identifier of the transaction or span
	// in which this error occurred. If the error is not associated
	// with a transaction or span, this will be the zero value.
	ParentID SpanID

	// Culprit is the name of the function that caused the error.
	//
	// This is initially unset; if it remains unset by the time
	// Send is invoked, and the error has a stacktrace, the first
	// non-library frame in the stacktrace will be considered the
	// culprit.
	Culprit string

	// Timestamp records the time at which the error occurred.
	// This is set when the Error object is created, but may
	// be overridden any time before the Send method is called.
	Timestamp time.Time

	// Handled records whether or not the error was handled. This
	// is ignored by "log" errors with no associated error value.
	Handled bool

	// Context holds the context for this error.
	Context Context
}

// Cause returns original error assigned to Error, nil if Error or Error.cause is nil.
// https://godoc.org/github.com/pkg/errors#Cause
func (e *Error) Cause() error {
	if e != nil {
		return e.cause
	}
	return nil
}

// Error returns string message for error.
// if Error or Error.cause is nil, "[EMPTY]" will be used.
func (e *Error) Error() string {
	if e != nil {
		return e.err
	}
	return "[EMPTY]"
}

// SetTransaction sets TraceID, TransactionID, and ParentID to the transaction's
// IDs, and records the transaction's Type and whether or not it was sampled.
//
// If any custom context has been recorded in tx, it will also be carried across
// to e, but will not override any custom context already recorded on e.
func (e *Error) SetTransaction(tx *Transaction) {
	tx.mu.RLock()
	traceContext := tx.traceContext
	var txType string
	var custom model.IfaceMap
	if !tx.ended() {
		txType = tx.Type
		custom = tx.Context.model.Custom
	}
	tx.mu.RUnlock()
	e.setSpanData(traceContext, traceContext.Span, txType, custom)
}

// SetSpan sets TraceID, TransactionID, and ParentID to the span's IDs.
//
// There is no need to call both SetTransaction and SetSpan. If you do call
// both, then SetSpan must be called second in order to set the error's
// ParentID correctly.
//
// If any custom context has been recorded in s's transaction, it will
// also be carried across to e, but will not override any custom context
// already recorded on e.
func (e *Error) SetSpan(s *Span) {
	var txType string
	var custom model.IfaceMap
	if s.tx != nil {
		s.tx.mu.RLock()
		if !s.tx.ended() {
			txType = s.tx.Type
			custom = s.tx.Context.model.Custom
		}
		s.tx.mu.RUnlock()
	}
	e.setSpanData(s.traceContext, s.transactionID, txType, custom)
}

func (e *Error) setSpanData(
	traceContext TraceContext,
	transactionID SpanID,
	transactionType string,
	customContext model.IfaceMap,
) {
	e.TraceID = traceContext.Trace
	e.ParentID = traceContext.Span
	e.TransactionID = transactionID
	e.transactionSampled = traceContext.Options.Recorded()
	if e.transactionSampled {
		e.transactionType = transactionType
	}
	if n := len(customContext); n != 0 {
		m := len(e.Context.model.Custom)
		e.Context.model.Custom = append(e.Context.model.Custom, customContext...)
		// If there was already custom context in e, shift the custom context from
		// tx to the beginning of the slice so that e's context takes precedence.
		if m != 0 {
			copy(e.Context.model.Custom[n:], e.Context.model.Custom[:m])
			copy(e.Context.model.Custom[:n], customContext)
		}
	}
}

// Send enqueues the error for sending to the Elastic APM server.
//
// Send will set e.ErrorData to nil, so the error must not be
// modified after Send returns.
func (e *Error) Send() {
	if e == nil || e.sent() {
		return
	}
	if e.recording {
		e.ErrorData.enqueue()
	} else {
		e.reset()
	}
	e.ErrorData = nil
}

func (e *Error) sent() bool {
	return e.ErrorData == nil
}

func (e *ErrorData) enqueue() {
	select {
	case e.tracer.events <- tracerEvent{eventType: errorEvent, err: e}:
	default:
		// Enqueuing an error should never block.
		e.tracer.statsMu.Lock()
		e.tracer.stats.ErrorsDropped++
		e.tracer.statsMu.Unlock()
		e.reset()
	}
}

func (e *ErrorData) reset() {
	*e = ErrorData{
		tracer:        e.tracer,
		logStacktrace: e.logStacktrace[:0],
		Context:       e.Context,
		exception:     e.exception,
	}
	e.Context.reset()
	e.exception.reset()
	e.tracer.errorDataPool.Put(e)
}

type exceptionData struct {
	message    string
	stacktrace []stacktrace.Frame
	cause      []exceptionData
	ErrorDetails
}

func (e *exceptionData) reset() {
	*e = exceptionData{
		cause:      e.cause[:0],
		stacktrace: e.stacktrace[:0],
		ErrorDetails: ErrorDetails{
			attrs: e.ErrorDetails.attrs,
			Cause: e.ErrorDetails.Cause[:0],
		},
	}
	for k := range e.attrs {
		delete(e.attrs, k)
	}
}

func initException(e *exceptionData, err error, stackTraceLimit int) {
	b := exceptionDataBuilder{stackTraceLimit: stackTraceLimit}
	b.init(e, err)
}

type exceptionDataBuilder struct {
	stackTraceLimit int
	errorCount      int
	pointerErrors   map[uintptr]struct{}
}

func (b *exceptionDataBuilder) init(e *exceptionData, err error) bool {
	b.errorCount++
	reflectValue := reflect.ValueOf(err)
	reflectType := reflectValue.Type()
	switch reflectType.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		// Prevent infinite recursion due to cyclic error causes.
		ptrVal := reflectValue.Pointer()
		if b.pointerErrors == nil {
			b.pointerErrors = map[uintptr]struct{}{ptrVal: struct{}{}}
		} else {
			if _, ok := b.pointerErrors[ptrVal]; ok {
				return false
			}
			b.pointerErrors[ptrVal] = struct{}{}
		}
	}

	e.message = truncateString(err.Error())
	if e.message == "" {
		e.message = "[EMPTY]"
	}

	namedType := reflectType
	if reflectType.Name() == "" && reflectType.Kind() == reflect.Ptr {
		namedType = reflectType.Elem()
	}
	e.Type.Name = namedType.Name()
	e.Type.PackagePath = namedType.PkgPath()

	// If the error implements Type, use that to
	// override the type name determined through
	// reflection.
	if err, ok := err.(interface {
		Type() string
	}); ok {
		e.Type.Name = err.Type()
	}

	// If the error implements a Code method, use
	// that to set the exception code.
	switch err := err.(type) {
	case interface {
		Code() string
	}:
		e.Code.String = err.Code()
	case interface {
		Code() float64
	}:
		e.Code.Number = err.Code()
	}

	// If the error implements an Unwrap or Cause method, use that to set the cause error.
	// Unwrap is defined by errors wrapped using fmt.Errorf, while Cause is defined by
	// errors wrapped using pkg/errors.Wrap.
	switch err := err.(type) {
	case interface{ Unwrap() error }:
		if cause := err.Unwrap(); cause != nil {
			e.ErrorDetails.Cause = append(e.ErrorDetails.Cause, cause)
		}
	case interface{ Cause() error }:
		if cause := err.Cause(); cause != nil {
			e.ErrorDetails.Cause = append(e.ErrorDetails.Cause, cause)
		}
	}

	// Run registered ErrorDetailers over the error.
	for _, ed := range typeErrorDetailers[reflectType] {
		ed.ErrorDetails(err, &e.ErrorDetails)
	}
	for _, ed := range errorDetailers {
		ed.ErrorDetails(err, &e.ErrorDetails)
	}

	e.Code.String = truncateString(e.Code.String)
	e.Type.Name = truncateString(e.Type.Name)
	e.Type.PackagePath = truncateString(e.Type.PackagePath)
	b.initErrorStacktrace(&e.stacktrace, err)

	for _, err := range e.ErrorDetails.Cause {
		if b.errorCount >= maxErrorTreeNodes {
			break
		}
		var data exceptionData
		if b.init(&data, err) {
			e.cause = append(e.cause, data)
		}
	}
	return true
}

func (b *exceptionDataBuilder) initErrorStacktrace(out *[]stacktrace.Frame, err error) {
	type internalStackTracer interface {
		StackTrace() []stacktrace.Frame
	}
	type errorsStackTracer interface {
		StackTrace() errors.StackTrace
	}
	type runtimeStackTracer interface {
		StackTrace() *runtime.Frames
	}
	switch stackTracer := err.(type) {
	case internalStackTracer:
		stackTrace := stackTracer.StackTrace()
		if b.stackTraceLimit >= 0 && len(stackTrace) > b.stackTraceLimit {
			stackTrace = stackTrace[:b.stackTraceLimit]
		}
		*out = append(*out, stackTrace...)
	case errorsStackTracer:
		stackTrace := stackTracer.StackTrace()
		pkgerrorsutil.AppendStacktrace(stackTrace, out, b.stackTraceLimit)
	case runtimeStackTracer:
		frames := stackTracer.StackTrace()
		count := 0
		for {
			if b.stackTraceLimit >= 0 && count == b.stackTraceLimit {
				break
			}
			frame, more := frames.Next()
			*out = append(*out, stacktrace.RuntimeFrame(frame))
			if !more {
				break
			}
			count++
		}
	}
}

// SetStacktrace sets the stacktrace for the error,
// skipping the first skip number of frames, excluding
// the SetStacktrace function.
func (e *Error) SetStacktrace(skip int) {
	out := &e.exception.stacktrace
	if e.log.Message != "" {
		out = &e.logStacktrace
	}
	*out = stacktrace.AppendStacktrace((*out)[:0], skip+1, e.stackTraceLimit)
}

// ErrorLogRecord holds details of an error log record.
type ErrorLogRecord struct {
	// Message holds the message for the log record,
	// e.g. "failed to connect to %s".
	//
	// If this is empty, "[EMPTY]" will be used.
	Message string

	// MessageFormat holds the non-interpolated format
	// of the log record, e.g. "failed to connect to %s".
	//
	// This is optional.
	MessageFormat string

	// Level holds the severity level of the log record.
	//
	// This is optional.
	Level string

	// LoggerName holds the name of the logger used.
	//
	// This is optional.
	LoggerName string

	// Error is an error associated with the log record.
	//
	// This is optional.
	Error error
}

// ErrorID uniquely identifies an error.
type ErrorID TraceID

// String returns id in its hex-encoded format.
func (id ErrorID) String() string {
	return TraceID(id).String()
}

func init() {
	RegisterErrorDetailer(ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		if errTemporary(err) {
			details.SetAttr("temporary", true)
		}
		if errTimeout(err) {
			details.SetAttr("timeout", true)
		}
	}))
	RegisterTypeErrorDetailer(reflect.TypeOf(&net.OpError{}), ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		opErr := err.(*net.OpError)
		details.SetAttr("op", opErr.Op)
		details.SetAttr("net", opErr.Net)
		if opErr.Source != nil {
			if addr := opErr.Source; addr != nil {
				details.SetAttr("source", fmt.Sprintf("%s:%s", addr.Network(), addr.String()))
			}
		}
		if opErr.Addr != nil {
			if addr := opErr.Addr; addr != nil {
				details.SetAttr("addr", fmt.Sprintf("%s:%s", addr.Network(), addr.String()))
			}
		}
		details.Cause = append(details.Cause, opErr.Err)
	}))
	RegisterTypeErrorDetailer(reflect.TypeOf(&os.LinkError{}), ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		linkErr := err.(*os.LinkError)
		details.SetAttr("op", linkErr.Op)
		details.SetAttr("old", linkErr.Old)
		details.SetAttr("new", linkErr.New)
		details.Cause = append(details.Cause, linkErr.Err)
	}))
	RegisterTypeErrorDetailer(reflect.TypeOf(&os.PathError{}), ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		pathErr := err.(*os.PathError)
		details.SetAttr("op", pathErr.Op)
		details.SetAttr("path", pathErr.Path)
		details.Cause = append(details.Cause, pathErr.Err)
	}))
	RegisterTypeErrorDetailer(reflect.TypeOf(&os.SyscallError{}), ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		syscallErr := err.(*os.SyscallError)
		details.SetAttr("syscall", syscallErr.Syscall)
		details.Cause = append(details.Cause, syscallErr.Err)
	}))
	RegisterTypeErrorDetailer(reflect.TypeOf(syscall.Errno(0)), ErrorDetailerFunc(func(err error, details *ErrorDetails) {
		errno := err.(syscall.Errno)
		details.Code.String = errnoName(errno)
		if details.Code.String == "" {
			details.Code.Number = float64(errno)
		}
	}))
}

func errTemporary(err error) bool {
	type temporaryError interface {
		Temporary() bool
	}
	terr, ok := err.(temporaryError)
	return ok && terr.Temporary()
}

func errTimeout(err error) bool {
	type timeoutError interface {
		Timeout() bool
	}
	terr, ok := err.(timeoutError)
	return ok && terr.Timeout()
}

// RegisterTypeErrorDetailer registers e to be called for any error with
// the concrete type t.
//
// Each ErrorDetailer registered in this way will be called, in the order
// registered, for each error of type t created via Tracer.NewError or
// Tracer.NewErrorLog.
//
// RegisterTypeErrorDetailer must not be called during tracer operation;
// it is intended to be called at package init time.
func RegisterTypeErrorDetailer(t reflect.Type, e ErrorDetailer) {
	typeErrorDetailers[t] = append(typeErrorDetailers[t], e)
}

// RegisterErrorDetailer registers e in the global list of ErrorDetailers.
//
// Each ErrorDetailer registered in this way will be called, in the order
// registered, for each error created via Tracer.NewError or Tracer.NewErrorLog.
//
// RegisterErrorDetailer must not be called during tracer operation; it is
// intended to be called at package init time.
func RegisterErrorDetailer(e ErrorDetailer) {
	errorDetailers = append(errorDetailers, e)
}

var (
	typeErrorDetailers = make(map[reflect.Type][]ErrorDetailer)
	errorDetailers     []ErrorDetailer
)

// ErrorDetails holds details of an error, which can be altered or
// extended by registering an ErrorDetailer with RegisterErrorDetailer
// or RegisterTypeErrorDetailer.
type ErrorDetails struct {
	attrs map[string]interface{}

	// Type holds information about the error type, initialized
	// with the type name and type package path using reflection.
	Type struct {
		// Name holds the error type name.
		Name string

		// PackagePath holds the error type package path.
		PackagePath string
	}

	// Code holds an error code.
	Code struct {
		// String holds a string-based error code. If this is set, then Number is ignored.
		//
		// This field will be initialized to the result of calling an error's Code method,
		// if the error implements the following interface:
		//
		//     type interface StringCoder {
		//         Code() string
		//     }
		String string

		// Number holds a numerical error code. This is ignored if String is set.
		//
		// This field will be initialized to the result of calling an error's Code
		// method, if the error implements the following interface:
		//
		//     type interface NumberCoder {
		//         Code() float64
		//     }
		Number float64
	}

	// Cause holds the errors that were the cause of this error.
	Cause []error
}

// SetAttr sets the attribute with key k to value v.
func (d *ErrorDetails) SetAttr(k string, v interface{}) {
	if d.attrs == nil {
		d.attrs = make(map[string]interface{})
	}
	d.attrs[k] = v
}

// ErrorDetailer defines an interface for altering or extending the ErrorDetails for an error.
//
// ErrorDetailers can be registered using the package-level functions RegisterErrorDetailer and
// RegisterTypeErrorDetailer.
type ErrorDetailer interface {
	// ErrorDetails is called to update or alter details for err.
	ErrorDetails(err error, details *ErrorDetails)
}

// ErrorDetailerFunc is a function type implementing ErrorDetailer.
type ErrorDetailerFunc func(error, *ErrorDetails)

// ErrorDetails calls f(err, details).
func (f ErrorDetailerFunc) ErrorDetails(err error, details *ErrorDetails) {
	f(err, details)
}
