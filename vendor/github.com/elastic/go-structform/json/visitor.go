package json

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode/utf8"

	structform "github.com/elastic/go-structform"
)

// Visitor implements the structform.Visitor interface, json encoding the
// structure being visited
type Visitor struct {
	w writer

	scratch [64]byte

	first   boolStack
	inArray boolStack
}

type boolStack struct {
	stack   []bool
	stack0  [32]bool
	current bool
}

var _ structform.Visitor = &Visitor{}

type writer struct {
	out io.Writer
}

func (w writer) write(b []byte) error {
	_, err := w.out.Write(b)
	return err
}

func NewVisitor(out io.Writer) *Visitor {
	v := &Visitor{w: writer{out}}
	return v
}

func (vs *Visitor) writeByte(b byte) error {
	vs.scratch[0] = b
	return vs.w.write(vs.scratch[:1])
}

func (vs *Visitor) writeString(s string) error {
	return vs.w.write(str2Bytes(s))
}

func (vs *Visitor) OnObjectStart(_ int, _ structform.BaseType) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	vs.first.push(true)
	vs.inArray.push(false)
	return vs.writeByte('{')
}

func (vs *Visitor) OnObjectFinished() error {
	vs.first.pop()
	vs.inArray.pop()
	return vs.writeByte('}')
}

func (vs *Visitor) OnKeyRef(s []byte) error {
	if err := vs.onFieldNext(); err != nil {
		return err
	}

	err := vs.OnStringRef(s)
	if err == nil {
		err = vs.writeByte(':')
	}
	return err
}

func (vs *Visitor) OnKey(s string) error {
	if err := vs.onFieldNext(); err != nil {
		return err
	}

	err := vs.OnString(s)
	if err == nil {
		err = vs.writeByte(':')
	}
	return err
}

func (vs *Visitor) onFieldNext() error {
	if vs.first.current {
		vs.first.current = false
		return nil
	}
	return vs.writeByte(',')
}

func (vs *Visitor) OnArrayStart(_ int, _ structform.BaseType) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	vs.first.push(true)
	vs.inArray.push(true)
	return vs.writeByte('[')
}

func (vs *Visitor) OnArrayFinished() error {
	vs.first.pop()
	vs.inArray.pop()
	return vs.writeByte(']')
}

func (vs *Visitor) tryElemNext() error {
	if !vs.inArray.current {
		return nil
	}

	if vs.first.current {
		vs.first.current = false
		return nil
	}
	return vs.w.write(commaSymbol)
}

var hex = "0123456789abcdef"

func (vs *Visitor) OnStringRef(s []byte) error {
	return vs.OnString(bytes2Str(s))
}

func (vs *Visitor) OnString(s string) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	vs.writeByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' && b != '&' {
				i++
				continue
			}
			if start < i {
				vs.writeString(s[start:i])
			}
			switch b {
			case '\\', '"':
				vs.scratch[0], vs.scratch[1] = '\\', b
				vs.w.write(vs.scratch[:2])
			case '\n':
				vs.scratch[0], vs.scratch[1] = '\\', 'n'
				vs.w.write(vs.scratch[:2])
			case '\r':
				vs.scratch[0], vs.scratch[1] = '\\', 'r'
				vs.w.write(vs.scratch[:2])
			case '\t':
				vs.scratch[0], vs.scratch[1] = '\\', 't'
				vs.w.write(vs.scratch[:2])
			default:
				// This vsodes bytes < 0x20 except for \n and \r,
				// as well as <, > and &. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				vs.scratch[0], vs.scratch[1], vs.scratch[2], vs.scratch[3] = '\\', 'u', '0', '0'
				vs.scratch[4] = hex[b>>4]
				vs.scratch[5] = hex[b&0xF]
				vs.w.write(vs.scratch[:6])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			if start < i {
				vs.writeString(s[start:i])
			}
			vs.w.write(invalidCharSym)
			i += size
			start = i
			continue
		}
		// U+2028 is LINE SEPARATOR.
		// U+2029 is PARAGRAPH SEPARATOR.
		// They are both technically valid characters in JSON strings,
		// but don't work in JSONP, which has to be evaluated as JavaScript,
		// and can lead to security holes there. It is valid JSON to
		// escape them, so we do so unconditionally.
		// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
		if c == '\u2028' || c == '\u2029' {
			if start < i {
				vs.writeString(s[start:i])
			}
			vs.writeString(`\u202`)
			vs.writeByte(hex[c&0xF])
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		vs.writeString(s[start:])
	}
	vs.writeByte('"')
	return nil
}

func (vs *Visitor) OnBool(b bool) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	var err error
	if b {
		err = vs.w.write(trueSymbol)
	} else {
		err = vs.w.write(falseSymbol)
	}
	return err
}

func (vs *Visitor) OnNil() error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	err := vs.w.write(nullSymbol)
	return err
}

func (vs *Visitor) OnInt8(i int8) error {
	return vs.onInt(int64(i))
}

func (vs *Visitor) OnInt16(i int16) error {
	return vs.onInt(int64(i))
}

func (vs *Visitor) OnInt32(i int32) error {
	return vs.onInt(int64(i))
}

func (vs *Visitor) OnInt64(i int64) error {
	return vs.onInt(i)
}

func (vs *Visitor) OnInt(i int) error {
	return vs.onInt(int64(i))
}

func (vs *Visitor) onInt(v int64) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	/*
		b := strconv.AppendInt(vs.scratch[:0], i, 10)
		_, err := vs.w.Write(b)
	*/
	vs.onNumber(v < 0, uint64(v))
	return nil
}

func (vs *Visitor) OnUint8(u uint8) error {
	return vs.onUint(uint64(u))
}

func (vs *Visitor) OnByte(b byte) error {
	return vs.onUint(uint64(b))
}

func (vs *Visitor) OnUint16(u uint16) error {
	return vs.onUint(uint64(u))
}

func (vs *Visitor) OnUint32(u uint32) error {
	return vs.onUint(uint64(u))
}

func (vs *Visitor) OnUint64(u uint64) error {
	return vs.onUint(u)
}

func (vs *Visitor) OnUint(u uint) error {
	return vs.onUint(uint64(u))
}

func (vs *Visitor) onUint(u uint64) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	return vs.onNumber(false, u)
	/*
		b := strconv.AppendUint(vs.scratch[:0], u, 10)
		_, err := vs.w.Write(b)
		return err
	*/
}

func (vs *Visitor) onNumber(neg bool, u uint64) error {
	if neg {
		u = -u
	}
	i := len(vs.scratch)

	// common case: use constants for / because
	// the compiler can optimize it into a multiply+shift
	if ^uintptr(0)>>32 == 0 {
		for u > uint64(^uintptr(0)) {
			q := u / 1e9
			us := uintptr(u - q*1e9) // us % 1e9 fits into a uintptr
			for j := 9; j > 0; j-- {
				i--
				qs := us / 10
				vs.scratch[i] = byte(us - qs*10 + '0')
				us = qs
			}
			u = q
		}
	}

	// u guaranteed to fit into a uintptr
	us := uintptr(u)
	for us >= 10 {
		i--
		q := us / 10
		vs.scratch[i] = byte(us - q*10 + '0')
		us = q
	}
	// u < 10
	i--
	vs.scratch[i] = byte(us + '0')

	if neg {
		i--
		vs.scratch[i] = '-'
	}
	return vs.w.write(vs.scratch[i:])
}

func (vs *Visitor) OnFloat32(f float32) error {
	return vs.onFloat(float64(f), 32)
}

func (vs *Visitor) OnFloat64(f float64) error {
	return vs.onFloat(f, 64)
}

func (vs *Visitor) onFloat(f float64, bits int) error {
	if err := vs.tryElemNext(); err != nil {
		return err
	}

	if math.IsInf(f, 0) || math.IsNaN(f) {
		return fmt.Errorf("unsupported float value: %v", f)
	}

	b := strconv.AppendFloat(vs.scratch[:0], f, 'g', -1, bits)
	err := vs.w.write(b)
	return err
}

func (s *boolStack) init() {
	s.stack = s.stack0[:0]
}

func (s *boolStack) push(b bool) {
	s.stack = append(s.stack, s.current)
	s.current = b
}

func (s *boolStack) pop() {
	if len(s.stack) == 0 {
		panic("pop from empty stack")
	}

	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
}
