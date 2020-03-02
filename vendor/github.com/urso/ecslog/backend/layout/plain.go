// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package layout

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/urso/diag"
	"github.com/urso/sderr"

	"github.com/urso/ecslog/backend"
)

type textLayout struct {
	out     io.Writer
	buf     bytes.Buffer
	withCtx bool
}

type textCtxPrinter struct {
	buf *bytes.Buffer
	n   int
}

// maximum logger buffer size to keep in between calls
const persistentTextBufferSize = 512

func Text(withCtx bool) Factory {
	return func(out io.Writer) (Layout, error) {
		return &textLayout{
			out:     out,
			withCtx: withCtx,
		}, nil
	}
}

func (l *textLayout) UseContext() bool {
	return l.withCtx
}

func (l *textLayout) Log(msg backend.Message) {
	defer func() {
		if l.buf.Len()+l.buf.Cap() > persistentTextBufferSize {
			l.buf = bytes.Buffer{}
		} else {
			l.buf.Reset()
		}
	}()

	ts := time.Now()

	l.buf.WriteString(ts.Format(time.RFC3339))
	l.buf.WriteByte(' ')
	l.buf.WriteString(l.level(msg.Level))
	l.buf.WriteByte('\t')
	if msg.Name != "" {
		fmt.Fprintf(&l.buf, "'%v' - ", msg.Name)
	}

	caller := msg.Caller
	fmt.Fprintf(&l.buf, "%v:%d", filepath.Base(caller.File()), caller.Line())
	l.buf.WriteByte('\t')
	l.buf.WriteString(msg.Message)

	msg.Context.VisitKeyValues(&textCtxPrinter{buf: &l.buf})
	l.buf.WriteRune('\n')

	// write errors
	switch len(msg.Causes) {
	case 0:
		// do nothing

	case 1:
		if ioErr := l.OnErrorValue(msg.Causes[0], "\t"); ioErr != nil {
			return
		}

	case 2:
		written := 0
		l.buf.WriteString("\tcaused by:\n")
		for _, err := range msg.Causes {
			if err == nil {
				continue
			}

			if written != 0 {
				l.buf.WriteString("\tand\n")
			}

			written++
			if ioErr := l.OnErrorValue(err, "\t    "); ioErr != nil {
				return
			}
		}
	}

	l.out.Write(l.buf.Bytes())
}

func (l *textLayout) OnErrorValue(err error, indent string) error {
	l.buf.WriteString(indent)

	if file, line := sderr.At(err); file != "" {
		fmt.Fprintf(&l.buf, "%v:%v\t", filepath.Base(file), line)
	}

	l.buf.WriteString(err.Error())

	if l.withCtx {
		if ctx := sderr.Context(err); ctx.Len() > 0 {
			ctx.VisitKeyValues(&textCtxPrinter{buf: &l.buf})
		}
	}

	if _, ioErr := l.buf.WriteRune('\n'); ioErr != nil {
		return ioErr
	}

	n := sderr.NumCauses(err)
	switch n {
	case 0:
		// do nothing
	case 1:
		cause := sderr.Unwrap(err)
		if cause != nil {
			return l.OnErrorValue(cause, indent)
		}
	default:
		causeIndent := indent + "    "
		written := 0
		fmt.Fprintf(&l.buf, "%vmulti-error caused by:\n", indent)
		for i := 0; i < n; i++ {
			cause := sderr.Cause(err, i)
			if cause != nil {
				if written != 0 {
					fmt.Fprintf(&l.buf, "%vand\n", indent)
				}

				written++
				if err := l.OnErrorValue(cause, causeIndent); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (_ *textLayout) level(lvl backend.Level) string {
	switch lvl {
	case backend.Trace:
		return "TRACE"
	case backend.Debug:
		return "DEBUG"
	case backend.Info:
		return "INFO"
	case backend.Error:
		return "ERROR"
	default:
		return fmt.Sprintf("<%v>", lvl)
	}
}

func (p *textCtxPrinter) OnObjStart(key string) error {
	if err := p.onKey(key); err != nil {
		return err
	}
	_, err := p.buf.WriteRune('{')
	return err
}

func (p *textCtxPrinter) OnObjEnd() error {
	_, err := p.buf.WriteRune('}')
	return err
}

func (p *textCtxPrinter) OnValue(key string, v diag.Value) (err error) {
	p.onKey(key)
	v.Reporter.Ifc(&v, func(value interface{}) {
		switch v := value.(type) {
		case *diag.Context:
			p.buf.WriteRune('{')
			err = v.VisitKeyValues(p)
			p.buf.WriteRune('}')
		case string, []byte:
			fmt.Fprintf(p.buf, "%q", v)
		default:
			fmt.Fprintf(p.buf, "%v", v)
		}
	})

	return err
}

func (p *textCtxPrinter) onKey(key string) error {
	if p.n > 0 {
		p.buf.WriteRune(' ')
	} else {
		p.buf.WriteString("\t| ")
	}
	p.buf.WriteString(key)
	p.buf.WriteRune('=')
	p.n++
	return nil
}
