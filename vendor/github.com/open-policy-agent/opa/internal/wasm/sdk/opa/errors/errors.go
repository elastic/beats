// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package errors

import (
	"errors"
	"fmt"
)

const (
	// InvalidConfigErr is the error code returned if the OPA initialization fails due to an invalid config.
	InvalidConfigErr string = "invalid_config"

	// InvalidPolicyOrDataErr is the error code returned if either policy or data is invalid.
	InvalidPolicyOrDataErr string = "invalid_policy_or_data"

	// InvalidBundleErr is the error code returned if the bundle loaded is corrupted.
	InvalidBundleErr string = "invalid_bundle"

	// NotReadyErr is the error code returned if the OPA instance is not initialized.
	NotReadyErr string = "not_ready"

	// InternalErr is the error code returned if the evaluation fails due to an internal error.
	InternalErr string = "internal_error"

	// CancelledErr is the error code returned if the evaluation is cancelled.
	CancelledErr string = "cancelled"
)

// Error is the error code type returned by the SDK functions when an error occurs.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

// New returns a new error with the passed code
func New(code, msg string) error {
	switch code {
	case InvalidConfigErr, InvalidPolicyOrDataErr, InvalidBundleErr, NotReadyErr, InternalErr, CancelledErr:
		return &Error{Code: code, Message: msg}
	default:
		panic("unknown error code: " + code)
	}
}

// IsError returns true if the err is an Error.
func IsError(err error) bool {
	return errorHasCode(err, "")
}

func errorHasCode(err error, code string) bool {
	return errors.Is(err, &Error{Code: code})
}

// IsCancel returns true if err was caused by cancellation.
func IsCancel(err error) bool {
	return errorHasCode(err, CancelledErr)
}

// Is allows matching error types using errors.Is (see IsCancel).
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return (t.Code == "" || e.Code == t.Code) &&
			(t.Message == "" || e.Message == t.Message)
	}
	return false
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%v: %v", e.Code, e.Message)
}
