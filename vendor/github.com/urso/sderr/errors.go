// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package sderr

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/urso/diag"
)

type errWithStack struct {
	err error
}

type errValue struct {
	at    loc
	stack StackTrace
	msg   string
	ctx   *diag.Context
}

type wrappedErrValue struct {
	errValue
	cause error
}

type multiErrValue struct {
	errValue
	causes []error
}

type errorReporter interface {
	report(verbose bool) string
}

type ctxValBuf strings.Builder

func (e *errValue) At() (string, int) {
	return e.at.file, e.at.line
}

func (e *errValue) StackTrace() StackTrace {
	return e.stack
}

func (e *errValue) Context() *diag.Context {
	if e.ctx.Len() == 0 {
		return nil
	}
	return diag.NewContext(e.ctx, nil)
}

func (e *errValue) Error() string               { return errorString(e) }
func (e *errValue) Format(st fmt.State, c rune) { formatErrReporter(e, st, c) }
func (e *errValue) report(verbose bool) string {
	buf := &strings.Builder{}

	if !verbose && e.msg != "" {
		return e.msg
	}

	if verbose && e.msg != "" {
		fmt.Fprintf(buf, "%v:%v", filepath.Base(e.at.file), e.at.line)
	}

	putStr(buf, e.msg)

	if verbose && e.ctx.Len() > 0 {
		pad(buf, " ")
		buf.WriteRune('(')
		e.ctx.VisitKeyValues((*ctxValBuf)(buf))
		buf.WriteRune(')')
	}

	return buf.String()
}

func (e *wrappedErrValue) Unwrap() error {
	return e.cause
}

func (e *wrappedErrValue) Error() string               { return errorString(e) }
func (e *wrappedErrValue) Format(st fmt.State, c rune) { formatErrReporter(e, st, c) }
func (e *wrappedErrValue) report(verbose bool) string {
	const sep = ": "

	buf := &strings.Builder{}
	buf.WriteString(e.errValue.report(verbose))
	if e.cause != nil {
		putSubErr(buf, sep, e.cause, verbose)
	}
	return buf.String()
}

func (e *multiErrValue) Unwrap() error {
	if len(e.causes) == 0 {
		return nil
	}
	return e.Cause(0)
}

func (e *multiErrValue) NumCauses() int { return len(e.causes) }
func (e *multiErrValue) Cause(i int) error {
	if i < len(e.causes) {
		return e.causes[i]
	}
	return nil
}

func (e *multiErrValue) Error() string               { return e.report(false) }
func (e *multiErrValue) Format(st fmt.State, c rune) { formatErrReporter(e, st, c) }
func (e *multiErrValue) report(verbose bool) string {
	const sep = "\n\t"

	buf := &strings.Builder{}
	buf.WriteString(e.errValue.report(verbose))
	for _, cause := range e.causes {
		putSubErr(buf, sep, cause, verbose)
	}
	return buf.String()
}

func (b *ctxValBuf) OnObjStart(key string) error {
	_, err := fmt.Fprintf((*strings.Builder)(b), "%v={", key)
	return err
}

func (b *ctxValBuf) OnObjEnd() error {
	_, err := fmt.Fprint((*strings.Builder)(b), "}")
	return err
}

func (b *ctxValBuf) OnValue(key string, v diag.Value) (err error) {
	v.Reporter.Ifc(&v, func(val interface{}) {
		_, err = fmt.Fprintf((*strings.Builder)(b), "%v=%v", key, val)
	})
	return err
}

func errorString(reporter errorReporter) string {
	return reporter.report(false)
}

func formatErrReporter(reporter errorReporter, st fmt.State, c rune) {
	switch c {
	case 'v':
		if st.Flag('+') {
			io.WriteString(st, reporter.report(true))
			return
		}
		fallthrough
	case 's':
		io.WriteString(st, reporter.report(false))
	case 'q':
		io.WriteString(st, fmt.Sprintf("%q", reporter.report(false)))
	default:
		panic("unsupported format directive")
	}
}

func pad(buf *strings.Builder, pattern string) bool {
	if buf.Len() == 0 {
		return false
	}

	buf.WriteString(pattern)
	return true
}

func putStr(buf *strings.Builder, s string) bool {
	if s == "" {
		return false
	}
	pad(buf, ": ")
	buf.WriteString(s)
	return true
}

func putSubErr(b *strings.Builder, sep string, err error, verbose bool) bool {
	if err == nil {
		return false
	}

	var s string
	if verbose {
		s = fmt.Sprintf("%+v", err)
	} else {
		s = fmt.Sprintf("%v", err)
	}

	if s == "" {
		return false
	}

	pad(b, sep)

	// iterate lines
	r := strings.NewReader(s)
	scanner := bufio.NewScanner(r)
	first := true
	for scanner.Scan() {
		if !first {
			pad(b, sep)
		} else {
			first = false
		}

		b.WriteString(scanner.Text())
	}
	return true
}
