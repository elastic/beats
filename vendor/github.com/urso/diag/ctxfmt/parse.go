// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package ctxfmt

import "unicode/utf8"

type parser struct {
	handler tokenHandler
}

type tokenHandler interface {
	onString(s string)
	onToken(tok formatToken)
	onParseError(formatToken, error)
}

type formatToken struct {
	field     string
	verb      rune
	width     int
	precision int
	flags     flags
}

type flags struct {
	named        bool
	hasWidth     bool
	hasPrecision bool
	plus         bool
	plusV        bool
	minus        bool
	sharp        bool
	sharpV       bool
	space        bool
	zero         bool
}

var validVerbs [256]bool

func init() {
	for _, v := range "vtTbcdoOqxXUeEfFgGsqxXp" {
		validVerbs[v] = true
	}
}

func (p *parser) parse(msg string) {
	var i int
	end := len(msg)
	for i < end {
		var (
			lasti = i
			err   error
		)

		i = findFmt(msg, i, end)
		if i >= end {
			i = lasti
			break
		}
		if i+1 == end {
			if i > lasti {
				p.handler.onString(msg[lasti:i])
			}
			p.handler.onParseError(formatToken{}, errNoVerb)
			return
		}

		// found escaped '%'. Report string printing '%' and ignore current '%'
		if msg[i+1] == '%' {
			p.handler.onString(msg[lasti : i+1])
			i += 2
			continue
		}

		if i > lasti {
			p.handler.onString(msg[lasti:i])
		}

		var tok formatToken
		i, tok, err = parseFmt(msg, i, end)
		if err != nil {
			p.handler.onParseError(tok, err)
		} else if tok.verb > utf8.RuneSelf || !validVerbs[tok.verb] {
			p.handler.onParseError(tok, errInvalidVerb)
		} else {
			if tok.verb == 'v' {
				tok.flags.sharpV = tok.flags.sharp
				tok.flags.plusV = tok.flags.plus
				tok.flags.sharp = false
				tok.flags.plus = false
			}
			p.handler.onToken(tok)
		}
	}

	if i < end {
		p.handler.onString(msg[i:])
	}
}

func findFmt(in string, start, end int) (i int) {
	for i = start; i < end && in[i] != '%'; {
		i++
	}
	return i
}

func parseFmt(msg string, start, end int) (i int, tok formatToken, err error) {
	i = start + 1
	if i < end && msg[i] == '{' {
		return parseField(msg, start, end)
	}

	i, err = parseFmtSpec(&tok, msg, i, end)
	return i, tok, err
}

func parseFmtSpec(tok *formatToken, msg string, start, end int) (int, error) {
	i := start

	// parse flags
	for i < end {
		newi, isflag := parseFlag(&tok.flags, msg, i)
		if !isflag {
			break
		}
		i = newi
	}

	// fast path for common case of simple lower case verbs without width or
	// precision.
	if c := msg[i]; 'a' <= c && c <= 'z' {
		tok.verb = rune(c)
		return i + 1, nil
	}

	// try to parse width
	num, isnum, newi := parseNum(msg, i, end)
	if isnum {
		if !tok.flags.hasWidth {
			tok.width = num
			tok.flags.hasWidth = true
		}
		i = newi
	}

	// try to parse precision
	if i < end && msg[i] == '.' {
		i++
		num, isnum, newi := parseNum(msg, i, end)
		if isnum {
			if !tok.flags.hasPrecision {
				tok.precision = num
				tok.flags.hasPrecision = true
			}
			i = newi
		} else if !tok.flags.hasPrecision {
			tok.precision = 0
			tok.flags.hasPrecision = true
		}
	}

	if i >= end {
		return i, errNoVerb
	}

	// parse verb
	verb := rune(msg[i])
	if verb >= utf8.RuneSelf {
		verb, size := utf8.DecodeRuneInString(msg[i:])
		tok.verb = verb
		i += size
		return i + size, errInvalidVerb
	}
	tok.verb = verb

	return i + 1, nil
}

// parseField parses a named field format specifier into st.
// The syntax of a field formatter is '%{[+#@]<name>[:<format>]}'.
//
// The prefix '+', '#', '@' modify the printing if no format is configured.
// In this case the 'v' verb is assumed. The '@' flag is synonymous to '#'.
//
// The 'format' section can be any valid format specification
func parseField(msg string, start, end int) (i int, tok formatToken, err error) {
	tok.flags.named = true
	tok.verb = 'v' // default verb for fields is 'v'

	i = start + 2 // start is at '%'
	if i >= end {
		return end, tok, errCloseMissing
	}

	switch msg[i] {
	case '+':
		tok.flags.plus = true
		i++
	case '#', '@':
		tok.flags.sharp = true
		i++
	}

	pos := i
	for i < end && msg[i] != '}' && msg[i] != ':' {
		i++
	}

	if pos == i {
		return i, tok, errNoFieldName
	}
	tok.field = msg[pos:i]

	if i >= end {
		return i, tok, errCloseMissing
	}

	if msg[i] == '}' {
		return i + 1, tok, nil
	}

	// msg[i] == ':' => parse format specification
	i, err = parseFmtSpec(&tok, msg, i+1, end)
	if err != nil {
		return i, tok, nil
	}

	// skip to end of formatter:
	for i < end && msg[i] != '}' {
		i++
	}
	if i >= end {
		return end, tok, errCloseMissing
	}
	return i + 1, tok, nil
}

func parseFlag(flags *flags, msg string, pos int) (int, bool) {
	switch msg[pos] {
	case '#':
		flags.sharp = true
		return pos + 1, true
	case '+':
		flags.plus = true
		return pos + 1, true
	case '-':
		flags.minus = true
		flags.zero = false
		return pos + 1, true
	case '0':
		flags.zero = !flags.minus
		return pos + 1, true
	case ' ':
		flags.space = true
		return pos + 1, true
	}

	return 0, false
}

func parseNum(msg string, start, end int) (num int, isnum bool, i int) {
	for i = start; i < end && '0' <= msg[i] && msg[i] <= '9'; i++ {
		if tooLarge(num) {
			return 0, false, end
		}
		num = 10*num + int(msg[i]-'0')
	}
	return num, i > start, i
}

func tooLarge(i int) bool {
	const max int = 1e6
	return !(-max <= i && i <= max)
}
