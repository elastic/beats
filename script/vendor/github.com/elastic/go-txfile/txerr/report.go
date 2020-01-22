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

package txerr

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-txfile/internal/strbld"
)

// Format provides a common formatting implementation for adding the
// fmt.Formatter interface to custom errors.
// Usage:
//    func (e *myError) Format(s fmt.State, c rune) { txerr.Format(e, s, c) }
func Format(err error, s fmt.State, c rune) {
	switch c {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, Report(err, true))
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, Report(err, false))
	case 'q':
		fmt.Fprintf(s, "%q", Report(err, true))
	}
}

// Report formats a strings from an error value satisfying a subset of the
// Error interface. A multiline message will be generated if the error has
// nested errors and verbose is true.
func Report(in error, verbose bool) string {
	buf := &strbld.Builder{}
	putStr(buf, directOp(in))
	putStr(buf, directCtx(in))

	// if hasMsg is false, new newline will be added when printing the 'cause'
	hasMsg := any(
		putKind(buf, directKind(in)),
		putStr(buf, directMsg(in)),
	)

	if !verbose {
		return buf.String()
	}

	switch err := in.(type) {
	case withChild:
		putErr(buf, hasMsg, err.Cause())

	case withChildren:
		for _, sub := range err.Causes() {
			putSubErr(buf, sub)
		}
	}

	if buf.Len() == 0 {
		return "unknown error"
	}
	return buf.String()
}

func putStr(b *strbld.Builder, s string) bool {
	if s != "" {
		b.Pad(": ")
		b.WriteString(s)
		return true
	}
	return false
}

func putErr(b *strbld.Builder, nl bool, err error) bool {
	if err == nil {
		return false
	}

	s := fmt.Sprintf("%+v", err)
	if s == "" {
		return false
	}

	if nl {
		b.Pad("\n\t")
	} else {
		b.Pad(": ")
	}
	b.WriteString(s)
	return true
}

func putSubErr(b *strbld.Builder, err error) bool {
	if err == nil {
		return false
	}

	s := fmt.Sprintf("%+v", err)
	if s == "" {
		return false
	}

	b.Pad("\n\t")

	// iterate lines
	r := strings.NewReader(s)
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		if !first {
			b.Pad("\n\t")
		} else {
			first = false
		}

		b.WriteString(scanner.Text())
	}
	return true
}

func putKind(b *strbld.Builder, err error) bool {
	if err != nil {
		return putStr(b, err.Error())
	}
	return false
}

func any(bs ...bool) bool {
	for _, b := range bs {
		if b {
			return true
		}
	}
	return false
}

func directCtx(in error) string {
	if err, ok := in.(withContext); ok {
		return err.Context()
	}
	return ""
}
