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

package parse

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type flagParser struct {
	input string
}

// stopSet definitions for handling unquoted strings
const (
	toplevelStopSet  = ","
	arrayElemStopSet = ",]"
	objKeyStopSet    = ":"
	objValueStopSet  = ",}"
)

// Value parses command line arguments, supporting
// boolean, numbers, strings, arrays, objects.
//
// The parser implements a superset of JSON, but only a subset of YAML by
// allowing for arrays and objects having a trailing comma. In addition 3
// strings types are supported:
//
// 1. single quoted string (no unescaping of any characters)
// 2. double quoted strings (characters are escaped)
// 3. strings without quotes. String parsing stops in
//   special characters like '[]{},:'
//
// In addition, top-level values can be separated by ',' to build arrays
// without having to use [].
func Value(content string) (interface{}, error) {
	p := &flagParser{strings.TrimSpace(content)}
	v, err := p.parse()
	if err != nil {
		return nil, fmt.Errorf("%v when parsing '%v'", err.Error(), content)
	}
	return v, nil
}

func (p *flagParser) parse() (interface{}, error) {
	var values []interface{}

	for {
		v, err := p.parseValue(toplevelStopSet)
		if err != nil {
			return nil, err
		}
		values = append(values, v)

		p.ignoreWhitespace()
		if p.input == "" {
			break
		}

		if err := p.expectChar(','); err != nil {
			return nil, err
		}
	}

	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		return values[0], nil
	}
	return values, nil
}

func (p *flagParser) parseValue(stopSet string) (interface{}, error) {
	p.ignoreWhitespace()
	in := p.input

	if in == "" {
		return nil, nil
	}

	switch in[0] {
	case '[':
		return p.parseArray()
	case '{':
		return p.parseObj()
	case '"':
		return p.parseStringDQuote()
	case '\'':
		return p.parseStringSQuote()
	default:
		return p.parsePrimitive(stopSet)
	}
}

func (p *flagParser) ignoreWhitespace() {
	p.input = strings.TrimLeftFunc(p.input, unicode.IsSpace)
}

func (p *flagParser) parseArray() (interface{}, error) {
	p.input = p.input[1:]

	var values []interface{}
loop:
	for {
		p.ignoreWhitespace()
		if p.input[0] == ']' {
			p.input = p.input[1:]
			break
		}

		v, err := p.parseValue(arrayElemStopSet)
		if err != nil {
			return nil, err
		}
		values = append(values, v)

		p.ignoreWhitespace()
		if p.input == "" {
			return nil, errors.New("array closing ']' missing")
		}

		next := p.input[0]
		p.input = p.input[1:]

		switch next {
		case ']':
			break loop
		case ',':
			continue
		default:
			return nil, errors.New("array expected ',' or ']'")
		}

	}

	if len(values) == 0 {
		return nil, nil
	}

	return values, nil
}

func (p *flagParser) parseObj() (interface{}, error) {
	p.input = p.input[1:]

	O := map[string]interface{}{}

loop:
	for {
		p.ignoreWhitespace()
		if p.input[0] == '}' {
			p.input = p.input[1:]
			break
		}

		k, err := p.parseKey()
		if err != nil {
			return nil, err
		}

		p.ignoreWhitespace()
		if err := p.expectChar(':'); err != nil {
			return nil, err
		}

		v, err := p.parseValue(objValueStopSet)
		if err != nil {
			return nil, err
		}

		if p.input == "" {
			return nil, errors.New("dictionary expected ',' or '}'")
		}

		O[k] = v
		next := p.input[0]
		p.input = p.input[1:]

		switch next {
		case '}':
			break loop
		case ',':
			continue
		default:
			return nil, errors.New("dictionary expected ',' or '}'")
		}
	}

	// empty object
	if len(O) == 0 {
		return nil, nil
	}

	return O, nil
}

func (p *flagParser) parseKey() (string, error) {
	in := p.input
	if in == "" {
		return "", errors.New("expected key")
	}

	switch in[0] {
	case '"':
		return p.parseStringDQuote()
	case '\'':
		return p.parseStringSQuote()
	default:
		return p.parseNonQuotedString(objKeyStopSet)
	}
}

func (p *flagParser) parseStringDQuote() (string, error) {
	in := p.input
	off := 1
	var i int
	for {
		i = strings.IndexByte(in[off:], '"')
		if i < 0 {
			return "", errors.New("Missing \" to close string ")
		}

		i += off
		if in[i-1] != '\\' {
			break
		}
		off = i + 1
	}

	p.input = in[i+1:]
	return strconv.Unquote(in[:i+1])
}

func (p *flagParser) parseStringSQuote() (string, error) {
	in := p.input
	i := strings.IndexByte(in[1:], '\'')
	if i < 0 {
		return "", errors.New("missing ' to close string")
	}

	p.input = in[i+2:]
	return in[1 : 1+i], nil
}

func (p *flagParser) parseNonQuotedString(stopSet string) (string, error) {
	in := p.input
	idx := strings.IndexAny(in, stopSet)
	if idx == 0 {
		return "", fmt.Errorf("unexpected '%v'", string(in[idx]))
	}

	content, in := in, ""
	if idx > 0 {
		content, in = content[:idx], content[idx:]
	}
	p.input = in

	return strings.TrimSpace(content), nil
}

func (p *flagParser) parsePrimitive(stopSet string) (interface{}, error) {
	content, err := p.parseNonQuotedString(stopSet)
	if err != nil {
		return nil, err
	}

	if content == "null" {
		return nil, nil
	}
	if b, ok := parseBoolValue(content); ok {
		return b, nil
	}
	if n, err := strconv.ParseUint(content, 0, 64); err == nil {
		return n, nil
	}
	if n, err := strconv.ParseInt(content, 0, 64); err == nil {
		return n, nil
	}
	if n, err := strconv.ParseFloat(content, 64); err == nil {
		return n, nil
	}

	return content, nil
}

func (p *flagParser) expectChar(c byte) error {
	if p.input == "" || p.input[0] != c {
		return fmt.Errorf("expected '%v'", string(c))
	}

	p.input = p.input[1:]
	return nil
}

func parseBoolValue(str string) (value bool, ok bool) {
	switch str {
	case "t", "T", "true", "TRUE", "True", "on", "ON":
		return true, true
	case "f", "F", "false", "FALSE", "False", "off", "OFF":
		return false, true
	}
	return false, false
}
