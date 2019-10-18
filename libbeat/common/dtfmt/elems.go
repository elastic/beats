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

package dtfmt

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

type element interface {
	requires(c *ctxConfig) error
	estimateSize() int
	compile() (prog, error)
}

type runeLiteral struct {
	r rune
}

type stringLiteral struct {
	s []byte
}

type unpaddedNumber struct {
	ft        fieldType
	maxDigits int
	signed    bool
}

type paddedNumber struct {
	ft                   fieldType
	div                  int
	minDigits, maxDigits int
	signed               bool
}

type textField struct {
	ft    fieldType
	short bool
}

type twoDigitYear struct {
	ft fieldType
}

type paddingZeros struct {
	count int
}

func (runeLiteral) requires(*ctxConfig) error { return nil }
func (runeLiteral) estimateSize() int         { return 1 }

func (stringLiteral) requires(*ctxConfig) error { return nil }
func (s stringLiteral) estimateSize() int       { return len(s.s) }

func (n unpaddedNumber) requires(c *ctxConfig) error {
	return numRequires(c, n.ft)
}

func (n unpaddedNumber) estimateSize() int {
	return numSize(n.maxDigits, n.signed)
}

func (n paddedNumber) requires(c *ctxConfig) error {
	return numRequires(c, n.ft)
}

func (n paddedNumber) estimateSize() int {
	return numSize(n.maxDigits, n.signed)
}

func (n twoDigitYear) requires(c *ctxConfig) error {
	return numRequires(c, n.ft)
}

func (twoDigitYear) estimateSize() int { return 2 }

func numSize(digits int, signed bool) int {
	if signed {
		return digits + 1
	}
	return digits
}

func numRequires(c *ctxConfig, ft fieldType) error {
	switch ft {
	case ftYear, ftMonthOfYear, ftDayOfMonth:
		c.enableDate()

	case ftWeekyear, ftWeekOfWeekyear:
		c.enableISO()

	case ftDayOfYear:
		c.enableYearday()

	case ftDayOfWeek:
		c.enableWeekday()

	case ftHalfdayOfDay,
		ftHourOfHalfday,
		ftClockhourOfHalfday,
		ftClockhourOfDay,
		ftHourOfDay,
		ftMinuteOfDay,
		ftMinuteOfHour,
		ftSecondOfDay,
		ftSecondOfMinute:
		c.enableClock()

	case ftMillisOfDay:
		c.enableClock()
		c.enableMillis()

	case ftMillisOfSecond:
		c.enableMillis()
	}

	return nil
}

func (f textField) requires(c *ctxConfig) error {
	switch f.ft {
	case ftHalfdayOfDay:
		c.enableClock()
	case ftMonthOfYear:
		c.enableDate()
	case ftDayOfWeek:
		c.enableWeekday()
	case ftTimeZoneOffset:
		c.enableTimeZoneOffset()
	default:
		return fmt.Errorf("time field %v not supported by text", f.ft)
	}
	return nil
}

func (f textField) estimateSize() int {
	switch f.ft {
	case ftHalfdayOfDay:
		return 2
	case ftDayOfWeek:
		if f.short {
			return 3
		}
		return 9 // max(weekday) = len(Wednesday)
	case ftMonthOfYear:
		if f.short {
			return 6
		}
		return 9 // max(month) = len(September)
	case ftTimeZoneOffset:
		return 6
	default:
		return 0
	}
}

func (r runeLiteral) compile() (prog, error) {
	switch utf8.RuneLen(r.r) {
	case -1:
		return prog{}, errors.New("invalid rune")
	}

	var tmp [8]byte
	l := utf8.EncodeRune(tmp[:], r.r)
	return makeCopy(tmp[:l])
}

func (s stringLiteral) compile() (prog, error) {
	return makeCopy([]byte(s.s))
}

func (n unpaddedNumber) compile() (prog, error) {
	return makeProg(opNum, byte(n.ft))
}

func (n paddedNumber) compile() (prog, error) {
	if n.div == 0 {
		return makeProg(opNumPadded, byte(n.ft), byte(n.maxDigits))
	}
	return makeProg(opExtNumPadded, byte(n.ft), byte(n.div), byte(n.maxDigits))
}

func (n twoDigitYear) compile() (prog, error) {
	return makeProg(opTwoDigit, byte(n.ft))
}

func (f textField) compile() (prog, error) {
	if f.short {
		return makeProg(opTextShort, byte(f.ft))
	}
	return makeProg(opTextLong, byte(f.ft))
}

func (p paddingZeros) requires(c *ctxConfig) error { return nil }
func (p paddingZeros) estimateSize() int           { return p.count }
func (p paddingZeros) compile() (prog, error) {
	return makeProg(opZeros, byte(p.count))
}
