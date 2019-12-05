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

package timestamp

import (
	"fmt"
	"io"
	"time"
)

type parseError struct {
	field  string
	time   interface{}
	causes []error
}

func (e *parseError) Error() string {
	return fmt.Sprintf("failed parsing time field %v='%v'", e.field, e.time)
}

// Errors returns a list of parse errors. This implements the errorGroup
// interface that the logger recognizes for including a list of causes.
func (e *parseError) Errors() []error {
	return e.causes
}

func (e *parseError) Format(f fmt.State, c rune) {
	io.WriteString(f, e.Error())

	if c == 'v' && f.Flag('+') {
		f.Write([]byte(": "))
		first := true
		for _, item := range e.causes {
			if first {
				first = false
			} else {
				f.Write([]byte("; "))
			}
			io.WriteString(f, item.Error())
		}
	}
}

type parseErrorCause struct {
	*time.ParseError
}

func (e *parseErrorCause) Error() string {
	if e.Message != "" {
		return e.Message
	}

	return "failed using layout [" + e.Layout + "] " +
		"cannot parse [" + e.ValueElem + "] as [" + e.LayoutElem + "]"
}
