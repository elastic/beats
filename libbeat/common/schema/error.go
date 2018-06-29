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

import "fmt"

const (
	RequiredType ErrorType = iota
	OptionalType ErrorType = iota
)

type ErrorType int

type Error struct {
	key       string
	message   string
	errorType ErrorType
}

func NewError(key string, message string) *Error {
	return &Error{
		key:       key,
		message:   message,
		errorType: RequiredType,
	}
}

func (err *Error) SetType(errorType ErrorType) {
	err.errorType = errorType
}

func (err *Error) IsType(errorType ErrorType) bool {
	return err.errorType == errorType
}

func (err *Error) Error() string {
	return fmt.Sprintf("Missing field: %s, Error: %s", err.key, err.message)
}
