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

package ucfg

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
)

// Error type returned by all public functions in go-ucfg.
type Error interface {
	error

	// error class, one of ErrConfig, ErrImplementation, ErrUnknown
	Class() error

	// The internal reason error code like ErrMissing, ErrRequired,
	// ErrTypeMismatch and others.
	Reason() error

	// The error message.
	Message() string

	// [optional] path of config element error occurred for
	Path() string

	// [optional] stack trace
	Trace() string
}

type baseError struct {
	reason  error
	class   error
	message string
	path    string
}

type criticalError struct {
	baseError
	trace string
}

// Error Reasons
var (
	ErrMissing = errors.New("missing field")

	ErrNoParse = errors.New("parsing dynamic configs is disabled")

	ErrCyclicReference = errors.New("cyclic reference detected")

	ErrDuplicateValidator = errors.New("validator already registered")

	ErrTypeNoArray = errors.New("field is no array")

	ErrTypeMismatch = errors.New("type mismatch")

	ErrKeyTypeNotString = errors.New("key must be a string")

	ErrIndexOutOfRange = errors.New("out of range index")

	ErrPointerRequired = errors.New("pointer required for unpacking configurations")

	ErrArraySizeMistach = errors.New("Array size mismatch")

	ErrExpectedObject = errors.New("expected object")

	ErrNilConfig = errors.New("config is nil")

	ErrNilValue = errors.New("nil value is invalid")

	ErrTODO = errors.New("TODO - implement me")

	ErrDuplicateKey = errors.New("duplicate key")

	ErrOverflow = errors.New("integer overflow")

	ErrNegative = errors.New("negative value")

	ErrZeroValue = errors.New("zero value")

	ErrRequired = errors.New("missing required field")

	ErrEmpty = errors.New("empty field")

	ErrArrayEmpty = errors.New("empty array")

	ErrMapEmpty = errors.New("empty map")

	ErrRegexEmpty = errors.New("regex value is not set")

	ErrStringEmpty = errors.New("string value is not set")
)

// Error Classes
var (
	ErrConfig         = errors.New("Configuration error")
	ErrImplementation = errors.New("Implementation error")
	ErrUnknown        = errors.New("Unspecified")
)

func (e baseError) Error() string { return e.Message() }
func (e baseError) Reason() error { return e.reason }
func (e baseError) Class() error  { return e.class }
func (e baseError) Trace() string { return "" }
func (e baseError) Path() string  { return e.path }

func (e baseError) Message() string {
	if e.message == "" {
		return e.reason.Error()
	}
	return e.message
}

func (e criticalError) Trace() string { return e.trace }

func (e criticalError) Error() string {
	return fmt.Sprintf("%s\nTrace:%v\n", e.baseError.Message(), e.trace)
}

func raiseErr(reason error, message string) Error {
	return baseError{
		reason:  reason,
		message: message,
		class:   ErrConfig,
	}
}

func raiseImplErr(reason error) Error {
	return baseError{
		reason: reason,
		class:  ErrImplementation,
	}
}

func raiseCritical(reason error, message string) Error {
	if message == "" {
		message = reason.Error()
	}
	if message != "" {
		message = fmt.Sprintf("(assert) %v", message)
	}
	return criticalError{
		baseError{reason, ErrImplementation, message, ""},
		string(debug.Stack()),
	}
}

func raisePathErr(reason error, meta *Meta, message, path string) Error {
	message = messagePath(reason, meta, message, path)
	return baseError{reason, ErrConfig, message, path}
}

func messageMeta(message string, meta *Meta) string {
	if meta == nil || meta.Source == "" {
		return message
	}
	return fmt.Sprintf("%v (source:'%v')", message, meta.Source)
}

func messagePath(reason error, meta *Meta, message, path string) string {
	if path == "" {
		path = "config"
	} else {
		path = fmt.Sprintf("'%v'", path)
	}

	if message == "" {
		message = reason.Error()
	}

	message = fmt.Sprintf("%v accessing %v", message, path)
	return messageMeta(message, meta)
}

func raiseDuplicateKey(cfg *Config, name string) Error {
	return raisePathErr(ErrDuplicateKey, cfg.metadata, "", cfg.PathOf(name, "."))
}

func raiseCyclicErr(field string) Error {
	message := fmt.Sprintf("cyclic reference detected for key: '%s'", field)

	return baseError{
		reason:  ErrCyclicReference,
		class:   ErrConfig,
		message: message,
	}
}

func raiseMissing(c *Config, field string) Error {
	// error reading field from config, as missing in c
	return raiseMissingMsg(c, field, "")
}

func raiseMissingMsg(c *Config, field string, message string) Error {
	return raisePathErr(ErrMissing, c.metadata, message, c.PathOf(field, "."))
}

func raiseMissingArr(ctx context, meta *Meta, idx int) Error {
	message := fmt.Sprintf("no value in array at %v", idx)
	return raisePathErr(ErrMissing, meta, message, ctx.path("."))
}

func raiseIndexOutOfBounds(opts *options, value value, idx int) Error {
	reason := ErrIndexOutOfRange
	ctx := value.Context()
	len, _ := value.Len(opts)
	message := fmt.Sprintf("index '%v' out of range (length=%v)", idx, len)
	return raisePathErr(reason, value.meta(), message, ctx.path("."))
}

func raiseInvalidTopLevelType(v interface{}, meta *Meta) Error {
	// could be developers or user fault
	t := chaseTypePointers(chaseValue(reflect.ValueOf(v)).Type())
	message := fmt.Sprintf("type '%v' is not supported on top level of config, only dictionary or list", t)
	return raiseErr(ErrTypeMismatch, messageMeta(message, meta))
}

func raiseKeyInvalidTypeUnpack(t reflect.Type, from *Config) Error {
	// most likely developers fault
	ctx := from.ctx
	reason := ErrKeyTypeNotString
	message := fmt.Sprintf("string key required when unpacking into '%v'", t)
	return raiseCritical(reason, messagePath(reason, from.metadata, message, ctx.path(".")))
}

func raiseKeyInvalidTypeMerge(cfg *Config, t reflect.Type) Error {
	ctx := cfg.ctx
	reason := ErrKeyTypeNotString
	message := fmt.Sprintf("string key required when merging into '%v'", t)
	return raiseCritical(reason, messagePath(reason, cfg.metadata, message, ctx.path(".")))
}

func raiseSquashNeedsObject(cfg *Config, opts *options, f string, t reflect.Type) Error {
	reason := ErrTypeMismatch
	message := fmt.Sprintf("require map or struct when squash merging '%v' (%v)", f, t)

	return raiseCritical(reason, messagePath(reason, opts.meta, message, cfg.Path(".")))
}

func raiseInlineNeedsObject(cfg *Config, f string, t reflect.Type) Error {
	reason := ErrTypeMismatch
	message := fmt.Sprintf("require map or struct when inling '%v' (%v)", f, t)
	return raiseCritical(reason,
		messagePath(reason, cfg.metadata, message, cfg.Path(".")))
}

func raiseUnsupportedInputType(ctx context, meta *Meta, v reflect.Value) Error {
	reason := ErrTypeMismatch
	message := fmt.Sprintf("unspported input type (%v) with value '%#v'",
		v.Type(), v)

	return raiseCritical(reason, messagePath(reason, meta, message, ctx.path(".")))
}

func raiseNoParse(ctx context, meta *Meta) Error {
	reason := ErrNoParse
	return raisePathErr(reason, meta, "", ctx.path("."))
}

func raiseNil(reason error) Error {
	// programmers error (passed unexpected nil pointer)
	return raiseCritical(reason, "")
}

func raisePointerRequired(v reflect.Value) Error {
	// developer did not pass pointer, unpack target is not settable
	return raiseCritical(ErrPointerRequired, "")
}

func raiseToTypeNotSupported(opts *options, v value, goT reflect.Type) Error {
	reason := ErrTypeMismatch
	t, _ := v.typ(opts)
	message := fmt.Sprintf("value of type '%v' not convertible into unsupported go type '%v'",
		t.name, goT)
	ctx := v.Context()

	return raiseCritical(reason, messagePath(reason, v.meta(), message, ctx.path(".")))
}

func raiseArraySize(ctx context, meta *Meta, n int, to int) Error {
	reason := ErrArraySizeMistach
	message := fmt.Sprintf("array of length %v does not meet required length %v",
		n, to)

	return raisePathErr(reason, meta, message, ctx.path("."))
}

func raiseConversion(opts *options, v value, err error, to string) Error {
	ctx := v.Context()
	path := ctx.path(".")
	t, _ := v.typ(opts)
	message := fmt.Sprintf("can not convert '%v' into '%v'", t.name, to)
	return raisePathErr(err, v.meta(), message, path)
}

func raiseExpectedObject(opts *options, v value) Error {
	ctx := v.Context()
	path := ctx.path(".")
	t, _ := v.typ(opts)
	message := fmt.Sprintf("required 'object', but found '%v' in field '%v'",
		t.name, path)

	return raiseErr(ErrExpectedObject, messageMeta(message, v.meta()))
}

func raiseInvalidDuration(v value, err error) Error {
	ctx := v.Context()
	path := ctx.path(".")
	return raisePathErr(err, v.meta(), "", path)
}

func raiseValidation(ctx context, meta *Meta, field string, err error) Error {
	path := ""
	if field == "" {
		path = ctx.path(".")
	} else {
		path = ctx.pathOf(field, ".")
	}
	return raiseErr(err, messagePath(err, meta, err.Error(), path))
}

func raiseInvalidRegexp(v value, err error) Error {
	ctx := v.Context()
	path := ctx.path(".")
	message := fmt.Sprintf("Failed to compile regular expression with '%v'", err)
	return raisePathErr(err, v.meta(), message, path)
}

func raiseParseSplice(ctx context, meta *Meta, err error) Error {
	message := fmt.Sprintf("%v parsing splice", err)
	return raisePathErr(err, meta, message, ctx.path("."))
}
