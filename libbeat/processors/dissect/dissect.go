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

import "fmt"

// Map  represents the keys and their values extracted with the defined tokenizer.
type Map = map[string]string

// positions represents the start and end position of the keys found in the string.
type positions []position

type position struct {
	start int
	end   int
}

// Dissector is a tokenizer based on the Dissect syntax as defined at:
// https://www.elastic.co/guide/en/logstash/current/plugins-filters-dissect.html
type Dissector struct {
	raw    string
	parser *parser
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

	return d.resolve(s, positions), nil
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
		end = dl.Next().IndexOf(s, offset)
		if end == -1 {
			return nil, fmt.Errorf(
				"could not find delimiter: `%s` in remaining: `%s`, (offset: %d)",
				dl.Delimiter(), s[offset:], offset,
			)
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
