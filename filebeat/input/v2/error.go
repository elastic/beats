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

package v2

import (
	"errors"
	"fmt"
	"strings"

	"github.com/urso/diag"
)

// LoaderError is returned by Loaders in case of failures.
type LoaderError struct {
	// Additional metadata for structure logging (if applicable)
	Diagnostics *diag.Context

	// Name of input/module that failed to load (if applicable)
	Name string

	// Reason why the loader failed. Can either be the cause reported by the
	// Plugin or some other indicator like ErrUnknown
	Reason error

	// (optional) Message to report in additon.
	Message string
}

// ErrUnknown indicates that the plugin type does not exist. Either
// because the 'type' setting name does not match the loaders expectations,
// or because the type is unknown.
var ErrUnknown = errors.New("unknown input type")

// ErrNoInputConfigured indicates that the 'type' setting is missing.
var ErrNoInputConfigured = errors.New("no input type configured")

// ErrInfiniteLoadLoop is reported by loaders supporting recursive input
// configurations (inputs referencing other inputs). The error indicates
// that the loader has detected a loop that most not be.
var ErrInfiniteLoadLoop = errors.New("infinite load loop detected")

// ErrPluginWithoutName reports that the operation failed because
// the plugin is required to have a Name.
var ErrPluginWithoutName = errors.New("the plugin has no name")

// IsUnknownInputError checks if an error value indicates an input load
// error because there is no existing plugin that can create the input.
func IsUnknownInputError(err error) bool { return errors.Is(err, ErrUnknown) }

func failedInputName(err error) string {
	switch e := err.(type) {
	case *LoaderError:
		return e.Name
	default:
		return ""
	}
}

// Context returns the errors diagnostics if present
func (e *LoaderError) Context() *diag.Context { return e.Diagnostics }

// Unwrap returns the reason if present
func (e *LoaderError) Unwrap() error { return e.Reason }

// Error returns the errors string repesentation
func (e *LoaderError) Error() string {
	var buf strings.Builder

	if e.Message != "" {
		buf.WriteString(e.Message)
	} else if e.Name != "" {
		buf.WriteString("failed to load ")
		buf.WriteString(e.Name)
	}

	if e.Reason != nil {
		if buf.Len() > 0 {
			buf.WriteString(": ")
		}
		fmt.Fprintf(&buf, "%v", e.Reason)
	}

	if buf.Len() == 0 {
		return "<loader error>"
	}
	return buf.String()
}
