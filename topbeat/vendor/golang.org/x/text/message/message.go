// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package message implements formatted I/O for localized strings with functions
// analogous to the fmt's print functions.
//
// Under construction. See https://golang.org/design/text/12750-localization
// and its corresponding proposal issue https://golang.org/issues/12750.
package message

import (
	"fmt"
	"io"

	"golang.org/x/text/internal/format"
	"golang.org/x/text/language"
)

// A Printer implements language-specific formatted I/O analogous to the fmt
// package. Only one goroutine may use a Printer at the same time.
type Printer struct {
	tag language.Tag

	// NOTE: limiting one goroutine per Printer allows for many optimizations
	// and simplifications. We can consider removing this restriction down the
	// road if it the benefits do not seem to outweigh the disadvantages.
}

// NewPrinter returns a Printer that formats messages tailored to language t.
func NewPrinter(t language.Tag) *Printer {
	return &Printer{tag: t}
}

// Sprint is like fmt.Sprint, but using language-specific formatting.
func (p *Printer) Sprint(a ...interface{}) string {
	return fmt.Sprint(p.bindArgs(a)...)
}

// Fprint is like fmt.Fprint, but using language-specific formatting.
func (p *Printer) Fprint(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprint(w, p.bindArgs(a)...)
}

// Print is like fmt.Print, but using language-specific formatting.
func (p *Printer) Print(a ...interface{}) (n int, err error) {
	return fmt.Print(p.bindArgs(a)...)
}

// bindArgs wraps arguments with implementation of fmt.Formatter, if needed.
func (p *Printer) bindArgs(a []interface{}) []interface{} {
	out := make([]interface{}, len(a))
	for i, x := range a {
		switch v := x.(type) {
		case fmt.Formatter:
			// Wrap the value with a Formatter that augments the State with
			// language-specific attributes.
			out[i] = &value{v, p}

			// NOTE: as we use fmt.Formatter, we can't distinguish between
			// regular and localized formatters, so we always need to wrap it.

			// TODO: handle
			// - numbers
			// - lists
			// - time?
		default:
			out[i] = x
		}
	}
	return out
}

// state implements "golang.org/x/text/internal/format".State.
type state struct {
	fmt.State
	p *Printer
}

func (s *state) Language() language.Tag { return s.p.tag }

var _ format.State = &state{}

type value struct {
	x fmt.Formatter
	p *Printer
}

func (v *value) Format(s fmt.State, verb rune) {
	v.x.Format(&state{s, v.p}, verb)
}
