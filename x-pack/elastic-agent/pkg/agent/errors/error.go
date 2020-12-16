// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	goerrors "errors"
	"reflect"

	"github.com/pkg/errors"
)

// As is just a helper so user dont have to use multiple imports for errors.
func As(err error, target interface{}) bool {
	return goerrors.As(err, target)
}

// Is is just a helper so user dont have to use multiple imports for errors.
func Is(err, target error) bool {
	return goerrors.Is(err, target)
}

// Unwrap is just a helper so user dont have to use multiple imports for errors.
func Unwrap(err error) error {
	return goerrors.Unwrap(err)
}

// MetaRecord is a entry of metadata enhancing an error.
type MetaRecord struct {
	key string
	val interface{}
}

// Error is an interface defining custom agent error.
type Error interface {
	Error() string
	Type() ErrorType
	ReadableType() string
	Meta() map[string]interface{}
}

type agentError struct {
	msg     string
	err     error
	errType ErrorType
	meta    map[string]interface{}
}

// Unwrap returns nested error.
func (e agentError) Unwrap() error {
	return e.err
}

// Error returns a string consisting of a message and originating error.
func (e agentError) Error() string {
	if e.msg != "" {
		return errors.Wrap(e.err, e.msg).Error()
	}

	return e.err.Error()
}

// Type recursively checks errors and return first known not default error type.
func (e agentError) Type() ErrorType {
	if e.errType != 0 {
		return e.errType
	}

	if e.err == nil {
		return TypeUnexpected
	}

	inner, ok := e.err.(Error)
	if causeErr := errors.Cause(e.err); !ok && causeErr == e.err {
		return TypeUnexpected
	} else if !ok {
		// err is wrapped
		customCause := New(causeErr).(Error)
		return customCause.Type()
	}

	return inner.Type()
}

// ReadableType recursively checks errors and return first known not default
// error type and returns its readable representation.
func (e agentError) ReadableType() string {
	etype := e.Type()
	if hrt, found := readableTypes[etype]; found {
		return hrt
	}

	return "UNEXPECTED"
}

func (e agentError) Meta() map[string]interface{} {
	inner, ok := e.err.(Error)
	if causeErr := errors.Cause(e.err); !ok && causeErr == e.err {
		return e.meta
	} else if !ok {
		inner = New(causeErr).(Error)
	}

	innerMeta := inner.Meta()
	resultingMeta := make(map[string]interface{})

	// copy so we don't modify values
	for k, v := range e.meta {
		resultingMeta[k] = v
	}

	for k, v := range innerMeta {
		if _, found := resultingMeta[k]; found {
			continue
		}

		resultingMeta[k] = v
	}

	return resultingMeta
}

// Equal compares errors and evaluates if they are the same or not.
// Agent error is not comparable due to included map so we need to
// do the heavy lifting ourselves.
func (e agentError) Equal(target error) bool {
	targetErr, ok := target.(agentError)
	if !ok {
		return false
	}

	return errors.Is(e.err, targetErr.err) &&
		e.errType == targetErr.errType &&
		e.msg == targetErr.msg &&
		reflect.DeepEqual(e.meta, targetErr.meta)

}

// Is checks whether agent err is an err.
func (e agentError) Is(target error) bool {
	if agentErr, ok := target.(agentError); ok {
		return e.Equal(agentErr)
	}

	return goerrors.Is(e.err, target)
}

// Check it implements Error
var _ Error = agentError{}

// Check it implements error
var _ error = agentError{}
