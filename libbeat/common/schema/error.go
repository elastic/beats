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

package schema

import (
	"fmt"
)

// KeyError is an error with a field key
type KeyError interface {
	Key() string
	SetKey(k string)
}

type errorKey struct {
	key string
}

// Key returns the value of the field key
func (k *errorKey) Key() string {
	return k.key
}

// SetKey sets the value of the field key
func (k *errorKey) SetKey(v string) {
	k.key = v
}

// KeyNotFoundError is an error happening when a field key is not found
type KeyNotFoundError struct {
	errorKey

	Err      error
	Optional bool
	Required bool
}

// NewKeyNotFoundError builds a KeyNotFoundError
func NewKeyNotFoundError(key string) *KeyNotFoundError {
	var e KeyNotFoundError
	e.SetKey(key)
	return &e
}

// Error returns the error message of a KeyNotFoundError
func (err *KeyNotFoundError) Error() string {
	msg := fmt.Sprintf("key `%s` not found", err.Key())
	if err.Err != nil {
		msg += ": " + err.Err.Error()
	}
	return msg
}

// WrongFormatError is an error happening when a field format is incorrect
type WrongFormatError struct {
	errorKey

	Msg string
}

// NewWrongFormatError builds a new WrongFormatError
func NewWrongFormatError(key string, msg string) *WrongFormatError {
	e := WrongFormatError{
		Msg: msg,
	}
	e.SetKey(key)
	return &e
}

// Error returns the error message of a WrongFormatError
func (err *WrongFormatError) Error() string {
	msg := fmt.Sprintf("wrong format in `%s`", err.Key())
	if err.Msg != "" {
		msg += ": " + err.Msg
	}
	return msg
}
