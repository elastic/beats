package hjson

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

type hjsonParser struct {
	data []byte
	at   int  // The index of the current character
	ch   byte // The current character
}

func (p *hjsonParser) resetAt() {
	p.at = 0
	p.ch = ' '
}

func isPunctuatorChar(c byte) bool {
	return c == '{' || c == '}' || c == '[' || c == ']' || c == ',' || c == ':'
}

func (p *hjsonParser) errAt(message string) error {
	var i int
	col := 0
	line := 1
	for i = p.at - 1; i > 0 && p.data[i] != '\n'; i-- {
		col++
	}
	for ; i > 0; i-- {
		if p.data[i] == '\n' {
			line++
		}
	}
	samEnd := p.at - col + 20
	if samEnd > len(p.data) {
		samEnd = len(p.data)
	}
	return fmt.Errorf("%s at line %d,%d >>> %s", message, line, col, string(p.data[p.at-col:samEnd]))
}

func (p *hjsonParser) next() bool {
	// get the next character.
	if p.at < len(p.data) {
		p.ch = p.data[p.at]
		p.at++
		return true
	}
	p.ch = 0
	return false
}

func (p *hjsonParser) peek(offs int) byte {
	pos := p.at + offs
	if pos >= 0 && pos < len(p.data) {
		return p.data[p.at+offs]
	}
	return 0
}

var escapee = map[byte]byte{
	'"':  '"',
	'\'': '\'',
	'\\': '\\',
	'/':  '/',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
}

func (p *hjsonParser) readString(allowML bool) (string, error) {

	// Parse a string value.
	res := new(bytes.Buffer)

	// callers make sure that (ch === '"' || ch === "'")
	// When parsing for string values, we must look for " and \ characters.
	exitCh := p.ch
	for p.next() {
		if p.ch == exitCh {
			p.next()
			if allowML && exitCh == '\'' && p.ch == '\'' && res.Len() == 0 {
				// ''' indicates a multiline string
				p.next()
				return p.readMLString()
			} else {
				return res.String(), nil
			}
		}
		if p.ch == '\\' {
			p.next()
			if p.ch == 'u' {
				uffff := 0
				for i := 0; i < 4; i++ {
					p.next()
					var hex int
					if p.ch >= '0' && p.ch <= '9' {
						hex = int(p.ch - '0')
					} else if p.ch >= 'a' && p.ch <= 'f' {
						hex = int(p.ch - 'a' + 0xa)
					} else if p.ch >= 'A' && p.ch <= 'F' {
						hex = int(p.ch - 'A' + 0xa)
					} else {
						return "", p.errAt("Bad \\u char " + string(p.ch))
					}
					uffff = uffff*16 + hex
				}
				res.WriteRune(rune(uffff))
			} else if ech, ok := escapee[p.ch]; ok {
				res.WriteByte(ech)
			} else {
				return "", p.errAt("Bad escape \\" + string(p.ch))
			}
		} else if p.ch == '\n' || p.ch == '\r' {
			return "", p.errAt("Bad string containing newline")
		} else {
			res.WriteByte(p.ch)
		}
	}
	return "", p.errAt("Bad string")
}

func (p *hjsonParser) readMLString() (value string, err error) {

	// Parse a multiline string value.
	res := new(bytes.Buffer)
	triple := 0

	// we are at ''' +1 - get indent
	indent := 0
	for {
		c := p.peek(-indent - 5)
		if c == 0 || c == '\n' {
			break
		}
		indent++
	}

	skipIndent := func() {
		skip := indent
		for p.ch > 0 && p.ch <= ' ' && p.ch != '\n' && skip > 0 {
			skip--
			p.next()
		}
	}

	// skip white/to (newline)
	for p.ch > 0 && p.ch <= ' ' && p.ch != '\n' {
		p.next()
	}
	if p.ch == '\n' {
		p.next()
		skipIndent()
	}

	// When parsing multiline string values, we must look for ' characters.
	lastLf := false
	for {
		if p.ch == 0 {
			return "", p.errAt("Bad multiline string")
		} else if p.ch == '\'' {
			triple++
			p.next()
			if triple == 3 {
				sres := res.Bytes()
				if lastLf {
					return string(sres[0 : len(sres)-1]), nil // remove last EOL
				}
				return string(sres), nil
			}
			continue
		} else {
			for triple > 0 {
				res.WriteByte('\'')
				triple--
				lastLf = false
			}
		}
		if p.ch == '\n' {
			res.WriteByte('\n')
			lastLf = true
			p.next()
			skipIndent()
		} else {
			if p.ch != '\r' {
				res.WriteByte(p.ch)
				lastLf = false
			}
			p.next()
		}
	}
}

func (p *hjsonParser) readKeyname() (string, error) {

	// quotes for keys are optional in Hjson
	// unless they include {}[],: or whitespace.

	if p.ch == '"' || p.ch == '\'' {
		return p.readString(false)
	}

	name := new(bytes.Buffer)
	start := p.at
	space := -1
	for {
		if p.ch == ':' {
			if name.Len() == 0 {
				return "", p.errAt("Found ':' but no key name (for an empty key name use quotes)")
			} else if space >= 0 && space != name.Len() {
				p.at = start + space
				return "", p.errAt("Found whitespace in your key name (use quotes to include)")
			}
			return name.String(), nil
		} else if p.ch <= ' ' {
			if p.ch == 0 {
				return "", p.errAt("Found EOF while looking for a key name (check your syntax)")
			}
			if space < 0 {
				space = name.Len()
			}
		} else {
			if isPunctuatorChar(p.ch) {
				return "", p.errAt("Found '" + string(p.ch) + "' where a key name was expected (check your syntax or use quotes if the key name includes {}[],: or whitespace)")
			}
			name.WriteByte(p.ch)
		}
		p.next()
	}
}

func (p *hjsonParser) white() {
	for p.ch > 0 {
		// Skip whitespace.
		for p.ch > 0 && p.ch <= ' ' {
			p.next()
		}
		// Hjson allows comments
		if p.ch == '#' || p.ch == '/' && p.peek(0) == '/' {
			for p.ch > 0 && p.ch != '\n' {
				p.next()
			}
		} else if p.ch == '/' && p.peek(0) == '*' {
			p.next()
			p.next()
			for p.ch > 0 && !(p.ch == '*' && p.peek(0) == '/') {
				p.next()
			}
			if p.ch > 0 {
				p.next()
				p.next()
			}
		} else {
			break
		}
	}
}

func (p *hjsonParser) readTfnns() (interface{}, error) {

	// Hjson strings can be quoteless
	// returns string, true, false, or null.

	if isPunctuatorChar(p.ch) {
		return nil, p.errAt("Found a punctuator character '" + string(p.ch) + "' when expecting a quoteless string (check your syntax)")
	}
	chf := p.ch
	value := new(bytes.Buffer)
	value.WriteByte(p.ch)

	for {
		p.next()
		isEol := p.ch == '\r' || p.ch == '\n' || p.ch == 0
		if isEol ||
			p.ch == ',' || p.ch == '}' || p.ch == ']' ||
			p.ch == '#' ||
			p.ch == '/' && (p.peek(0) == '/' || p.peek(0) == '*') {
			switch chf {
			case 'f':
				if strings.TrimSpace(value.String()) == "false" {
					return false, nil
				}
			case 'n':
				if strings.TrimSpace(value.String()) == "null" {
					return nil, nil
				}
			case 't':
				if strings.TrimSpace(value.String()) == "true" {
					return true, nil
				}
			default:
				if chf == '-' || chf >= '0' && chf <= '9' {
					if n, err := tryParseNumber(value.Bytes(), false); err == nil {
						return n, nil
					}
				}
			}
			if isEol {
				// remove any whitespace at the end (ignored in quoteless strings)
				return strings.TrimSpace(value.String()), nil
			}
		}
		value.WriteByte(p.ch)
	}
}

func (p *hjsonParser) readArray() (value interface{}, err error) {

	// Parse an array value.
	// assuming ch == '['

	array := make([]interface{}, 0, 1)

	p.next()
	p.white()

	if p.ch == ']' {
		p.next()
		return array, nil // empty array
	}

	for p.ch > 0 {
		var val interface{}
		if val, err = p.readValue(); err != nil {
			return nil, err
		}
		array = append(array, val)
		p.white()
		// in Hjson the comma is optional and trailing commas are allowed
		if p.ch == ',' {
			p.next()
			p.white()
		}
		if p.ch == ']' {
			p.next()
			return array, nil
		}
		p.white()
	}

	return nil, p.errAt("End of input while parsing an array (did you forget a closing ']'?)")
}

func (p *hjsonParser) readObject(withoutBraces bool) (value interface{}, err error) {
	// Parse an object value.

	object := make(map[string]interface{})

	if !withoutBraces {
		// assuming ch == '{'
		p.next()
	}

	p.white()
	if p.ch == '}' && !withoutBraces {
		p.next()
		return object, nil // empty object
	}
	for p.ch > 0 {
		var key string
		if key, err = p.readKeyname(); err != nil {
			return nil, err
		}
		p.white()
		if p.ch != ':' {
			return nil, p.errAt("Expected ':' instead of '" + string(p.ch) + "'")
		}
		p.next()
		// duplicate keys overwrite the previous value
		var val interface{}
		if val, err = p.readValue(); err != nil {
			return nil, err
		}
		object[key] = val
		p.white()
		// in Hjson the comma is optional and trailing commas are allowed
		if p.ch == ',' {
			p.next()
			p.white()
		}
		if p.ch == '}' && !withoutBraces {
			p.next()
			return object, nil
		}
		p.white()
	}

	if withoutBraces {
		return object, nil
	}
	return nil, p.errAt("End of input while parsing an object (did you forget a closing '}'?)")
}

func (p *hjsonParser) readValue() (interface{}, error) {

	// Parse a Hjson value. It could be an object, an array, a string, a number or a word.

	p.white()
	switch p.ch {
	case '{':
		return p.readObject(false)
	case '[':
		return p.readArray()
	case '"', '\'':
		return p.readString(true)
	default:
		return p.readTfnns()
	}
}

func (p *hjsonParser) rootValue() (interface{}, error) {
	// Braces for the root object are optional

	p.white()
	switch p.ch {
	case '{':
		return p.checkTrailing(p.readObject(false))
	case '[':
		return p.checkTrailing(p.readArray())
	}

	// assume we have a root object without braces
	res, err := p.checkTrailing(p.readObject(true))
	if err == nil {
		return res, nil
	}

	// test if we are dealing with a single JSON value instead (true/false/null/num/"")
	p.resetAt()
	if res2, err2 := p.checkTrailing(p.readValue()); err2 == nil {
		return res2, nil
	}
	return res, err
}

func (p *hjsonParser) checkTrailing(v interface{}, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}
	p.white()
	if p.ch > 0 {
		return nil, p.errAt("Syntax error, found trailing characters")
	}
	return v, nil
}

// Unmarshal parses the Hjson-encoded data and stores the result
// in the value pointed to by v.
//
// Unmarshal uses the inverse of the encodings that
// Marshal uses, allocating maps, slices, and pointers as necessary.
//
func Unmarshal(data []byte, v interface{}) (err error) {
	var value interface{}
	parser := &hjsonParser{data, 0, ' '}
	parser.resetAt()
	value, err = parser.rootValue()
	if err != nil {
		return err
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("non-pointer %v", reflect.TypeOf(v))
	}
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	rv.Set(reflect.ValueOf(value))
	return err
}
