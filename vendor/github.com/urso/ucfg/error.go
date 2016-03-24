package ucfg

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
)

type Error interface {
	error
	Reason() error

	// error class, one of ErrConfig, ErrImplementation, ErrUnknown
	Class() error

	Message() string

	// [optional] path of config element error occured for
	Path() string

	// [optional] stack trace
	Trace() string
}

type baseError struct {
	reason  error
	class   error
	message string
}

type criticalError struct {
	baseError
	trace string
}

type pathError struct {
	baseError
	meta *Meta
	path string
}

// error Reasons
var (
	ErrMissing = errors.New("missing field")

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

	ErrDuplicateKeey = errors.New("duplicate key")
)

// error classes
var (
	ErrConfig         = errors.New("Configuration error")
	ErrImplementation = errors.New("Implementation error")
	ErrUnknown        = errors.New("Unspecified")
)

func (e baseError) Message() string { return e.Error() }
func (e baseError) Reason() error   { return e.reason }
func (e baseError) Class() error    { return e.class }
func (e baseError) Trace() string   { return "" }
func (e baseError) Path() string    { return "" }

func (e baseError) Error() string {
	if e.message == "" {
		return e.reason.Error()
	}
	return e.message
}

func (e criticalError) Error() string {
	return fmt.Sprintf("%s\nTrace:%v\n", e.baseError, e.trace)
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
		baseError{reason, ErrImplementation, message},
		string(debug.Stack()),
	}
}

func raisePathErr(reason error, meta *Meta, message, path string) Error {
	// fmt.Printf("path err, reason='%v', meta=%v, message='%v', path='%v'\n", reason, meta, message, path)
	message = messagePath(reason, meta, message, path)
	// fmt.Printf("  -> report message: %v\n", message)

	return pathError{
		baseError{reason, ErrConfig, message},
		meta,
		path,
	}
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
	return raisePathErr(ErrDuplicateKeey, cfg.metadata, "", cfg.PathOf(name, "."))
}

func raiseMissing(c *Config, field string) Error {
	// error reading field from config, as missing in c
	return raisePathErr(ErrMissing, c.metadata, "", c.PathOf(field, "."))
}

func raiseMissingArr(arr *cfgArray, idx int) Error {
	message := fmt.Sprintf("no value in array at %v", idx)
	return raisePathErr(ErrMissing, arr.meta(), message, arr.ctx.path("."))
}

func raiseIndexOutOfBounds(value value, idx int) Error {
	reason := ErrIndexOutOfRange
	ctx := value.Context()
	message := fmt.Sprintf("index '%v' out of range (length=%v)", idx, value.Len())
	return raisePathErr(reason, value.meta(), message, ctx.path("."))
}

func raiseInvalidTopLevelType(v interface{}) Error {
	// most likely developers fault
	t := chaseTypePointers(chaseValue(reflect.ValueOf(v)).Type())
	message := fmt.Sprintf("can not use go type '%v' for merging/unpacking configurations", t)
	return raiseCritical(ErrTypeMismatch, message)
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

func raiseSquashNeedsObject(cfg *Config, opts options, f string, t reflect.Type) Error {
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

func raiseUnsupportedInputType(ctx context, opts options, v reflect.Value) Error {
	reason := ErrTypeMismatch
	message := fmt.Sprintf("unspported input type (%v) with value '%#v'",
		v.Type(), v)

	return raiseCritical(reason, messagePath(reason, opts.meta, message, ctx.path(".")))
}

func raiseNil(reason error) Error {
	// programmers error (passed unexpected nil pointer)
	return raiseCritical(reason, "")
}

func raisePointerRequired(v reflect.Value) Error {
	// developer did not pass pointer, unpack target is not settable
	return raiseCritical(ErrPointerRequired, "")
}

func raiseToTypeNotSupported(v value, t reflect.Type) Error {
	reason := ErrTypeMismatch
	message := fmt.Sprintf("value of type '%v' not convertible into unsupported go type '%v'", v.typeName(), t)
	ctx := v.Context()

	return raiseCritical(reason, messagePath(reason, v.meta(), message, ctx.path(".")))
}

func raiseArraySize(to reflect.Type, arr *cfgArray) Error {
	reason := ErrArraySizeMistach
	message := fmt.Sprintf("array of length %v does not meet required length %v",
		arr.Len(), to.Len())

	return raisePathErr(reason, arr.meta(), message, arr.ctx.path("."))
}

func raiseConversion(v value, err error, to string) Error {
	ctx := v.Context()
	path := ctx.path(".")
	message := fmt.Sprintf("can not convert '%v' into '%v'", v.typeName(), to)
	return raisePathErr(err, v.meta(), message, path)
}

func raiseExpectedObject(v value) Error {
	ctx := v.Context()
	path := ctx.path(".")
	message := fmt.Sprintf("required 'object', but found '%v' in field '%v'",
		v.typeName(), path)

	return raiseErr(ErrExpectedObject, messageMeta(message, v.meta()))
}
