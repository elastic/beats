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

package integration

import (
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
)

// NewCounter returns an output counter for the given substring.
//
// If given multiple strings, they get checked in order.
// The first substring must match first, then second, etc.
//
// Only when all substrings match in order the counter gets incremented.
func NewCounter(out *atomic.Int64, strs ...string) OutputInspector {
	return &counter{
		strs:    strs,
		out:     out,
		matched: 0,
	}
}

type counter struct {
	strs    []string
	matched int
	out     *atomic.Int64
}

func (c *counter) Inspect(line string) {
	str := c.strs[c.matched]
	// trying to match the current substring in order
	if !strings.Contains(line, str) {
		return
	}

	// move to the next substring
	c.matched++

	// if we reached the end of the list, we reset and count the entire match
	if c.matched == len(c.strs) {
		c.matched = 0
		c.out.Add(1)
	}
}

func (c *counter) String() string {
	var strs strings.Builder

	for i, s := range c.strs {
		if i != 0 {
			strs.WriteString(" -> ")
		}
		strs.WriteString("'" + s + "'")
	}

	return fmt.Sprintf("counter(%s)", strs.String())
}

// NewRegexpCounter returns an output counter fo the given regular expression.
//
// Every future output line produced by the Beat will be matched
// against the given regular expression and counted.
//
// If given multiple expressions, they get checked in order.
// The first expression must match first, then second, etc.
//
// Only when all expressions match in order the counter gets incremented.
func NewRegexpCounter(out *atomic.Int64, exprs ...*regexp.Regexp) OutputInspector {
	return &regexpCounter{
		exprs:   exprs,
		out:     out,
		matched: 0,
	}
}

type regexpCounter struct {
	exprs   []*regexp.Regexp
	matched int
	out     *atomic.Int64
}

func (c *regexpCounter) Inspect(line string) {
	expr := c.exprs[c.matched]
	// trying to match the current expression in order
	if !expr.MatchString(line) {
		return
	}

	// move to the next expression
	c.matched++

	// if we reached the end of the list, we reset and count the entire match
	if c.matched == len(c.exprs) {
		c.matched = 0
		c.out.Add(1)
	}
}

func (c *regexpCounter) String() string {
	var exprs strings.Builder

	for i, e := range c.exprs {
		if i != 0 {
			exprs.WriteString(" -> ")
		}
		exprs.WriteString("regexp(" + e.String() + ")")
	}

	return fmt.Sprintf("counter(%s)", exprs.String())
}
