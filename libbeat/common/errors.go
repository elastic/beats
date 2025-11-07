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

package common

import (
	"errors"
	"fmt"
)

// ErrInputNotFinished struct for reporting errors related to not finished inputs
type ErrInputNotFinished struct {
	State string
	File  string
}

// Error method of ErrInputNotFinished
func (e *ErrInputNotFinished) Error() string {
	return fmt.Sprintf("Can only start an input when all related states are finished: %+v", e.State)
}

type ErrNonReloadable struct {
	Err error
}

func (e ErrNonReloadable) Error() string {
	return fmt.Sprintf("ErrNonReloadable: %v", e.Err)
}

func (e ErrNonReloadable) Unwrap() error { return e.Err }

func (e ErrNonReloadable) Is(err error) bool {
	switch err.(type) {
	case ErrNonReloadable:
		return true
	default:
		return errors.Is(e.Err, err)
	}
}

// IsInputReloadable returns true if err, or any error wrapped
// by err can be retried.
//
// Effectively, it will only return false if ALL
// errors are ErrNonReloadable.
func IsInputReloadable(err error) bool {
	if err == nil {
		return false
	}

	type unwrapList interface {
		Unwrap() []error
	}

	//nolint:errorlint // we only want to check that specific error, not all errors in the chain
	errList, isErrList := err.(unwrapList)
	if !isErrList {
		return !errors.Is(err, ErrNonReloadable{})
	}

	for _, e := range errList.Unwrap() {
		if !errors.Is(e, ErrNonReloadable{}) {
			return true
		}
	}

	return false
}
