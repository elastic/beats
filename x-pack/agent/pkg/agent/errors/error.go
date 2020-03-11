// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import "github.com/pkg/errors"

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

// Check it implements Error
var _ Error = agentError{}

// Check it implements error
var _ error = agentError{}
