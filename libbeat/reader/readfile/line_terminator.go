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

package readfile

import "fmt"

// LineTerminator is the option storing the line terminator characters
// Supported newline reference: https://en.wikipedia.org/wiki/Newline#Unicode
type LineTerminator uint8

const (
	// InvalidTerminator is the invalid terminator
	InvalidTerminator LineTerminator = iota
	// AutoLineTerminator accepts both LF and CR+LF
	AutoLineTerminator
	// LineFeed is the unicode char LF
	LineFeed
	// VerticalTab is the unicode char VT
	VerticalTab
	// FormFeed is the unicode char FF
	FormFeed
	// CarriageReturn is the unicode char CR
	CarriageReturn
	// CarriageReturnLineFeed is the unicode chars CR+LF
	CarriageReturnLineFeed
	// NextLine is the unicode char NEL
	NextLine
	// LineSeparator is the unicode char LS
	LineSeparator
	// ParagraphSeparator is the unicode char PS
	ParagraphSeparator
)

var (
	lineTerminators = map[string]LineTerminator{
		"auto":                      AutoLineTerminator,
		"line_feed":                 LineFeed,
		"vertical_tab":              VerticalTab,
		"form_feed":                 FormFeed,
		"carriage_return":           CarriageReturn,
		"carriage_return_line_feed": CarriageReturnLineFeed,
		"next_line":                 NextLine,
		"line_separator":            LineSeparator,
		"paragraph_separator":       ParagraphSeparator,
	}

	lineTerminatorCharacters = map[LineTerminator][]byte{
		AutoLineTerminator:     []byte{'\u000A'},
		LineFeed:               []byte{'\u000A'},
		VerticalTab:            []byte{'\u000B'},
		FormFeed:               []byte{'\u000C'},
		CarriageReturn:         []byte{'\u000D'},
		CarriageReturnLineFeed: []byte("\u000D\u000A"),
		NextLine:               []byte{'\u0085'},
		LineSeparator:          []byte("\u2028"),
		ParagraphSeparator:     []byte("\u2029"),
	}
)

// Unpack unpacks the configuration from the config file
func (l *LineTerminator) Unpack(option string) error {
	terminator, ok := lineTerminators[option]
	if !ok {
		return fmt.Errorf("invalid line terminator: %s", option)
	}

	*l = terminator

	return nil
}
