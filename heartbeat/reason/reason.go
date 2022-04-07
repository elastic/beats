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

package reason

import "github.com/elastic/beats/v8/libbeat/common"

type Reason interface {
	error
	Type() string
	Unwrap() error
}

type ValidateError struct {
	err error
}

type IOError struct {
	err error
}

func ValidateFailed(err error) Reason {
	if err == nil {
		return nil
	}
	return ValidateError{err}
}

func IOFailed(err error) Reason {
	if err == nil {
		return nil
	}
	return IOError{err}
}

func (e ValidateError) Error() string { return e.err.Error() }
func (e ValidateError) Unwrap() error { return e.err }
func (ValidateError) Type() string    { return "validate" }

func (e IOError) Error() string { return e.err.Error() }
func (e IOError) Unwrap() error { return e.err }
func (IOError) Type() string    { return "io" }

func FailError(typ string, err error) common.MapStr {
	return common.MapStr{
		"type":    typ,
		"message": err.Error(),
	}
}

func Fail(r Reason) common.MapStr {
	return common.MapStr{
		"type":    r.Type(),
		"message": r.Error(),
	}
}

func FailIO(err error) common.MapStr { return Fail(IOError{err}) }

// MakeValidateError creates an instance of ValidateError from the given error.
func MakeValidateError(err error) ValidateError {
	return ValidateError{err}
}
