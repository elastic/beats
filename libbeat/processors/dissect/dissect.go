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

package dissect

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/pkg/errors"
)

// Map  represents the keys and their values extracted with the defined tokenizer.
type Map = map[string]string
type MapConverted = map[string]interface{}

// positions represents the start and end position of the keys found in the string.
type positions []position

type position struct {
	start int
	end   int
}

// Dissector is a tokenizer based on the Dissect syntax as defined at:
// https://www.elastic.co/guide/en/logstash/current/plugins-filters-dissect.html
type Dissector struct {
	raw     string
	parser  *parser
	trimmer trimmer
}

// Dissect takes the raw string and will use the defined tokenizer to return a map with the
// extracted keys and their values.
//
// Dissect uses a 3 steps process:
// - Find the key positions
// - Extract and resolve the keys (append / indirect)
// - Ignore namedSkipField
func (d *Dissector) Dissect(s string) (Map, error) {
	if len(s) == 0 {
		return nil, errEmpty
	}

	positions, err := d.extract(s)
	if err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		return nil, errParsingFailure
	}
	if d.trimmer != nil {
		for idx, pos := range positions {
			pos.start, pos.end = d.trimmer.Trim(s, pos.start, pos.end)
			positions[idx] = pos
		}
	}
	return d.resolve(s, positions), nil
}

func (d *Dissector) DissectConvert(s string) (MapConverted, error) {
	if len(s) == 0 {
		return nil, errEmpty
	}

	positions, err := d.extract(s)
	if err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		return nil, errParsingFailure
	}

	return d.resolveConvert(s, positions), nil
}

// Raw returns the raw tokenizer used to generate the actual parser.
func (d *Dissector) Raw() string {
	return d.raw
}

// extract will navigate through the delimiters and will save the ending and starting position
// of the keys. After we will resolve the positions with the required fields and do the reordering.
func (d *Dissector) extract(s string) (positions, error) {
	positions := make([]position, len(d.parser.fields))
	var i, start, lookahead, end int

	// Position on the first delimiter, we assume a hard match on the first delimiter.
	// Previous version of dissect was doing a lookahead in the string until it can find the delimiter,
	// LS and Beats now have the same behavior and this is consistent with the principle of least
	// surprise.
	dl := d.parser.delimiters[0]
	offset := dl.IndexOf(s, 0)
	if offset == -1 || offset != 0 {
		return nil, fmt.Errorf(
			"could not find beginning delimiter: `%s` in remaining: `%s`, (offset: %d)",
			dl.Delimiter(), s, 0,
		)
	}
	offset += dl.Len()

	// move through all the other delimiters, until we have consumed all of them.
	for dl.Next() != nil {
		start = offset

		// corresponding field of the delimiter
		field := d.parser.fields[d.parser.fieldsIdMap[i]]

		// for fixed-length field, just step the same size of its length
		if field.IsFixedLength() {
			end = offset + field.Length()
			if end > len(s) {
				return nil, fmt.Errorf(
					"field length is grater than string length: remaining: `%s`, (offset: %d), field: %s",
					s[offset:], offset, field,
				)
			}
		} else {
			end = dl.Next().IndexOf(s, offset)
			if end == -1 {
				return nil, fmt.Errorf(
					"could not find delimiter: `%s` in remaining: `%s`, (offset: %d)",
					dl.Delimiter(), s[offset:], offset,
				)
			}
		}

		offset = end

		// Greedy consumes keys defined with padding.
		// Keys are defined with `->` suffix.
		if dl.IsGreedy() {
			for {
				lookahead = dl.Next().IndexOf(s, offset+1)
				if lookahead != offset+1 {
					break
				} else {
					offset = lookahead
				}
			}
		}

		positions[i] = position{start: start, end: end}
		offset += dl.Next().Len()
		i++
		dl = dl.Next()
	}

	field := d.parser.fields[d.parser.fieldsIdMap[i]]

	if field.IsFixedLength() && offset+field.Length() != len(s) {
		return nil, fmt.Errorf("last fixed length key `%s` (length: %d) does not fit into remaining: `%s`, (offset: %d)",
			field, field.Length(), s, offset,
		)
	}
	// If we have remaining contents and have not captured all the requested fields
	if offset < len(s) && i < len(d.parser.fields) {
		positions[i] = position{start: offset, end: len(s)}
	}
	return positions, nil
}

// resolve takes the raw string and the extracted positions and apply fields syntax.
func (d *Dissector) resolve(s string, p positions) Map {
	m := make(Map, len(p))
	for _, f := range d.parser.fields {
		pos := p[f.ID()]
		f.Apply(s[pos.start:pos.end], m)
	}

	for _, f := range d.parser.referenceFields {
		delete(m, f.Key())
	}
	return m
}

func (d *Dissector) resolveConvert(s string, p positions) MapConverted {
	lookup := make(mapstr.M, len(p))
	m := make(Map, len(p))
	mc := make(MapConverted, len(p))
	for _, f := range d.parser.fields {
		pos := p[f.ID()]
		f.Apply(s[pos.start:pos.end], m) // using map[string]string to avoid another set of apply methods
		if !f.IsSaveable() {
			lookup[f.Key()] = s[pos.start:pos.end]
		} else {
			key := f.Key()
			if k, ok := lookup[f.Key()]; ok {
				key = k.(string)
			}
			v, _ := m[key]
			if f.DataType() != "" {
				mc[key] = convertData(f.DataType(), v)
			} else {
				mc[key] = v
			}
		}
	}

	for _, f := range d.parser.referenceFields {
		delete(mc, f.Key())
	}
	return mc
}

// New creates a new Dissector from a tokenized string.
func New(tokenizer string) (*Dissector, error) {
	p, err := newParser(tokenizer)
	if err != nil {
		return nil, err
	}

	if err := validate(p); err != nil {
		return nil, err
	}

	return &Dissector{parser: p, raw: tokenizer}, nil
}

// strToInt is a helper to interpret a string as either base 10 or base 16.
func strToInt(s string, bitSize int) (int64, error) {
	base := 10
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		// strconv.ParseInt will accept the '0x' or '0X` prefix only when base is 0.
		base = 0
	}
	return strconv.ParseInt(s, base, bitSize)
}

func transformType(typ dataType, value string) (interface{}, error) {
	value = strings.TrimRight(value, " ")
	switch typ {
	case String:
		return value, nil
	case Long:
		return strToInt(value, 64)
	case Integer:
		i, err := strToInt(value, 32)
		return int32(i), err
	case Float:
		f, err := strconv.ParseFloat(value, 32)
		return float32(f), err
	case Double:
		d, err := strconv.ParseFloat(value, 64)
		return float64(d), err
	case Boolean:
		return strconv.ParseBool(value)
	case IP:
		if net.ParseIP(value) != nil {
			return value, nil
		}
		return "", errors.New("value is not a valid IP address")
	default:
		return value, nil
	}
}

func convertData(typ string, b string) interface{} {
	if dt, ok := dataTypeNames[typ]; ok {
		value, err := transformType(dt, b)
		if err == nil {
			return value
		}
	}
	return b
}
