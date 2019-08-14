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
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Formatter will format time values into strings, based on pattern used to
// create the Formatter.
type Formatter struct {
	prog   prog
	sz     int
	config ctxConfig
}

var ctxPool = &sync.Pool{
	New: func() interface{} { return &ctx{} },
}

func newCtx() *ctx {
	return ctxPool.Get().(*ctx)
}

func newCtxWithSize(sz int) *ctx {
	ctx := newCtx()
	if ctx.buf == nil || cap(ctx.buf) < sz {
		ctx.buf = make([]byte, 0, sz)
	}
	return ctx
}

func releaseCtx(c *ctx) {
	ctxPool.Put(c)
}

// NewFormatter creates a new time formatter based on provided pattern.
// If pattern is invalid an error is returned.
func NewFormatter(pattern string) (*Formatter, error) {
	b := newBuilder()

	err := parsePatternTo(b, pattern)
	if err != nil {
		return nil, err
	}

	b.optimize()

	cfg, err := b.createConfig()
	if err != nil {
		return nil, err
	}

	prog, err := b.compile()
	if err != nil {
		return nil, err
	}

	sz := b.estimateSize()
	f := &Formatter{
		prog:   prog,
		sz:     sz,
		config: cfg,
	}
	return f, nil
}

// EstimateSize estimates the required buffer size required to hold
// the formatted time string. Estimated size gives no exact guarantees.
// Estimated size might still be too low or too big.
func (f *Formatter) EstimateSize() int {
	return f.sz
}

func (f *Formatter) appendTo(ctx *ctx, b []byte, t time.Time) ([]byte, error) {
	ctx.initTime(&f.config, t)
	return f.prog.eval(b, ctx, t)
}

// AppendTo appends the formatted time value to the given byte buffer.
func (f *Formatter) AppendTo(b []byte, t time.Time) ([]byte, error) {
	ctx := newCtx()
	defer releaseCtx(ctx)
	return f.appendTo(ctx, b, t)
}

// Write writes the formatted time value to the given writer. Returns
// number of bytes written or error if formatter or writer fails.
func (f *Formatter) Write(w io.Writer, t time.Time) (int, error) {
	var err error

	ctx := newCtxWithSize(f.sz)
	defer releaseCtx(ctx)

	ctx.buf, err = f.appendTo(ctx, ctx.buf[:0], t)
	if err != nil {
		return 0, err
	}
	return w.Write(ctx.buf)
}

// Format formats the given time value into a new string.
func (f *Formatter) Format(t time.Time) (string, error) {
	var err error

	ctx := newCtxWithSize(f.sz)
	defer releaseCtx(ctx)

	ctx.buf, err = f.appendTo(ctx, ctx.buf[:0], t)
	if err != nil {
		return "", err
	}
	return string(ctx.buf), nil
}

func parsePatternTo(b *builder, pattern string) error {
	for i := 0; i < len(pattern); {
		tok, tokText, err := parseToken(pattern, &i)
		if err != nil {
			return err
		}

		tokLen := len(tokText)
		switch tok {
		case 'x': // weekyear (year)
			if tokLen == 2 {
				b.twoDigitWeekYear()
			} else {
				b.weekyear(tokLen, 4)
			}

		case 'y', 'Y': // year and year of era (year) == 'y'
			if tokLen == 2 {
				b.twoDigitYear()
			} else {
				b.year(tokLen, 4)
			}

		case 'w': // week of weekyear (num)
			b.weekOfWeekyear(tokLen)

		case 'e': // day of week (num)
			b.dayOfWeek(tokLen)

		case 'E': // day of week (text)
			if tokLen >= 4 {
				b.dayOfWeekText()
			} else {
				b.dayOfWeekShortText()
			}

		case 'D': // day of year (number)
			b.dayOfYear(tokLen)

		case 'M': // month of year (month)
			if tokLen >= 3 {
				if tokLen >= 4 {
					b.monthOfYearText()
				} else {
					b.monthOfYearShortText()
				}
			} else {
				b.monthOfYear(tokLen)
			}

		case 'd': //day of month (number)
			b.dayOfMonth(tokLen)

		case 'a': // half of day (text) 'AM/PM'
			b.halfdayOfDayText()

		case 'K': // hour of half day (number) (0 - 11)
			b.hourOfHalfday(tokLen)

		case 'h': // clock hour of half day (number) (1 - 12)
			b.clockhourOfHalfday(tokLen)

		case 'H': // hour of day (number) (0 - 23)
			b.hourOfDay(tokLen)

		case 'k': // clock hour of half day (number) (1 - 24)
			b.clockhourOfDay(tokLen)

		case 'm': // minute of hour
			b.minuteOfHour(tokLen)

		case 's': // second of minute
			b.secondOfMinute(tokLen)

		case 'S': // fraction of second
			b.millisOfSecond(tokLen)

		case '\'': // literal
			if tokLen == 1 {
				b.appendRune(rune(tokText[0]))
			} else {
				b.appendLiteral(tokText)
			}

		default:
			return fmt.Errorf("unsupport format '%c'", tok)

		}
	}

	return nil
}

func parseToken(pattern string, i *int) (rune, string, error) {
	start := *i
	idx := start
	length := len(pattern)

	r, w := utf8.DecodeRuneInString(pattern[idx:])
	idx += w
	if ('A' <= r && r <= 'Z') || ('a' <= r && r <= 'z') {
		// Scan a run of the same character, which indicates a time pattern.

		for idx < length {
			peek, w := utf8.DecodeRuneInString(pattern[idx:])
			if peek != r {
				break
			}

			idx += w
		}

		*i = idx
		return r, pattern[start:idx], nil
	}

	if r != '\'' { // single character, no escaped string
		*i = idx
		return '\'', pattern[start:idx], nil
	}

	start = idx // skip ' character
	iEnd := strings.IndexRune(pattern[start:], '\'')
	if iEnd < 0 {
		return r, "", errors.New("missing closing '")
	}

	if iEnd == 0 {
		// escape single quote literal
		*i = idx + 1
		return r, pattern[start : idx+1], nil
	}

	iEnd += start

	*i = iEnd + 1 // point after '
	if len(pattern) > iEnd+1 && pattern[iEnd+1] == '\'' {
		return r, pattern[start : iEnd+1], nil
	}

	return r, pattern[start:iEnd], nil
}
