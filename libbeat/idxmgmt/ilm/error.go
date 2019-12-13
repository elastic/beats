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

package ilm

import (
	"errors"
	"fmt"
)

// Error indicates an error + reason describing the last error.
// The Reason() method returns a sentinal error value for comparison.
type Error struct {
	reason  error
	cause   error
	message string
}

var (
	ErrESVersionNotSupported = errors.New("ILM is not supported by the Elasticsearch version in use")
	ErrILMCheckRequestFailed = errors.New("request checking for ILM availability failed")
	ErrInvalidResponse       = errors.New("invalid response received")
	ErrESILMDisabled         = errors.New("ILM is disabled in Elasticsearch")
	ErrRequestFailed         = errors.New("request failed")
	ErrAliasAlreadyExists    = errors.New("alias already exists")
	ErrAliasCreateFailed     = errors.New("failed to create write alias")
	ErrInvalidAlias          = errors.New("invalid alias")
	ErrOpNotAvailable        = errors.New("operation not available")
)

func errOf(reason error) error {
	return &Error{reason: reason}
}

func errf(reason error, msg string, vs ...interface{}) error {
	return wrapErrf(nil, reason, msg, vs...)
}

func wrapErr(cause, reason error) error {
	return wrapErrf(cause, reason, "")
}

func wrapErrf(cause, reason error, msg string, vs ...interface{}) error {
	return &Error{
		cause:   cause,
		reason:  reason,
		message: fmt.Sprintf(msg, vs...),
	}
}

// ErrReason calls Reason() if the error implements this method. Otherwise return nil.
func ErrReason(err error) error {
	if err == nil {
		return nil
	}

	ifc, ok := err.(interface{ Reason() error })
	if !ok {
		return nil
	}
	return ifc.Reason()
}

// Cause returns the errors cause, if present.
func (e *Error) Cause() error { return e.cause }

// Reason returns a sentinal error value define within the ilm package.
func (e *Error) Reason() error { return e.reason }

// Error returns the formatted error string.
func (e *Error) Error() string {
	msg := e.message
	if e.message == "" {
		msg = e.reason.Error()
	}

	if e.cause != nil {
		return fmt.Sprintf("%v: %+v", msg, e.cause)
	}
	return msg
}
