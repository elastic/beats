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

package ecs

// These fields can represent errors of any kind.
// Use them for errors that happen while fetching events or in cases where the
// event itself contains an error.
type Error struct {
	// Unique identifier for the error.
	ID string `ecs:"id"`

	// Error message.
	Message string `ecs:"message"`

	// Error code describing the error.
	Code string `ecs:"code"`

	// The type of the error, for example the class name of the exception.
	Type string `ecs:"type"`

	// The stack trace of this error in plain text.
	StackTrace string `ecs:"stack_trace"`
}
