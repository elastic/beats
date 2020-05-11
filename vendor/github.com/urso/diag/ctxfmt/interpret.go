// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ctxfmt

import (
	"fmt"
	"reflect"
	"strconv"
	"unicode/utf8"
)

type interpreter struct {
	p    *printer
	args argstate
	st   state
	cb   CB

	fmtBuf [128]byte
}

type state struct {
	inError bool
	inPanic bool
	arg     interface{}
	val     reflect.Value
}

type formatterState struct {
	*printer
	tok *formatToken
}

const lHexDigits = "0123456789abcdefx"
const uHexDigits = "0123456789ABCDEFX"

func (in *interpreter) onString(s string) {
	in.p.WriteString(s)
}

func (in *interpreter) onToken(tok formatToken) {
	arg, argIdx, exists := in.args.next()
	if !exists {
		in.formatErr(&tok, exists, arg, errMissingArg)
		return
	}

	if tok.flags.named || isErrorValue(arg) || isFieldValue(arg) {
		in.cb(tok.field, argIdx, arg)
	}

	in.formatArg(&tok, arg)
}

func (in *interpreter) onParseError(tok formatToken, err error) {
	arg, _, has := in.args.next()
	in.formatErr(&tok, has, arg, err)
}

func (in *interpreter) formatErr(tok *formatToken, hasArg bool, arg interface{}, err error) {
	switch err {
	case errInvalidVerb:
		in.p.WriteString("%!")
		in.p.WriteRune(tok.verb)
		in.p.WriteString("(INVALID)")
		if hasArg {
			in.formatErrArg(tok, arg)
		}
	case errNoVerb:
		in.p.WriteString("%!(NOVERB)")
		if hasArg {
			in.formatErrArg(tok, arg)
		}
	case errCloseMissing:
		in.p.WriteString("%!(MISSING })")
	case errNoFieldName:
		in.p.WriteString("%!(NO FIELD)")
	case errMissingArg:
		in.p.WriteString("%!")
		in.p.WriteRune(tok.verb)
		in.p.WriteString("(MISSING)")
	}
}

func (in *interpreter) formatErrArg(tok *formatToken, arg interface{}) {
	if arg == nil {
		in.p.WriteString("(<nil>)")
		return
	}

	in.p.WriteByte('(')
	in.p.WriteString(reflect.TypeOf(arg).String())
	in.p.WriteByte('=')
	tmpTok := *tok
	tmpTok.verb = 'v'
	in.formatArg(&tmpTok, arg)
	in.p.WriteByte(')')
}

func (in *interpreter) formatArg(tok *formatToken, arg interface{}) {
	in.st.arg = arg
	in.st.val = reflect.Value{}
	verb := tok.verb

	if arg == nil {
		switch verb {
		case 'T', 'v':
			in.formatPadString(tok, "<nil>")
		default:
			in.formatBadVerb(tok)
		}
		return
	}

	// Type and pointer are special. Let's treat them first
	switch verb {
	case 'T':
		in.fmtStr(tok, reflect.TypeOf(arg).String())
		return
	case 'p':
		in.fmtPointer(tok, reflect.ValueOf(arg))
		return
	}

	// try to print primitive types without reflection
	switch value := arg.(type) {
	case bool:
		in.fmtBool(tok, value)
	case int:
		in.fmtInt(tok, uint64(value), true)
	case int8:
		in.fmtInt(tok, uint64(value), true)
	case int16:
		in.fmtInt(tok, uint64(value), true)
	case int32:
		in.fmtInt(tok, uint64(value), true)
	case int64:
		in.fmtInt(tok, uint64(value), true)
	case uint:
		in.fmtInt(tok, uint64(value), false)
	case uint8:
		in.fmtInt(tok, uint64(value), false)
	case uint16:
		in.fmtInt(tok, uint64(value), false)
	case uint32:
		in.fmtInt(tok, uint64(value), false)
	case uint64:
		in.fmtInt(tok, value, false)
	case uintptr:
		in.fmtInt(tok, uint64(value), false)
	case string:
		in.fmtString(tok, value)
	case []byte:
		in.fmtBytes(tok, "[]byte", value)
	case float32:
		in.fmtFloat(tok, float64(value), 32)
	case float64:
		in.fmtFloat(tok, value, 64)
	case complex64:
		in.fmtComplex(tok, complex128(value), 64)
	case complex128:
		in.fmtComplex(tok, value, 128)
	case reflect.Value:
		in.fmtValue(tok, value, 0)

	default:
		in.fmtValue(tok, reflect.ValueOf(arg), 0)
	}
}

func (in *interpreter) fmtValue(tok *formatToken, v reflect.Value, depth int) {
	if in.handleMethods(tok, v) {
		return
	}

	in.st.arg = nil
	in.st.val = v

	verb, flags := tok.verb, &tok.flags

	switch v.Kind() {
	case reflect.Invalid:
		switch {
		case depth == 0:
			in.p.WriteString("<invalid reflect.Value>")
		case verb == 'v':
			in.p.WriteString("<nil>")
		default:
			in.formatBadVerb(tok)
		}

	case reflect.Bool:
		in.fmtBool(tok, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		in.fmtInt(tok, uint64(v.Int()), true)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		in.fmtInt(tok, v.Uint(), false)
	case reflect.Float32:
		in.fmtFloat(tok, v.Float(), 32)
	case reflect.Float64:
		in.fmtFloat(tok, v.Float(), 64)
	case reflect.Complex64:
		in.fmtComplex(tok, v.Complex(), 64)
	case reflect.Complex128:
		in.fmtComplex(tok, v.Complex(), 128)
	case reflect.String:
		in.fmtString(tok, v.String())
	case reflect.Ptr:
		if depth == 0 && v.Pointer() != 0 {
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Map, reflect.Struct, reflect.Array, reflect.Slice:
				in.p.WriteByte('&')
				in.fmtValue(tok, elem, depth+1)
				return
			}
		}
		fallthrough
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		in.fmtPointer(tok, v)

	case reflect.Interface:
		switch elem := v.Elem(); {
		case elem.IsValid():
			in.fmtValue(tok, elem, depth+1)
		case flags.sharpV:
			in.p.WriteString(v.Type().String())
			in.p.WriteString("(nil)")
		default:
			in.p.WriteString("<nil>")
		}

	case reflect.Map:
		if flags.sharpV {
			in.p.WriteString(v.Type().String())
			if v.IsNil() {
				in.p.WriteString("(nil)")
				return
			}
			in.p.WriteByte('{')
		} else {
			in.p.WriteString("map[")
		}

		for iter, i := newMapIter(v), 0; iter.Next(); i++ {
			if i > 0 {
				if flags.sharpV {
					in.p.WriteString(", ")
				} else {
					in.p.WriteByte(' ')
				}
			}

			key := iter.Key()
			val := iter.Value()
			in.fmtValue(tok, key, depth+1)
			in.p.WriteByte(':')
			in.fmtValue(tok, val, depth+1)
		}

		if flags.sharpV {
			in.p.WriteByte('}')
		} else {
			in.p.WriteByte(']')
		}

	case reflect.Struct:
		if flags.sharpV {
			in.p.WriteString(v.Type().String())
		}
		in.p.WriteByte('{')
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				if flags.sharpV {
					in.p.WriteString(", ")
				} else {
					in.p.WriteByte(' ')
				}
			}

			if flags.plusV || flags.sharpV {
				if name := v.Type().Field(i).Name; name != "" {
					in.p.WriteString(name)
					in.p.WriteByte(':')
				}
			}

			fld := v.Field(i)
			if fld.Kind() == reflect.Interface && !fld.IsNil() {
				fld = fld.Elem()
			}
			in.fmtValue(tok, fld, depth+1)
		}
		in.p.WriteByte('}')

	case reflect.Array, reflect.Slice:
		// handle variants of []byte
		switch verb {
		case 's', 'q', 'x', 'X':
			t := v.Type()
			if t.Elem().Kind() == reflect.Uint8 {
				var bytes []byte
				if v.Kind() == reflect.Slice {
					bytes = v.Bytes()
				} else if v.CanAddr() {
					bytes = v.Slice(0, v.Len()).Bytes()
				} else {
					// Copy original bytes into tempoary buffer.
					// TODO: can we read the original bytes via reflection/pointers?
					bytes = make([]byte, v.Len())
					for i := range bytes {
						bytes[i] = byte(v.Index(i).Uint())
					}
				}
				in.fmtBytes(tok, t.String(), bytes)
				return
			}
		}

		if flags.sharpV {
			in.p.WriteString(v.Type().String())
			if v.Kind() == reflect.Slice && v.IsNil() {
				in.p.WriteString("(nil)")
				return
			}

			in.p.WriteByte('{')
			for i := 0; i < v.Len(); i++ {
				if i > 0 {
					in.p.WriteString(", ")
				}
				in.fmtValue(tok, v.Index(i), depth+1)
			}
			in.p.WriteByte('}')
			return
		}

		in.p.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				in.p.WriteByte(' ')
			}
			in.fmtValue(tok, v.Index(i), depth+1)
		}
		in.p.WriteByte(']')
	}
}

func (in *interpreter) handleMethods(tok *formatToken, v reflect.Value) bool {
	flags := &tok.flags

	if in.st.inError || !v.IsValid() || !v.CanInterface() {
		return false
	}

	arg := v.Interface()
	if formatter, ok := arg.(fmt.Formatter); ok {
		defer in.recoverPanic(tok, arg)
		formatter.Format(&formatterState{in.p, tok}, rune(tok.verb))
		return true
	}

	if flags.sharpV {
		stringer, ok := arg.(fmt.GoStringer)
		if ok {
			defer in.recoverPanic(tok, arg)
			in.fmtStr(tok, stringer.GoString())
		}
		return ok
	}

	switch tok.verb {
	case 'v', 's', 'x', 'X', 'q':
		break
	default:
		return false
	}

	switch v := arg.(type) {
	case error:
		defer in.recoverPanic(tok, arg)
		in.fmtString(tok, v.Error())
	case fmt.Stringer:
		defer in.recoverPanic(tok, arg)
		in.fmtString(tok, v.String())
	default:
		return false
	}
	return true
}

func (in *interpreter) recoverPanic(tok *formatToken, arg interface{}) {
	p := in.p

	if err := recover(); err != nil {
		if v := reflect.ValueOf(arg); v.Kind() == reflect.Ptr && v.IsNil() {
			p.WriteString("<nil>")
			return
		}
		if in.st.inPanic {
			// nested recursive panic.
			panic(err)
		}

		errTok := *tok
		errTok.verb = 'v'
		errTok.flags = flags{}

		p.WriteString("%!")
		p.WriteRune(tok.verb)
		p.WriteString("(PANIC=")
		in.st.inPanic = true
		in.formatArg(&errTok, err)
		in.st.inPanic = false
		p.WriteByte(')')
	}
}

func (in *interpreter) fmtBool(tok *formatToken, b bool) {
	switch tok.verb {
	case 't', 'v':
		if b {
			in.formatPadString(tok, "true")
		} else {
			in.formatPadString(tok, "false")
		}
	default:
		in.formatBadVerb(tok)
	}
}

func (in *interpreter) fmtInt(tok *formatToken, v uint64, signed bool) {
	flags := &tok.flags
	verb := tok.verb

	switch verb {
	case 'v':
		if flags.sharpV && !signed {
			in.fmtHex64(tok, v, true)
		} else {
			in.fmtIntBase(tok, v, 10, signed, lHexDigits)
		}
	case 'd':
		in.fmtIntBase(tok, v, 10, signed, lHexDigits)
	case 'b':
		in.fmtIntBase(tok, v, 2, signed, lHexDigits)
	case 'o', 'O':
		in.fmtIntBase(tok, v, 8, signed, lHexDigits)
	case 'x':
		in.fmtIntBase(tok, v, 16, signed, lHexDigits)
	case 'X':
		in.fmtIntBase(tok, v, 16, signed, uHexDigits)
	case 'c':
		in.fmtRune(tok, v)
	case 'q':
		if v <= utf8.MaxRune {
			in.fmtRuneQ(tok, v)
		} else {
			in.formatBadVerb(tok)
		}
	case 'U':
		in.fmtUnicode(tok, v)
	default:
		in.formatBadVerb(tok)
	}
}

func (in *interpreter) fmtRune(tok *formatToken, v uint64) {
	n := utf8.EncodeRune(in.fmtBuf[:], convRune(v))
	in.formatPad(tok, in.fmtBuf[:n])
}

func (in *interpreter) fmtRuneQ(tok *formatToken, v uint64) {
	r := convRune(v)
	buf := in.fmtBuf[:0]
	if tok.flags.plus {
		buf = strconv.AppendQuoteRuneToASCII(buf, r)
	} else {
		buf = strconv.AppendQuoteRune(buf, r)
	}
	in.formatPad(tok, buf)
}

func (in *interpreter) fmtUnicode(tok *formatToken, u uint64) {
	precision := 4

	// prepare temporary format buffer
	buf := in.fmtBuf[0:]
	if tok.flags.hasPrecision && tok.precision > precision {
		precision = tok.precision
		width := 2 + precision + 2 + utf8.UTFMax + 1
		if width > len(buf) {
			buf = make([]byte, width)
		}
	}

	// format from right to left
	i := len(buf)

	// print rune with quote if '#' is set and rune is quoted
	if tok.flags.sharp && u < utf8.MaxRune && strconv.IsPrint(rune(u)) {
		i--
		buf[i] = '\''

		i -= utf8.RuneLen(rune(u))
		utf8.EncodeRune(buf[i:], rune(u))

		i -= 2
		copy(buf[i:], " '")
	}

	// format hex digits
	for u >= 16 {
		i--
		buf[i] = uHexDigits[u&0x0F]
		precision--
		u >>= 4
	}
	i--
	buf[i] = uHexDigits[u]
	precision--

	// left-pad zeroes
	for precision > 0 {
		i--
		buf[i] = '0'
		precision--
	}

	// leading 'U+'
	i -= 2
	copy(buf[i:], "U+")

	// ensure we pad with ' ' always:
	in.formatPadWidth(tok, buf[i:], ' ')
}

func (in *interpreter) fmtIntBase(tok *formatToken, v uint64, base int, signed bool, digits string) {
	flags := &tok.flags

	neg := signed && int64(v) < 0
	if neg {
		v = -v
	}

	buf := in.fmtBuf[:]
	if flags.hasWidth || flags.hasPrecision {
		width := 3 + tok.width + tok.precision
		if width > len(buf) {
			buf = make([]byte, width)
		}
	}

	precision := 0
	if flags.hasPrecision {
		precision = tok.precision
		if precision == 0 && v == 0 {
			in.formatPaddingWith(tok.width, ' ')
			return
		}
	} else if flags.zero && flags.hasWidth {
		precision = tok.width
		if neg || flags.plus || flags.space {
			precision-- // reserve space for '-' sign
		}
	}

	// print right-to-left
	i := len(buf)
	switch base {
	case 10:
		for v >= 10 {
			next := v / 10
			i--
			buf[i] = byte('0' + v - next*10)
			v = next
		}
	case 16:
		for ; v >= 16; v >>= 4 {
			i--
			buf[i] = digits[v&0xF]
		}
	case 8:
		for ; v >= 8; v >>= 3 {
			i--
			buf[i] = byte('0' + v&7)
		}
	case 2:
		for ; v >= 2; v >>= 1 {
			i--
			buf[i] = byte('0' + v&1)
		}
	default:
		panic("unknown base")
	}
	i--
	buf[i] = digits[v]

	// left-pad zeros
	for i > 0 && precision > len(buf)-i {
		i--
		buf[i] = '0'
	}

	// '#' triggers prefix
	if flags.sharp {
		switch base {
		case 2:
			i--
			buf[i] = 'b'
			i--
			buf[i] = '0'
		case 8:
			if buf[i] != '0' {
				i--
				buf[i] = '0'
			}
		case 16:
			i--
			buf[i] = digits[16]
			i--
			buf[i] = '0'
		}
	}

	if tok.verb == 'O' {
		i--
		buf[i] = 'o'
		i--
		buf[i] = '0'
	}

	// add sign
	if neg {
		i--
		buf[i] = '-'
	} else if flags.plus {
		i--
		buf[i] = '+'
	} else if flags.space {
		i--
		buf[i] = ' '
	}

	in.formatPadWidth(tok, buf[i:], ' ')
}

func (in *interpreter) fmtString(tok *formatToken, s string) {
	switch tok.verb {
	case 'v':
		if tok.flags.sharpV {
			in.fmtQualified(tok, s)
		} else {
			in.fmtStr(tok, s)
		}
	case 's':
		in.fmtStr(tok, s)
	case 'x':
		in.fmtStrHex(tok, s, lHexDigits)
	case 'X':
		in.fmtStrHex(tok, s, uHexDigits)
	case 'q':
		in.fmtQualified(tok, s)
	default:
		in.formatBadVerb(tok)
	}
}

func (in *interpreter) fmtHex64(tok *formatToken, v uint64, leading0x bool) {
	tokFmt := *tok
	tokFmt.flags.sharp = leading0x
	in.fmtIntBase(&tokFmt, v, 16, false, lHexDigits)
}

func (in *interpreter) fmtBytes(tok *formatToken, typeName string, b []byte) {
	flags := &tok.flags

	switch tok.verb {
	case 'v', 'd':
		if flags.sharpV { // print hex dump of '#' is set
			in.p.WriteString(typeName)
			if b == nil {
				in.p.WriteString("(nil)")
				return
			}

			in.p.WriteByte('{')
			for i, c := range b {
				if i > 0 {
					in.p.WriteString(", ")
				}
				in.fmtHex64(tok, uint64(c), true)
			}
			in.p.WriteByte('}')
		} else { // print base-10 digits if '#' is not set
			in.p.WriteByte('[')
			for i, c := range b {
				if i > 0 {
					in.p.WriteByte(' ')
				}
				in.fmtIntBase(tok, uint64(c), 10, false, lHexDigits)
			}
			in.p.WriteByte(']')
		}
	case 's':
		in.fmtStr(tok, unsafeString(b))
	case 'x':
		in.fmtBytesHex(tok, b, lHexDigits)
	case 'X':
		in.fmtBytesHex(tok, b, uHexDigits)
	case 'q':
		in.fmtQualified(tok, unsafeString(b))
	default:
		in.fmtValue(tok, reflect.ValueOf(b), 0)
	}
}

func (in *interpreter) fmtStrHex(tok *formatToken, s string, digits string) {
	in.fmtSBHex(tok, s, nil, digits)
}

func (in *interpreter) fmtBytesHex(tok *formatToken, b []byte, digits string) {
	in.fmtSBHex(tok, "", b, digits)
}

func (in *interpreter) fmtSBHex(tok *formatToken, s string, b []byte, digits string) {
	flags := &tok.flags

	N := len(b)
	if b == nil {
		N = len(s)
	}

	if tok.flags.hasPrecision && tok.precision < N {
		N = tok.precision
	}

	// Compute total width of hex encoded string. Each byte requires 2 symbols.
	// Codes are separates by a space character if the 'space' flag is set.
	// Leading 0x will be added if the '#' modifier has been used.
	width := 2 * N
	if width <= 0 {
		if flags.hasWidth {
			in.formatPadding(tok, tok.width)
		}
		return
	}
	if flags.space {
		if flags.sharp {
			width *= 2
		}
		width += N - 1
	} else if flags.sharp {
		width += 2
	}
	needsPadding := tok.width > width && flags.hasWidth

	// handle left padding if '-' modifier is not set.
	if needsPadding && !flags.minus {
		in.formatPadding(tok, tok.width-width)
	}

	buf := in.fmtBuf[:]
	if width >= len(buf) {
		buf = make([]byte, width)
	} else {
		buf = buf[:width]
	}
	pos := 0

	// write hex string
	if flags.sharp {
		buf[pos], buf[pos+1] = '0', digits[16]
		pos += 2
	}
	for i := 0; i < N; i++ {
		if flags.space && i > 0 {
			buf[pos] = ' '
			pos++
			if flags.sharp {
				buf[pos], buf[pos+1] = '0', digits[16]
				pos += 2
			}
		}

		var c byte
		if b != nil {
			c = b[i]
		} else {
			c = s[i]
		}

		buf[pos], buf[pos+1] = digits[c>>4], digits[c&0xf]
		pos += 2
	}
	in.p.Write(buf)

	// handle right padding if '-' modifier is set.
	if needsPadding && flags.minus {
		in.formatPadding(tok, tok.width-width)
	}
}

func (in *interpreter) fmtQualified(tok *formatToken, s string) {
	flags := &tok.flags
	s = in.truncate(tok, s)

	if flags.sharp && strconv.CanBackquote(s) {
		in.formatPadString(tok, "`"+s+"`")
		return
	}

	buf := in.fmtBuf[:0]
	if flags.plus {
		buf = strconv.AppendQuoteToASCII(buf, s)
	} else {
		buf = strconv.AppendQuote(buf, s)
	}

	in.formatPad(tok, buf)
}

func (in *interpreter) fmtFloat(tok *formatToken, f float64, sz int) {
	verb := tok.verb
	switch verb {
	case 'v':
		in.fmtFloatBase(tok, f, sz, 'g', -1)
	case 'b', 'g', 'G', 'x', 'X':
		in.fmtFloatBase(tok, f, sz, verb, -1)
	case 'f', 'e', 'E':
		in.fmtFloatBase(tok, f, sz, verb, 6)
	case 'F':
		in.fmtFloatBase(tok, f, sz, 'f', 6)
	default:
		in.formatBadVerb(tok)
	}
}

func (in *interpreter) fmtFloatBase(tok *formatToken, f float64, sz int, verb rune, precision int) {
	flags := &tok.flags
	if flags.hasPrecision {
		precision = tok.precision
	}

	// format number + ensure sign is always present
	buf := strconv.AppendFloat(in.fmtBuf[:1], f, byte(verb), precision, sz)
	if buf[1] == '-' || buf[1] == '+' {
		buf = buf[1:]
	} else {
		buf[0] = '+'
	}

	// make '+' sign optional
	if buf[0] == '+' && flags.space && !flags.plus {
		buf[0] = ' '
	}

	// Ensure Inf and NaN values are not padded with '0'
	if buf[1] == 'I' || buf[1] == 'N' {
		if buf[1] == 'N' && !flags.space && !flags.plus {
			buf = buf[1:]
		}
		in.formatPadWidth(tok, buf, ' ')
		return
	}

	// requred post-processing if '#' is set.
	// -> print decimal point and retain/restore trailing zeros
	if flags.sharp && verb != 'b' {
		digits := 0
		switch verb {
		case 'v', 'g', 'G':
			digits = precision
			if digits < 0 {
				digits = 6
			}
		}

		// expBuf holds the exponent, so we can overwrite the current
		// buffer with the decimal
		var expBuf [5]byte
		exp := expBuf[:0]

		hasDecimal := false
		for i := 1; i < len(buf); i++ {
			switch buf[i] {
			case '.':
				hasDecimal = true
			case 'p', 'P':
				exp = append(exp, buf[i:]...)
				buf = buf[:i]
			case 'e', 'E':
				if verb != 'x' && verb != 'X' {
					exp = append(exp, buf[i:]...)
					buf = buf[:i]
					break
				}
				fallthrough
			default:
				digits--
			}
		}
		if !hasDecimal {
			buf = append(buf, '.')
		}
		for digits > 0 {
			buf = append(buf, '0')
			digits--
		}
		buf = append(buf, exp...)
	}

	// print number with sign
	if flags.plus || buf[0] != '+' {
		if flags.zero && flags.hasWidth && tok.width > len(buf) {
			in.p.WriteByte(buf[0])
			in.formatPadding(tok, tok.width-len(buf))
			in.p.Write(buf[1:])
			return
		}

		in.formatPad(tok, buf)
		return
	}

	// print positive number without sign
	in.formatPad(tok, buf[1:])
}

func (in *interpreter) fmtComplex(tok *formatToken, v complex128, sz int) {
	switch tok.verb {
	case 'v', 'b', 'g', 'G', 'x', 'X', 'f', 'F', 'e', 'E':
		break
	default:
		in.formatBadVerb(tok)
	}

	in.p.WriteByte('(')
	in.fmtFloat(tok, real(v), sz/2)

	iTok := *tok
	iTok.flags.plus = true
	in.fmtFloat(&iTok, imag(v), sz/2)
	in.p.WriteString("i)")
}

// truncate returns the number of configured unicode symbols.
func (in *interpreter) truncate(tok *formatToken, s string) string {
	if tok.flags.hasPrecision {
		n := tok.precision
		for i := range s { // handle utf-8 with help of range loop
			n--
			if n < 0 {
				return s[:i]
			}
		}
	}
	return s
}

func (in *interpreter) fmtStr(tok *formatToken, s string) {
	in.formatPadString(tok, in.truncate(tok, s))
}

func (in *interpreter) fmtPointer(tok *formatToken, value reflect.Value) {
	verb := tok.verb

	var u uintptr
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		u = value.Pointer()
	default:
		in.formatBadVerb(tok)
		return
	}

	switch verb {
	case 'v':
		if tok.flags.sharpV {
			in.p.WriteByte('(')
			in.p.WriteString(value.Type().String())
			in.p.WriteString(")(")
			if u == 0 {
				in.p.WriteString("nil")
			} else {
				in.fmtHex64(tok, uint64(u), true)
			}
			in.p.WriteByte(')')
		} else if u != 0 {
			in.fmtHex64(tok, uint64(u), !tok.flags.sharp)
		} else {
			in.formatPadString(tok, "<nil>")
		}
	case 'p':
		in.fmtHex64(tok, uint64(u), !tok.flags.sharp)
	case 'b', 'o', 'd', 'x', 'X':
		in.fmtInt(tok, uint64(u), false)
	default:
		in.formatBadVerb(tok)
	}
}

func (in *interpreter) formatBadVerb(tok *formatToken) {
	st := &in.st
	st.inError = true

	in.p.WriteString("%!")
	in.p.WriteRune(tok.verb)
	in.p.WriteByte('(')

	switch {
	case st.arg != nil:
		in.p.WriteString(reflect.TypeOf(st.arg).String())
		in.p.WriteByte('=')
		tmpTok := *tok
		tmpTok.verb = 'v'
		in.formatArg(&tmpTok, st.arg)
	case st.val.IsValid():
		in.p.WriteString(st.val.Type().String())
		in.p.WriteByte('=')
		tmpTok := *tok
		tmpTok.verb = 'v'
		in.fmtValue(&tmpTok, st.val, 0)
	default:
		in.p.WriteString("<nil>")
	}

	in.p.WriteByte(')')
	st.inError = false
}

func (in *interpreter) formatPad(tok *formatToken, b []byte) {
	if !tok.flags.hasWidth || tok.width == 0 {
		in.p.Write(b)
		return
	}

	width := tok.width - utf8.RuneCount(b)
	if !tok.flags.minus {
		in.formatPadding(tok, width)
		in.p.Write(b)
	} else {
		in.p.Write(b)
		in.formatPadding(tok, width)
	}
}

func (in *interpreter) formatPadString(tok *formatToken, s string) {
	if !tok.flags.hasWidth || tok.width == 0 {
		in.p.WriteString(s)
		return
	}

	width := tok.width - utf8.RuneCountInString(s)
	if !tok.flags.minus {
		in.formatPadding(tok, width)
		in.p.WriteString(s)
	} else {
		in.p.WriteString(s)
		in.formatPadding(tok, width)
	}
}

func (in *interpreter) fmtPaddingZeros(n int) {
	in.formatPaddingWith(n, '0')
}

func (in *interpreter) formatPadWidth(tok *formatToken, b []byte, padByte byte) {
	if !tok.flags.hasWidth || tok.width == 0 {
		in.p.Write(b)
		return
	}

	width := tok.width - utf8.RuneCount(b)
	if !tok.flags.minus {
		in.formatPaddingWith(width, padByte)
		in.p.Write(b)
	} else {
		in.p.Write(b)
		in.formatPaddingWith(width, padByte)
	}
}

func (in *interpreter) formatPadding(tok *formatToken, n int) {
	if tok.flags.zero {
		in.formatPaddingWith(n, '0')
	} else {
		in.formatPaddingWith(n, ' ')
	}
}

func (in *interpreter) formatPaddingWith(n int, padByte byte) {
	if n <= 0 {
		return
	}

	var padBuf [128]byte
	for n > 0 {
		buf := padBuf[:]
		if n < len(padBuf) {
			buf = buf[:n]
		}
		for i := range buf {
			buf[i] = padByte
		}

		in.p.Write(buf)
		n -= len(buf)
	}
}

// Width returns the value of the width option and whether it has been set.
func (f *formatterState) Width() (wid int, ok bool) {
	return f.tok.width, f.tok.flags.hasWidth
}

// Precision returns the value of the precision option and whether it has been set.
func (f *formatterState) Precision() (prec int, ok bool) {
	return f.tok.precision, f.tok.flags.hasPrecision
}

// Flag reports whether the flag c, a character, has been set.
func (f *formatterState) Flag(c int) bool {
	flags := &f.tok.flags
	switch c {
	case '-':
		return flags.minus
	case '+':
		return flags.plus || flags.plusV
	case '#':
		return flags.sharp || flags.sharpV
	case ' ':
		return flags.space
	case '0':
		return flags.zero
	default:
		return false
	}
}
