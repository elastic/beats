// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

// Package ctxfmt provides formatters similar to the fmt package, that support
// capturing named fields from within the format string. The functions provided
// here shoud be used by other packages wanting to provide custom APIs
// on top of the diag package and wishing to provide support for printf style
// pritning, while capturing field defintions into a diag.Context.
// All functions provided accept an additionl callback, and return a list of
// arguments that have not been consumed by the format string. The callback
// will be called if the format string contains a field-spec or an error value
// is passed to the argument list. All formatters also return
//
//
// A format string can contain formatting verbs similar to the fmt package, or
// field-specs.
//
// Verbs accept almost all flags, width, and precision arguments as are present in the fmt package.
// Index selection or '*' is not supported.
//
// A field-spec has the form `%{[+#@]<field-name>[:<format-verb>]}`.
// The field name is mandatory. If no <format-verb> is given, then the value
// will be printed using `v` as verb. The prefix modifiers '+', '#', and
// '@'(=alias for '#') change how the argument will be printed, similar to
// normal verb flags.
// More complex formatting directives can be configured after the `:`. The <format-verb> uses the same syntax.
//
// For example:
//
//    Printf(cb, "hello %v", "world")
//
// will just print hello world. But:
//
//    Printf(cb, "hello %{who}", "world")
//
// will call your callback with like this: `cb("who", 0, "world")`
//
// We can print an padded integer with a text with of 5 digits like this:
//
//    Printf(cb, "%{value:05d}", 23)
//
// This will print '00023' thanks to the '0' flag and call the provided callback.
//
// Named and anonymous formatting using can be freely mixed. The callback will only
// be called if a named field or error value is encountered.
//
// The printf-style functions in ctxfmt all respect the fmt.Stringer,
// fmt.GoStringer, and fmt.Formatter interfaces.
package ctxfmt

import (
	"io"
	"os"
	"strings"
)

type CB func(key string, idx int, val interface{})

// Printf formats according to the format specifier and writes to stdout.
// It returns the unprocessed arguments.
func Printf(cb CB, msg string, vs ...interface{}) (rest []interface{}, n int, err error) {
	return Fprintf(os.Stdout, cb, msg, vs...)
}

// Sprintf formats according to the format specifier and returns the resulting
// string and the list of unprocessed arguments.
func Sprintf(cb CB, msg string, vs ...interface{}) (string, []interface{}) {
	var buf strings.Builder
	rest, _, _ := Fprintf(&buf, cb, msg, vs...)
	return buf.String(), rest
}

// Fprintf formats according to the format specifier and writes to w.
// It returns the unprocessed arguments.
func Fprintf(w io.Writer, cb CB, msg string, vs ...interface{}) (rest []interface{}, n int, err error) {
	printer := &printer{To: w}
	in := &interpreter{
		cb:   cb,
		p:    printer,
		args: argstate{args: vs},
	}
	parser := &parser{handler: in}
	parser.parse(msg)

	used := in.args.idx
	if used >= len(vs) {
		return nil, printer.written, printer.err
	}

	// collect errors from extra variables
	rest = vs[used:]
	for i := range rest {
		if isErrorValue(rest[i]) {
			cb("", used+i, rest[i])
		}
	}
	return rest, printer.written, printer.err
}
