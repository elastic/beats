package fld

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"unicode/utf8"
)

type CB func(key string, idx int, val interface{})

type printer struct {
	buf  buffer
	buf0 [128]byte

	state state

	fmtBuf [128]byte // temporary buffer for formatting single values
}

type state struct {
	inError   bool
	inPanic   bool
	arg       interface{}
	value     reflect.Value
	field     string
	verb      byte
	width     int
	precision int
	flags     flags
}

type argstate struct {
	idx  int
	args []interface{}
}

type flags struct {
	named        bool
	hasWidth     bool
	hasPrecision bool
	plus         bool
	minus        bool
	sharp        bool
	space        bool
	zero         bool
}

type fmtFail struct {
	invalidVerb bool
	noVerb      bool
	notClosed   bool
}

type buffer []byte

var (
	errInvalidVerb  = errors.New("invalid verb")
	errNoVerb       = errors.New("no verb")
	errCloseMissing = errors.New("missing '}'")
	errNoFieldName  = errors.New("field name missing")
	errMissingArg   = errors.New("missing arg")
)

var printerPool = sync.Pool{
	New: func() interface{} {
		return &printer{}
	},
}

var validVerbs [256]bool

const lHexDigits = "0123456789abcdefx"
const uHexDigits = "0123456789ABCDEFX"

func init() {
	for _, v := range "vtTbcdoqxXUeEfFgGsqxXp" {
		validVerbs[v] = true
	}
}

func Format(cb CB, msg string, vs ...interface{}) (str string, rest []interface{}) {
	p := newPrinter(cb)
	defer p.release()

	str, used := p.printf(cb, msg, vs)
	if used >= len(vs) {
		return str, nil
	}

	// collect errors from extra variables
	rest = vs[used:]
	for i := range rest {
		if _, ok := rest[i].(error); ok {
			cb("", used+i, rest[i])
		}
	}

	return str, rest
}

func newPrinter(cb CB) *printer {
	p := printerPool.Get().(*printer)
	p.init(cb)
	return p
}

func (p *printer) init(cb CB) {
	if p.buf == nil {
		p.buf = p.buf0[:0]
	}
}

func (p *printer) release() {
	if cap(p.buf) > (32 << 10) {
		return
	}

	p.buf = p.buf[:0]
	p.state = state{}
	printerPool.Put(p)
}

func (p *printer) printf(cb CB, msg string, vs []interface{}) (str string, used int) {
	args := argstate{args: vs}
	st := &p.state
	i := 0
	end := len(msg)

	for i < end {
		var (
			lasti = i
			err   error
		)

		i = findFmt(msg, i, end)
		if i+1 >= end {
			i = lasti
			break
		}
		if i > lasti {
			p.WriteString(msg[lasti:i])
		}

		// print '%' and skip processing if '%%' pattern is found
		if msg[i+1] == '%' {
			p.buf.WriteByte('%')
			i += 2
			continue
		}

		i, err = parseFmt(st, msg, i, end)
		arg, argIdx, exists := args.next()
		if !exists && err == nil {
			err = errMissingArg
		}
		if err != nil {
			p.writeErr(st, arg, err)
			continue
		}

		if st.flags.named || isErrorValue(arg) || isFieldValue(arg) {
			cb(st.field, argIdx, arg)
		}
		p.writeArg(arg, p.state.verb)
	}

	// return original string in case `msg` did not contain any format specifier
	if len(p.buf) == 0 {
		return msg, 0
	}

	p.WriteString(msg[i:]) // write missing/pending string
	return string(p.buf), args.idx
}

func (p *printer) writeArg(arg interface{}, verb byte) {
	p.state.arg = arg
	p.state.value = reflect.Value{}

	if arg == nil {
		switch verb {
		case 'T', 'v':
			p.writePadString("<nil>")
		default:
			p.writeBadVerb(verb)
		}
		return
	}

	// Type and pointer are special. Let's treat them first
	switch verb {
	case 'T':
		p.fmtStr(reflect.TypeOf(arg).String())
		return
	case 'p':
		p.fmtPointer(reflect.ValueOf(arg), 'p')
	}

	// try to print primitive types without reflection
	switch value := arg.(type) {
	case bool:
		p.fmtBool(value, p.state.verb)
	case int:
		p.fmtInt(uint64(value), true, p.state.verb)
	case int8:
		p.fmtInt(uint64(value), true, p.state.verb)
	case int16:
		p.fmtInt(uint64(value), true, p.state.verb)
	case int32:
		p.fmtInt(uint64(value), true, p.state.verb)
	case int64:
		p.fmtInt(uint64(value), true, p.state.verb)
	case uint:
		p.fmtInt(uint64(value), true, p.state.verb)
	case uint8:
		p.fmtInt(uint64(value), true, p.state.verb)
	case uint16:
		p.fmtInt(uint64(value), true, p.state.verb)
	case uint32:
		p.fmtInt(uint64(value), true, p.state.verb)
	case uint64:
		p.fmtInt(value, true, p.state.verb)
	case uintptr:
		p.fmtInt(uint64(value), true, p.state.verb)
	case string:
		p.fmtString(value, p.state.verb)
	case []byte:
		p.fmtBytes("[]byte", value, p.state.verb)
	case float32:
		p.fmtFloat(float64(value), 32, p.state.verb)
	case float64:
		p.fmtFloat(value, 32, p.state.verb)
	case complex64:
		p.fmtComplex(complex128(value), 64, verb)
	case complex128:
		p.fmtComplex(value, 128, verb)
	default:
		p.writeValue(reflect.ValueOf(arg), verb, 0)
	}
}

func (p *printer) writeValue(v reflect.Value, verb byte, depth int) {
	if p.handleMethods(v, verb) {
		return
	}

	p.state.arg = nil
	p.state.value = v

	flags := &p.state.flags

	switch v.Kind() {
	case reflect.Invalid:
		switch {
		case depth == 0:
			p.WriteString("<invalid reflect.Value>")
		case verb == 'v':
			p.WriteString("<nil>")
		default:
			p.writeBadVerb(verb)
		}

	case reflect.Bool:
		p.fmtBool(v.Bool(), verb)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.fmtInt(uint64(v.Int()), true, verb)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p.fmtInt(v.Uint(), false, verb)
	case reflect.Float32:
		p.fmtFloat(v.Float(), 32, verb)
	case reflect.Float64:
		p.fmtFloat(v.Float(), 64, verb)
	case reflect.Complex64:
		p.fmtComplex(v.Complex(), 64, verb)
	case reflect.Complex128:
		p.fmtComplex(v.Complex(), 128, verb)
	case reflect.String:
		p.fmtString(v.String(), verb)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		p.fmtPointer(v, verb)
	case reflect.Ptr:
		if depth == 0 && v.Pointer() != 0 {
			elem := v.Elem()
			switch elem.Kind() {
			case reflect.Map, reflect.Struct, reflect.Array, reflect.Slice:
				p.WriteByte('&')
				p.writeValue(elem, verb, depth+1)
				return
			}
		}
		p.fmtPointer(v, verb)

	case reflect.Interface:
		switch elem := v.Elem(); {
		case elem.IsValid():
			p.writeValue(elem, verb, depth+1)
		case flags.sharp:
			p.WriteString(v.Type().String())
			p.WriteString("(nil)")
		default:
			p.WriteString("<nil>")
		}

	case reflect.Map:
		if flags.sharp {
			p.WriteString(v.Type().String())
			if v.IsNil() {
				p.WriteString("(nil)")
				return
			}
			p.WriteByte('{')
		} else {
			p.buf.WriteString("map[")
		}

		keys := v.MapKeys()
		for i, key := range keys {
			if i > 0 {
				if flags.sharp {
					p.WriteString(", ")
				} else {
					p.WriteByte(' ')
				}
			}

			p.writeValue(key, verb, depth+1)
			p.WriteByte(':')
			p.writeValue(v.MapIndex(key), verb, depth+1)
		}

		if flags.sharp {
			p.WriteByte('}')
		} else {
			p.WriteByte(']')
		}

	case reflect.Struct:
		if flags.sharp {
			p.WriteString(v.Type().String())
		}
		p.WriteByte('{')
		for i := 0; i < v.NumField(); i++ {
			if i > 0 {
				if flags.sharp {
					p.WriteString(", ")
				} else {
					p.WriteByte(' ')
				}
			}

			if flags.plus || flags.sharp {
				if name := v.Type().Field(i).Name; name != "" {
					p.WriteString(name)
					p.WriteByte(':')
				}
			}

			fld := v.Field(i)
			if fld.Kind() == reflect.Interface && !fld.IsNil() {
				fld = fld.Elem()
			}
			p.writeValue(fld, verb, depth+1)
		}
		p.WriteByte('}')

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
				p.fmtBytes(t.String(), bytes, verb)
				return
			}
		}

		if flags.sharp {
			p.WriteString(v.Type().String())
			if v.Kind() == reflect.Slice && v.IsNil() {
				p.WriteString("(nil)")
				return
			}

			p.WriteByte('{')
			for i := 0; i < v.Len(); i++ {
				if i > 0 {
					p.WriteString(", ")
				}
				p.writeValue(v.Index(i), verb, depth+1)
			}
			p.WriteByte('}')
			return
		}

		p.WriteByte('[')
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				p.WriteByte(' ')
			}
			p.writeValue(v.Index(i), verb, depth+1)
		}
		p.WriteByte(']')
	}
}

func (p *printer) handleMethods(v reflect.Value, verb byte) bool {
	flags := &p.state.flags

	if p.state.inError || !v.IsValid() || !v.CanInterface() {
		return false
	}

	arg := v.Interface()
	if formatter, ok := arg.(fmt.Formatter); ok {
		defer p.recoverPanic(arg, verb)
		formatter.Format(p, rune(verb))
		return true
	}

	if flags.sharp {
		stringer, ok := arg.(fmt.GoStringer)
		if ok {
			defer p.recoverPanic(arg, verb)
			p.fmtStr(stringer.GoString())
		}
		return ok
	}

	switch verb {
	case 'v', 's', 'x', 'X', 'q':
		break
	default:
		return false
	}

	switch v := arg.(type) {
	case error:
		defer p.recoverPanic(arg, verb)
		p.fmtString(v.Error(), verb)
	case fmt.Stringer:
		defer p.recoverPanic(arg, verb)
		p.fmtString(v.String(), verb)
	default:
		return false
	}
	return true
}

func (p *printer) recoverPanic(arg interface{}, verb byte) {
	if err := recover(); err != nil {
		if v := reflect.ValueOf(arg); v.Kind() == reflect.Ptr && v.IsNil() {
			p.WriteString("<nil>")
			return
		}
		if p.state.inPanic {
			// nested recursive panic.
			panic(err)
		}

		oldFlags := p.state.flags
		p.state.flags = flags{}

		p.WriteString("%!")
		p.WriteByte(verb)
		p.WriteString("(PANIC=")
		p.state.inPanic = true
		p.writeArg(err, 'v')
		p.state.inPanic = false
		p.WriteByte(')')

		p.state.flags = oldFlags
	}
}

func (p *printer) fmtBool(b bool, verb byte) {
	switch verb {
	case 't', 'v':
		if b {
			p.writePadString("true")
		} else {
			p.writePadString("false")
		}
	default:
		p.writeBadVerb(verb)
	}
}

func (p *printer) fmtInt(v uint64, signed bool, verb byte) {
	flags := &p.state.flags

	switch verb {
	case 'v':
		if flags.sharp && !signed {
			p.fmtIntBase(v, 10, signed, true, lHexDigits)
		} else {
			p.fmtIntBase(v, 10, signed, false, lHexDigits)
		}
	case 'd':
		p.fmtIntBase(v, 10, signed, false, lHexDigits)
	case 'b':
		p.fmtIntBase(v, 2, signed, false, lHexDigits)
	case 'o':
		p.fmtIntBase(v, 8, signed, false, lHexDigits)
	case 'x':
		p.fmtIntBase(v, 16, signed, false, lHexDigits)
	case 'X':
		p.fmtIntBase(v, 16, signed, false, uHexDigits)
	case 'c':
		p.fmtRune(v)
	case 'q':
		if v <= utf8.MaxRune {
			p.fmtRuneQ(v)
		} else {
			p.writeBadVerb(verb)
		}
	case 'U':
		p.fmtUnicode(v)
	default:
		p.writeBadVerb(verb)
	}
}

func (p *printer) fmtRune(v uint64) {
	n := utf8.EncodeRune(p.fmtBuf[:], convRune(v))
	p.writePad(p.fmtBuf[:n])
}

func (p *printer) fmtRuneQ(v uint64) {
	r := convRune(v)
	buf := p.fmtBuf[:0]
	if p.state.flags.plus {
		buf = strconv.AppendQuoteRuneToASCII(buf, r)
	} else {
		buf = strconv.AppendQuoteRune(buf, r)
	}
	p.writePad(buf)
}

func convRune(v uint64) rune {
	if v > utf8.MaxRune {
		return utf8.RuneError
	}
	return rune(v)
}

func (p *printer) fmtUnicode(u uint64) {
	precision := 4

	// prepare temporary format buffer
	buf := p.fmtBuf[0:]
	if p.state.flags.hasPrecision && p.state.precision > precision {
		precision = p.state.precision
		width := 2 + precision + 2 + utf8.UTFMax + 1
		if width > len(buf) {
			buf = make([]byte, width)
		}
	}

	// format from right to left
	i := len(buf)

	// print rune with quote if '#' is set and rune is quoted
	if p.state.flags.sharp && u < utf8.MaxRune && strconv.IsPrint(rune(u)) {
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

	// ensure we pad with '0' always:
	p.writePadWith(buf[i:], '0')
}

func (p *printer) fmtIntBase(v uint64, base int, signed, forcePrefix bool, digits string) {
	flags := &p.state.flags

	neg := signed && int64(v) < 0
	if neg {
		v = -v
	}

	buf := p.fmtBuf[:]
	if flags.hasWidth || flags.hasPrecision {
		width := 3 + p.state.width + p.state.precision
		if width > len(buf) {
			buf = make([]byte, width)
		}
	}

	precision := 0
	if flags.hasPrecision {
		precision = p.state.precision
		if precision == 0 && v == 0 {
			p.writePaddingZeros(p.state.width)
			return
		}
	} else if flags.zero && flags.hasWidth {
		precision = p.state.width
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
		for ; v >= 8; v >>= 1 {
			i--
			buf[i] = byte('0' + v&1)
		}
	default:
		panic("unknown base")
	}
	i--
	buf[i] = digits[v]

	// left-pad zeros
	for i > 0 && precision > len(buf)-1 {
		i--
		buf[i] = '0'
	}

	// '#' triggers prefix
	if flags.sharp {
		switch base {
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

	p.writePadWith(buf[i:], ' ')
}

func (p *printer) fmtString(s string, verb byte) {
	switch verb {
	case 'v':
		if p.state.flags.sharp {
			p.fmtQualified(s)
		} else {
			p.fmtStr(s)
		}
	case 's':
		p.fmtStr(s)
	case 'x':
		p.fmtStrHex(s, lHexDigits)
	case 'X':
		p.fmtStrHex(s, uHexDigits)
	case 'q':
		p.fmtQualified(s)
	default:
		p.writeBadVerb(verb)
	}
}

func (p *printer) fmtBytes(typeName string, b []byte, verb byte) {
	flags := &p.state.flags

	switch verb {
	case 'v', 'd':
		if flags.sharp { // print hex dump of '#' is set
			p.WriteString(typeName)
			if b == nil {
				p.WriteString("(nil)")
				return
			}

			p.WriteByte('{')
			for i, c := range b {
				if i > 0 {
					p.WriteString(", ")
				}
				p.fmtIntBase(uint64(c), 16, false, true, lHexDigits)
			}
			p.WriteByte('}')
		} else { // print base-10 digits if '#' is not set
			p.WriteByte('[')
			for i, c := range b {
				if i > 0 {
					p.WriteByte(' ')
				}
				p.fmtIntBase(uint64(c), 10, false, false, lHexDigits)
			}
			p.WriteByte(']')
		}
	case 's':
		p.fmtStr(string(b))
	case 'x':
		p.fmtBytesHex(b, lHexDigits)
	case 'X':
		p.fmtBytesHex(b, uHexDigits)
	case 'q':
		p.fmtQualified(string(b))
	default:
		p.writeValue(reflect.ValueOf(b), verb, 0)
	}
}

func (p *printer) fmtStrHex(s string, digits string) {
	p.fmtSBHex(s, nil, digits)
}

func (p *printer) fmtBytesHex(b []byte, digits string) {
	p.fmtSBHex("", b, digits)
}

func (p *printer) fmtSBHex(s string, b []byte, digits string) {
	flags := &p.state.flags

	N := len(b)
	if b == nil {
		N = len(s)
	}

	if p.state.flags.hasPrecision && p.state.precision < N {
		N = p.state.precision
	}

	// Compute total width of hex encoded string. Each byte requires 2 symbols.
	// Codes are separates by a space character if the 'space' flag is set.
	// Leading 0x will be added if the '#' modifier has been used.
	width := 2 * N
	if width <= 0 {
		if flags.hasWidth {
			p.writePadding(p.state.width)
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
	needsPadding := p.state.width > width && flags.hasWidth

	// handle left padding if '-' modifier is not set.
	if needsPadding && !flags.minus {
		p.writePadding(p.state.width - width)
	}

	start, end := p.buf.grow(width)
	buf := p.buf[start:end]
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

	// handle right padding if '-' modifier is set.
	if needsPadding && flags.minus {
		p.writePadding(p.state.width - width)
	}
}

func (p *printer) fmtQualified(s string) {
	flags := &p.state.flags
	s = p.truncate(s)

	if flags.sharp && strconv.CanBackquote(s) {
		p.writePadString("`" + s + "`")
		return
	}

	buf := p.fmtBuf[:0]
	if flags.plus {
		buf = strconv.AppendQuoteToASCII(buf, s)
	} else {
		buf = strconv.AppendQuote(buf, s)
	}

	p.writePad(buf)
}

func (p *printer) fmtFloat(f float64, sz int, verb byte) {
	flags := &p.state.flags

	var precision int
	switch verb {
	case 'v':
		verb, precision = 'g', -1
	case 'b', 'g', 'G':
		precision = -1
	case 'f', 'e', 'E':
		precision = 6
	case 'F':
		verb, precision = 'f', 6
	default:
		p.writeBadVerb(verb)
		return
	}
	if flags.hasPrecision {
		precision = p.state.precision
	}

	// format number + ensure sign is always present
	buf := strconv.AppendFloat(p.fmtBuf[:1], f, verb, precision, sz)
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
		p.writePadWith(buf, ' ')
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
			case 'e', 'E':
				exp = append(exp, buf[i:]...)
				buf = buf[:i]
			default:
				digits--
			}
		}
		if digits > 0 {
			if !hasDecimal {
				buf = append(buf, '.')
			}
			for digits > 0 {
				buf = append(buf, '0')
				digits--
			}
		}
		buf = append(buf, exp...)
	}

	// print number with sign
	if flags.plus || buf[0] != '+' {
		if flags.zero && flags.hasWidth && p.state.width > len(buf) {
			p.WriteByte(buf[0])
			p.writePadding(p.state.width - len(buf))
			p.Write(buf[1:])
			return
		}

		p.writePad(buf)
		return
	}

	// print positive number without sign
	p.writePad(buf[1:])
}

func (p *printer) fmtComplex(v complex128, sz int, verb byte) {
	switch verb {
	case 'v', 'b', 'g', 'G', 'f', 'F', 'e', 'E':
		break
	default:
		p.writeBadVerb(verb)
	}

	p.WriteByte('(')
	p.fmtFloat(real(v), sz/2, verb)

	oldPlus := p.state.flags.plus
	p.state.flags.plus = true
	p.fmtFloat(imag(v), sz/2, verb)
	p.WriteString("i)")
	p.state.flags.plus = oldPlus
}

func (p *printer) fmtStr(s string) {
	p.writePadString(p.truncate(s))
}

func (p *printer) fmtPointer(value reflect.Value, verb byte) {
	var u uintptr
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		u = value.Pointer()
	default:
		p.writeBadVerb(verb)
		return
	}

	switch verb {
	case 'v':
		if p.state.flags.sharp {
			p.WriteByte('(')
			p.WriteString(value.Type().String())
			p.WriteString(")(")
			if u == 0 {
				p.WriteString("nil")
			} else {
				p.fmtIntBase(uint64(u), 16, false, true, lHexDigits)
			}
			p.WriteByte(')')
		} else if u != 0 {
			p.fmtIntBase(uint64(u), 16, false, true, lHexDigits)
		} else {
			p.writePadString("<nil>")
		}
	case 'p':
		p.fmtIntBase(uint64(u), 16, false, true, lHexDigits)
	case 'b', 'o', 'd', 'x', 'X':
		p.fmtInt(uint64(u), false, verb)
	default:
		p.writeBadVerb(verb)
	}
}

func (p *printer) writePadString(s string) {
	if !p.state.flags.hasWidth || p.state.width == 0 {
		p.WriteString(s)
		return
	}

	width := p.state.width - utf8.RuneCountInString(s)
	if p.state.flags.minus {
		p.writePadding(width)
		p.WriteString(s)
	} else {
		p.WriteString(s)
		p.writePadding(width)
	}
}

func (p *printer) writePad(b []byte) {
	if !p.state.flags.hasWidth || p.state.width == 0 {
		p.Write(b)
		return
	}

	width := p.state.width - utf8.RuneCount(b)
	if p.state.flags.minus {
		p.writePadding(width)
		p.Write(b)
	} else {
		p.Write(b)
		p.writePadding(width)
	}
}

func (p *printer) writePadWith(b []byte, padByte byte) {
	if !p.state.flags.hasWidth || p.state.width == 0 {
		p.Write(b)
		return
	}

	width := p.state.width - utf8.RuneCount(b)
	if p.state.flags.minus {
		p.writePaddingWith(width, padByte)
		p.Write(b)
	} else {
		p.Write(b)
		p.writePaddingWith(width, padByte)
	}
}

func (p *printer) writePadding(n int) {
	padByte := byte(' ')
	if p.state.flags.zero {
		padByte = byte('0')
	}
	p.writePaddingWith(n, padByte)
}

func (p *printer) writePaddingZeros(n int) {
	p.writePaddingWith(n, '0')
}

func (p *printer) writePaddingWith(n int, padByte byte) {
	if n <= 0 {
		return
	}

	start, end := p.buf.grow(n)
	padding := p.buf[start:end]
	for i := range padding {
		padding[i] = padByte
	}
}

func (p *printer) writeErr(st *state, arg interface{}, err error) {
	switch err {
	case errInvalidVerb:
		p.WriteString("%!")
		p.WriteByte(st.verb)
		p.WriteString("(INVALID)")
	case errNoVerb:
		p.WriteString("%!{}(NO VERB)")
	case errCloseMissing:
		p.WriteString("%!{}(MISSING })")
	case errNoFieldName:
		p.WriteString("%!{}(NO FIELD)")
	case errMissingArg:
		p.WriteString("%!")
		p.WriteByte(st.verb)
		p.WriteString("(MISSING)")
	}
}

func (p *printer) writeBadVerb(verb byte) {
	st := &p.state
	st.inError = true

	p.WriteString("%!")
	p.WriteByte(verb)
	p.WriteByte('(')

	switch {
	case st.arg != nil:
		p.WriteString(reflect.TypeOf(st.arg).String())
		p.WriteByte('=')
		p.writeArg(st.arg, 'v')
	case st.value.IsValid():
		p.WriteString(reflect.TypeOf(st.arg).String())
		p.WriteByte('=')
		p.writeValue(st.value, 'v', 0)
	default:
		p.WriteString("<nil>")
	}

	p.WriteByte(')')
	st.inError = false
}

// truncate returns the number of configured unicode symbols.
func (p *printer) truncate(s string) string {
	if p.state.flags.hasPrecision {
		n := p.state.precision
		for i := range s { // handle utf-8 with help of range loop
			n--
			if n < 0 {
				return s[:i]
			}
		}
	}
	return s
}

func parseFmt(st *state, msg string, start, end int) (i int, err error) {
	*st = state{}
	inPrec := false

	i = start + 1
	if i < end && msg[i] == '{' {
		return parseField(st, msg, start, end)
	}

	for i < end {
		if newi, isflag := parseFlag(st, msg, i); isflag {
			i = newi
			continue
		}

		if msg[i] == '.' && !inPrec {
			inPrec = true
			i++
			continue
		}

		if validVerbs[msg[i]] {
			st.verb = msg[i]
			return i + 1, err
		}

		num, isnum, newi := parseNum(msg, i, end)
		if !isnum {
			st.verb = msg[i]
			return i + 1, errInvalidVerb
		}

		i = newi
		if inPrec {
			st.precision = num
		} else {
			st.width = num
		}
	}

	return end, errNoVerb
}

// parseField parses a named field format specifier into st.
// The syntax of a field formatter is '%{[+#@]<name>[:<format>]}'.
//
// The prefix '+', '#', '@' modify the printing if no format is configured.
// In this case the 'v' verb is assumed. The '@' flag is synonymous to '#'.
//
// The 'format' section can be any valid format specification
func parseField(st *state, msg string, start, end int) (i int, err error) {
	st.flags.named = true
	st.verb = 'v' // default verb for fields is 'v'

	i = start + 2 // start is at '%'
	if i >= end {
		return end, errCloseMissing
	}

	switch msg[i] {
	case '+':
		st.flags.plus = true
		i++
	case '#', '@':
		st.flags.sharp = true
		i++
	}

	pos := i
	for i < end && msg[i] != '}' && msg[i] != ':' {
		i++
	}

	if pos == i {
		return i, errNoFieldName
	}
	st.field = msg[pos:i]

	if msg[i] == '}' {
		return i + 1, nil
	}

	// msg[i] == ':' => parse format specification
	i++
	inPrec := false
	for i < end && msg[i] != '}' {
		if newi, isflag := parseFlag(st, msg, i); isflag {
			i = newi
			continue
		}

		if msg[i] == '.' && !inPrec {
			inPrec = true
			i++
			continue
		}

		if validVerbs[msg[i]] {
			st.verb = msg[i]

			// Verbs required closing '}'. Skip until end of field formatter
			// and validate if verb is correct.
			// The presence of a verb directly stops the parser loop
			pos := i
			i++
			for i < end && msg[i] != '}' {
				i++
			}
			if i >= end {
				return i, errCloseMissing
			} else if pos+1 != i {
				return i + 1, errInvalidVerb
			} else {
				return i + 1, nil
			}
		}

		num, isnum, newi := parseNum(msg, i, end)
		if !isnum {
			st.verb = msg[i]

			// skip to end of formatter:
			for i < end && msg[i] != '}' {
				i++
			}
			return i + 1, errInvalidVerb
		}

		i = newi
		if inPrec {
			st.precision = num
		} else {
			st.width = num
		}
	}

	return i + 1, nil
}

func parseFlag(st *state, msg string, pos int) (int, bool) {
	switch msg[pos] {
	case '#':
		st.flags.sharp = true
		return pos + 1, true
	case '+':
		st.flags.plus = true
		return pos + 1, true
	case '-':
		st.flags.minus = true
		st.flags.zero = false
		return pos + 1, true
	case '0':
		st.flags.zero = !st.flags.minus
		return pos + 1, true
	case ' ':
		st.flags.space = true
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

func findFmt(in string, start, end int) (i int) {
	for i = start; i < end && in[i] != '%'; {
		i++
	}
	return i
}

func (p *printer) Width() (int, bool) {
	return p.state.width, p.state.flags.hasWidth
}

func (p *printer) Precision() (int, bool) {
	return p.state.precision, p.state.flags.hasPrecision
}

func (p *printer) Flag(c int) bool {
	flags := &p.state.flags
	switch c {
	case '-':
		return flags.minus
	case '+':
		return flags.plus
	case '#':
		return flags.sharp
	case ' ':
		return flags.space
	case '0':
		return flags.zero
	default:
		return false
	}
}

func (p *printer) Write(in []byte) (int, error) {
	p.buf.Write(in)
	return len(in), nil
}

func (p *printer) WriteString(s string) (int, error) {
	p.buf.WriteString(s)
	return len(s), nil
}

func (p *printer) WriteRune(r rune) error {
	p.buf.WriteRune(r)
	return nil
}

func (p *printer) WriteByte(b byte) error {
	p.buf.WriteByte(b)
	return nil
}

func (b *buffer) grow(n int) (oldLen, newLen int) {
	oldLen = len(*b)
	newLen = oldLen + n
	if newLen > cap(*b) {
		tmp := make(buffer, cap(*b)*2+n)
		copy(tmp, *b)
		*b = tmp
	}
	(*b) = (*b)[:newLen]
	return oldLen, newLen
}

func (b *buffer) Write(p []byte)       { *b = append(*b, p...) }
func (b *buffer) WriteString(s string) { *b = append(*b, s...) }
func (b *buffer) WriteByte(v byte)     { *b = append(*b, v) }
func (b *buffer) WriteRune(r rune) {
	if r < utf8.RuneSelf {
		*b = append(*b, byte(r))
		return
	}

	var runeBuf [utf8.UTFMax]byte
	n := utf8.EncodeRune(runeBuf[:], r)
	*b = append(*b, runeBuf[:n]...)
}

func (a *argstate) next() (arg interface{}, idx int, has bool) {
	if a.idx < len(a.args) {
		arg, idx = a.args[a.idx], a.idx
		a.idx++
		return arg, idx, true
	}
	return nil, len(a.args), false
}

func isErrorValue(v interface{}) bool {
	if err, ok := v.(error); ok {
		return err != nil
	}
	return false
}

func isFieldValue(v interface{}) bool {
	_, ok := v.(Field)
	return ok
}
