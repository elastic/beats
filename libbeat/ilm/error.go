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

func (e *Error) Reason() error { return e.reason }

func (e *Error) Error() string {
	msg := e.message
	if e.message == "" {
		msg = e.reason.Error()
	}

	if e.cause != nil {
		return fmt.Sprintf("%v: %v", msg, e.cause.Error())
	}
	return msg
}
