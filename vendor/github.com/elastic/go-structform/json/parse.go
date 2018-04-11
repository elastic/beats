package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	structform "github.com/elastic/go-structform"
)

type Parser struct {
	visitor    structform.Visitor
	strVisitor structform.StringRefVisitor

	// last fail state
	err error

	// parser state machine
	states       []state // state stack for nested arrays/objects
	currentState state

	// preallocate stack memory for up to 32 nested arrays/objects
	statesBuf [32]state

	literalBuffer  []byte
	literalBuffer0 [64]byte

	inEscape bool
	isDouble bool

	required int
}

var (
	errFailing               = errors.New("JSON parser failed")
	errIncomplete            = errors.New("Incomplete JSON input")
	errUnknownChar           = errors.New("unknown character")
	errQuoteMissing          = errors.New("missing closing quote")
	errExpectColon           = errors.New("expected ':' after map key")
	errUnexpectedDictClose   = errors.New("unexpected '}'")
	errUnexpectedArrClose    = errors.New("unexpected ']'")
	errExpectedDigit         = errors.New("expected a digit")
	errExpectedObject        = errors.New("expected JSON object")
	errExpectedArray         = errors.New("expected JSON array")
	errExpectedFieldName     = errors.New("expected JSON object field name")
	errExpectedInteger       = errors.New("expected integer value")
	errExpectedNull          = errors.New("expected null value")
	errExpectedFalse         = errors.New("expected false value")
	errExpectedTrue          = errors.New("expected true value")
	errExpectedArrayField    = errors.New("expected ']' or ','")
	errUnquoteInEscape       = errors.New("incomplete escape at end of string")
	errUnquoteInvalidChar    = errors.New("invalid character found in string")
	errUnquoteInvalidUnicode = errors.New("unicode escape is no hex number")
	errUnquoteUnknownEscape  = errors.New("unknown escape sequence")
)

type state uint8

//go:generate stringer -type=state
const (
	failedState state = iota
	startState

	arrState
	arrStateValue
	arrStateNext

	dictState
	dictFieldState
	dictNextFieldState
	dictFieldValue
	dictFieldValueSep
	dictFieldStateEnd

	nullState
	trueState
	falseState
	stringState
	numberState
)

func ParseReader(in io.Reader, vs structform.Visitor) (int64, error) {
	p := NewParser(vs)
	i, err := io.Copy(p, in)
	if err == nil {
		err = p.finalize()
	}
	return i, err
}

func Parse(b []byte, vs structform.Visitor) error {
	return NewParser(vs).Parse(b)
}

func ParseString(str string, vs structform.Visitor) error {
	return NewParser(vs).ParseString(str)
}

func NewParser(vs structform.Visitor) *Parser {
	p := &Parser{}
	p.init(vs)
	return p
}

func (p *Parser) init(vs structform.Visitor) {
	*p = Parser{
		visitor:      vs,
		strVisitor:   structform.MakeStringRefVisitor(vs),
		currentState: startState,
	}
	p.states = p.statesBuf[:0]
	p.literalBuffer = p.literalBuffer0[:0]
}

func (p *Parser) Parse(b []byte) error {
	p.states = p.states[:0]
	p.literalBuffer = p.literalBuffer[:0]
	p.currentState = startState

	p.err = p.feed(b)
	if p.err == nil {
		p.err = p.finalize()
	}
	return p.err
}

func (p *Parser) ParseString(str string) error {
	return p.Parse(str2Bytes(str))
}

func (p *Parser) Write(b []byte) (int, error) {
	p.err = p.feed(b)
	if p.err != nil {
		return 0, p.err
	}
	return len(b), nil
}

func (p *Parser) feed(b []byte) error {
	for len(b) > 0 {
		n, _, err := p.feedUntil(b)
		if err != nil {
			return err
		}

		b = b[n:]
	}

	return nil
}

func (p *Parser) feedUntil(b []byte) (int, bool, error) {
	var (
		err      error
		reported bool
		orig     = b
	)

	for !reported && len(b) > 0 {
		switch p.currentState {
		case failedState:
			if p.err == nil {
				p.err = errors.New("invalid parser state")
			}
			return 0, false, p.err
		case startState:
			b, reported, err = p.stepStart(b)

		case dictState:
			b, reported, err = p.stepDict(b, true)

		case dictNextFieldState:
			b, reported, err = p.stepDict(b, false)

		case dictFieldState:
			b, err = p.stepDictKey(b)

		case dictFieldValueSep:
			if b = trimLeft(b); len(b) > 0 {
				if b[0] != ':' {
					err = errExpectColon
				}
				b = b[1:]
				p.currentState = dictFieldValue
			}

		case dictFieldValue:
			b, reported, err = p.stepValue(b, dictFieldStateEnd)

		case dictFieldStateEnd:
			b, reported, err = p.stepDictValueEnd(b)

		case arrState:
			b, reported, err = p.stepArray(b, true)

		case arrStateValue:
			b, _, err = p.stepValue(b, arrStateNext)

		case arrStateNext:
			b, reported, err = p.stepArrValueEnd(b)

		case nullState:
			b, reported, err = p.stepNULL(b)

		case trueState:
			b, reported, err = p.stepTRUE(b)

		case falseState:
			b, reported, err = p.stepFALSE(b)

		case stringState:
			b, reported, err = p.stepString(b)

		case numberState:
			b, reported, err = p.stepNumber(b)

		default:
			return 0, false, errFailing
		}

		reported = reported && len(p.states) == 0
	}

	consumed := len(orig) - len(b)
	return consumed, reported, err
}

func (p *Parser) finalize() error {
	if p.currentState == numberState {
		err := p.reportNumber(p.literalBuffer, p.isDouble)
		if err != nil {
			return err
		}
		p.popState()
	}

	if len(p.states) > 0 && p.currentState != startState {
		return errIncomplete
	}

	return nil
}

func (p *Parser) pushState(next state) {
	if p.currentState != failedState {
		p.states = append(p.states, p.currentState)
	}
	p.currentState = next
}

func (p *Parser) popState() {
	if len(p.states) == 0 {
		p.currentState = failedState
	} else {
		last := len(p.states) - 1
		p.currentState = p.states[last]
		p.states = p.states[:last]
	}
}

func (p *Parser) stepStart(b []byte) ([]byte, bool, error) {
	return p.stepValue(b, p.currentState)
}

func (p *Parser) stepValue(b []byte, retState state) ([]byte, bool, error) {
	b = trimLeft(b)
	if len(b) == 0 {
		return b, false, nil
	}

	p.currentState = retState
	c := b[0]
	switch c {
	case '{': // start dictionary
		p.pushState(dictState)
		return b[1:], false, p.visitor.OnObjectStart(-1, structform.AnyType)

	case '[': // start array
		p.pushState(arrState)
		return b[1:], false, p.visitor.OnArrayStart(-1, structform.AnyType)

	case 'n': // parse "null"
		p.pushState(nullState)
		p.required = 3
		return p.stepNULL(b[1:])

	case 'f': // parse "false"
		p.pushState(falseState)
		p.required = 4
		return p.stepFALSE(b[1:])

	case 't': // parse "true"
		p.pushState(trueState)
		p.required = 3
		return p.stepTRUE(b[1:])

	case '"': // parse string
		p.literalBuffer = p.literalBuffer[:0]
		p.pushState(stringState)
		p.inEscape = false
		return p.stepString(b[:])

	default:
		// parse number?
		p.isDouble = false

		isNumber := c == '-' || c == '+' || c == '.' || isDigit(c)
		if !isNumber {
			return b, false, errUnknownChar
		}

		p.literalBuffer = p.literalBuffer0[:0]
		p.pushState(numberState)
		p.isDouble = false
		return p.stepNumber(b)
	}
}

func (p *Parser) stepDict(b []byte, allowEnd bool) ([]byte, bool, error) {
	b = trimLeft(b)
	if len(b) == 0 {
		return b, false, nil
	}

	c := b[0]
	switch c {
	case '}':
		if !allowEnd {
			return nil, false, errUnexpectedDictClose
		}
		return p.endDict(b)

	case '"':
		p.currentState = dictFieldState
		return b, false, nil

	default:
		return nil, false, errExpectedFieldName
	}
}

func (p *Parser) stepDictKey(b []byte) ([]byte, error) {
	ref, allocated, done, b, err := p.doString(b)
	if done && err == nil {
		p.currentState = dictFieldValueSep

		if !allocated {
			err = p.strVisitor.OnKeyRef(ref)
		} else {
			err = p.visitor.OnKey(bytes2Str(ref))
		}
	}
	return b, err
}

func (p *Parser) stepDictValueEnd(b []byte) ([]byte, bool, error) {
	b = trimLeft(b)
	if len(b) == 0 {
		return b, false, nil
	}

	c := b[0]
	switch c {
	case '}':
		return p.endDict(b)
	case ',':
		p.currentState = dictNextFieldState
		return b[1:], false, nil
	default:
		return nil, false, errUnknownChar
	}
}

func (p *Parser) endDict(b []byte) ([]byte, bool, error) {
	p.popState()
	return b[1:], true, p.visitor.OnObjectFinished()
}

func (p *Parser) stepArray(b []byte, allowEnd bool) ([]byte, bool, error) {
	b = trimLeft(b)
	if len(b) == 0 {
		return b, false, nil
	}

	c := b[0]
	switch c {
	case ']':
		if !allowEnd {
			return nil, false, errUnexpectedArrClose
		}
		return p.endArray(b)
	}

	p.currentState = arrStateValue
	return b, false, nil
}

func (p *Parser) stepArrValueEnd(b []byte) ([]byte, bool, error) {
	b = trimLeft(b)
	if len(b) == 0 {
		return b, false, nil
	}

	c := b[0]
	switch c {
	case ']':
		return p.endArray(b)
	case ',':
		p.currentState = arrStateValue
		return b[1:], false, nil
	default:
		return nil, false, errUnknownChar
	}
}

func (p *Parser) endArray(b []byte) ([]byte, bool, error) {
	p.popState()
	return b[1:], true, p.visitor.OnArrayFinished()
}

func (p *Parser) stepString(b []byte) ([]byte, bool, error) {
	ref, allocated, done, b, err := p.doString(b)
	if done && err == nil {
		p.popState()

		if !allocated {
			err = p.strVisitor.OnStringRef(ref)
		} else {
			err = p.visitor.OnString(bytes2Str(ref))
		}
	}
	return b, done, err
}

func (p *Parser) doString(b []byte) ([]byte, bool, bool, []byte, error) {
	stop := -1
	done := false

	delta := 1
	buf := b
	atStart := len(p.literalBuffer) == 0
	if atStart {
		delta = 2
		buf = b[1:]
	}

	inEscape := p.inEscape
	for i, c := range buf {
		if inEscape {
			inEscape = false
			continue
		}

		if c == '"' {
			done = true
			stop = i + delta
			break
		} else if c == '\\' {
			inEscape = true
		}
	}
	p.inEscape = inEscape

	if !done {
		p.literalBuffer = append(p.literalBuffer, b...)
		return nil, false, false, nil, nil
	}

	rest := b[stop:]
	b = b[:stop]
	if len(p.literalBuffer) > 0 {
		b = append(p.literalBuffer, b...)
		p.literalBuffer = b[:0] // reset buffer
	}

	var err error
	var allocated bool
	b = b[1 : len(b)-1]
	b, allocated, err = p.unquote(b)
	if err != nil {
		return nil, false, false, nil, err
	}

	return b, allocated, done, rest, nil
}

func (p *Parser) unquote(in []byte) ([]byte, bool, error) {
	if len(in) == 0 {
		return in, false, nil
	}

	// Check for unusual characters and escape sequence. If none is found,
	// return slice as is:
	i := 0
	for i < len(in) {
		c := in[i]
		if c == '\\' || c == '"' || c < ' ' {
			break
		}

		if c < utf8.RuneSelf {
			i++
			continue
		}

		r, sz := utf8.DecodeRune(in[i:])
		if r == utf8.RuneError && sz == 1 {
			break
		}

		i += sz
	}

	// no special character found -> return as is
	if i == len(in) {
		return in, false, nil
	}

	// found escape character (or other unusual character) ->
	// allocate output buffer (try to use literalBuffer)
	out := p.literalBuffer[:0]
	allocated := false
	utf8Delta := 2 * utf8.UTFMax
	minLen := len(in) + utf8Delta
	if cap(out) < minLen {
		// TODO: is minLen < some upper bound, store in literalBuffer
		out = make([]byte, minLen)
		allocated = true
	} else {
		out = out[:minLen]
	}

	// init output buffer
	written := copy(out, in[:i])

	for i < len(in) {
		if written > len(out)-utf8Delta {
			// out of room -> increase write buffer
			newLen := len(out) * 2
			if cap(out) < newLen {
				tmp := make([]byte, len(out)*2)
				copy(tmp, out[:written])
				out = tmp
				allocated = true
			} else {
				out = out[:newLen]
			}
		}

		c := in[i]
		switch {
		case c == '\\':
			i++
			if i >= len(in) {
				return nil, false, errUnquoteInEscape
			}

			switch in[i] {
			default:
				return nil, false, errUnquoteUnknownEscape
			case '"', '\\', '/', '\'':
				out[written] = in[i]
				i++
				written++
			case 'b':
				out[written] = '\b'
				i++
				written++
			case 'f':
				out[written] = '\f'
				i++
				written++
			case 'n':
				out[written] = '\n'
				i++
				written++
			case 'r':
				out[written] = '\r'
				i++
				written++
			case 't':
				out[written] = '\t'
				i++
				written++
			case 'u':
				i++
				code, err := strconv.ParseUint(string(in[i:i+4]), 16, 64)
				if err != nil {
					return nil, false, errUnquoteInvalidUnicode
				}

				i += 4
				r := rune(code)
				if utf16.IsSurrogate(r) {
					var dec rune = unicode.ReplacementChar

					valid := in[i] == '\\' && in[i+1] == 'u'
					if valid {
						code, err := strconv.ParseUint(string(in[i+2:i+6]), 16, 64)
						if err == nil {
							dec = utf16.DecodeRune(r, rune(code))
							if dec != unicode.ReplacementChar {
								i += 6
							}
						}
					}

					r = dec
				}
				written += utf8.EncodeRune(out[written:], r)
			}

		case c == '"', c < ' ':
			return nil, false, errUnquoteInvalidChar

		case c < utf8.RuneSelf:
			out[written] = c
			i++
			written++

		default:
			_, sz := utf8.DecodeRune(in[i:])
			i += sz
			written += copy(out[written:], in[i:i+sz])
		}
	}

	return out[:written], allocated, nil
}

func (p *Parser) stepNumber(b []byte) ([]byte, bool, error) {
	// search for char in stop-set
	stop := -1
	done := false
	for i, c := range b {
		isStopChar := c == ' ' || c == '\t' || c == '\f' || c == '\n' || c == '\r' ||
			c == ',' ||
			c == ']' ||
			c == '}'
		if isStopChar {
			stop = i
			done = true
			break
		}

		p.isDouble = p.isDouble || c == '.' || c == 'e' || c == 'E'
	}

	if !done {
		p.literalBuffer = append(p.literalBuffer, b...)
		return nil, false, nil
	}

	rest := b[stop:]
	b = b[:stop]
	if len(p.literalBuffer) > 0 {
		b = append(p.literalBuffer, b...)
		p.literalBuffer = b[:0] // reset buffer
	}

	err := p.reportNumber(b, p.isDouble)
	p.popState()
	return rest, true, err
}

func (p *Parser) reportNumber(b []byte, isDouble bool) error {
	// parse number
	var err error
	if isDouble {
		var f float64
		if f, err = strconv.ParseFloat(bytes2Str(b), 64); err == nil {
			err = p.visitor.OnFloat64(f)
		}
	} else {
		var i int64
		if i, err = parseInt(b); err == nil {
			err = p.visitor.OnInt64(i)
		}
	}

	return err
}

func parseInt(b []byte) (int64, error) {
	neg := false
	if b[0] == '+' {
		b = b[1:]
	} else if b[0] == '-' {
		neg = true
		b = b[1:]
	}

	u, err := parseUint(b)
	n := int64(u)
	if neg {
		n = -n
	}
	return n, err
}

func parseUint(b []byte) (uint64, error) {
	const cutoff = math.MaxUint64/10 + 1
	const maxVal = math.MaxUint64

	var n uint64

	for _, c := range b {
		d := int(c) - '0'
		if d < 0 || d > 9 {
			return 0, fmt.Errorf("'%s' is no valid number", b)
		}

		if n >= cutoff {
			return 0, fmt.Errorf("number overflow parsing '%v'", b)
		}

		n *= 10
		n1 := n + uint64(d)
		if n1 < n || n1 > maxVal {
			return 0, fmt.Errorf("number overflow parsing '%v'", b)
		}

		n = n1
	}

	return n, nil
}

func (p *Parser) stepNULL(b []byte) ([]byte, bool, error) {
	b, done, err := p.stepKind(b, []byte("null"), errExpectedNull)
	if done {
		err = p.visitor.OnNil()
	}
	return b, done, err
}

func (p *Parser) stepTRUE(b []byte) ([]byte, bool, error) {
	b, done, err := p.stepKind(b, []byte("true"), errExpectedTrue)
	if done {
		err = p.visitor.OnBool(true)
	}
	return b, done, err
}

func (p *Parser) stepFALSE(b []byte) ([]byte, bool, error) {
	b, done, err := p.stepKind(b, []byte("false"), errExpectedFalse)
	if done {
		err = p.visitor.OnBool(false)
	}
	return b, done, err
}

func (p *Parser) stepKind(b []byte, kind []byte, err error) ([]byte, bool, error) {
	n := p.required
	s := kind[len(kind)-n:]
	done := true
	if L := len(b); L < n {
		done = false
		p.required = n - L
		n = L
		s = s[:L]
	}

	if !bytes.HasPrefix(b, s) {
		return b, false, err
	}

	if done {
		p.popState()
	}
	return b[n:], done, nil
}

func isDigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func trimLeft(b []byte) []byte {
	for i, c := range b {
		if !unicode.IsSpace(rune(c)) {
			return b[i:]
		}
	}
	return nil
}

var whitespace = " \t\r\n"
