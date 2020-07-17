// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package retry

// Fatal in retry package is an interface each error needs to implement
// in order to say whether or not it is fatal.
type Fatal interface {
	Fatal() bool
}

// FatalError wraps an error and is always fatal
type FatalError struct {
	error
}

// Fatal determines whether or not error is fatal
func (*FatalError) Fatal() bool {
	return true
}

// ErrorMakeFatal is a shorthand for making an error fatal
func ErrorMakeFatal(err error) error {
	if err == nil {
		return err
	}

	return FatalError{err}
}
