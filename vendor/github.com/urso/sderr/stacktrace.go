// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package sderr

import (
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

type StackTrace []Frame

type Frame uintptr

type loc struct {
	file string
	line int
}

func makeStackTrace(skip int) StackTrace {
	var pcs [20]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	stack := pcs[:n]
	return *(*StackTrace)(unsafe.Pointer(&stack))
}

func getCaller(skip int) loc {
	var pcs [1]uintptr
	n := runtime.Callers(skip+2, pcs[:])
	if n == 0 {
		return loc{}
	}

	pc := pcs[0] - 1
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return loc{}
	}

	file, line := fn.FileLine(pc)
	return loc{
		file: file,
		line: line,
	}
}

func (st StackTrace) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case s.Flag('+'):
			for i, frame := range st {
				if i > 0 {
					io.WriteString(s, "\n")
				}
				frame.Format(s, verb)
			}
		case s.Flag('#'):
			fmt.Fprintf(s, "%#v", []Frame(st))
		default:
			fmt.Fprintf(s, "%s", []Frame(st))
		}
	case 's':
		fmt.Fprintf(s, "%s", []Frame(st))
	}
}

func (f Frame) pc() uintptr { return uintptr(f) - 1 }

func (f Frame) Function() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "<unknown>"
	}
	return fn.Name()
}

func (f Frame) File() string {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "<unknown>"
	}
	file, _ := fn.FileLine(f.pc())
	return file
}

func (f Frame) Line() int {
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return 0
	}
	_, line := fn.FileLine(f.pc())
	return line
}

func (f Frame) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v', 's':
		if s.Flag('+') || s.Flag('#') {
			fmt.Fprintf(s, "%s\n\t%s:%d", f.Function(), f.File(), f.Line())
		} else {
			fmt.Fprintf(s, "%s:%d", f.File(), f.Line())
		}
	}
}
