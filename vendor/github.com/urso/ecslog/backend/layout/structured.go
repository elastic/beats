// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package layout

import (
	"bytes"
	"io"

	"github.com/urso/diag"
	"github.com/urso/diag-ecs/ecs"
	"github.com/urso/ecslog/backend"
	"github.com/urso/sderr"

	structform "github.com/elastic/go-structform"
	"github.com/elastic/go-structform/cborl"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
	"github.com/elastic/go-structform/ubjson"
)

type structLayout struct {
	out         io.Writer
	buf         bytes.Buffer
	fields      *diag.Context
	makeEncoder func(io.Writer) structform.Visitor
	types       *gotype.Iterator
	typeOpts    []gotype.FoldOption
	visitor     structform.Visitor
}

type structVisitor structLayout

// errorVal is used to wrap errors, so to notify encoding callback that
// we're dealing with special error value who's context doesn't need to be
// reported.
type errorVal struct {
	err error
}

// multiErrOf is used to wrap a multierror, so to notify the encoding
// callback that we're dealing with a special error value.
// Each error in the multierror must be deal separately, creating and reporting
// it's local context.
type multiErrOf struct {
	err error
}

type multiErr struct {
	errs []error
}

func JSON(fields []diag.Field, opts ...gotype.FoldOption) Factory {
	return Structured(func(w io.Writer) structform.Visitor {
		return json.NewVisitor(w)
	}, fields, opts...)
}

func UBJSON(fields []diag.Field, opts ...gotype.FoldOption) Factory {
	return Structured(func(w io.Writer) structform.Visitor {
		return ubjson.NewVisitor(w)
	}, fields, opts...)
}

func CBOR(fields []diag.Field, opts ...gotype.FoldOption) Factory {
	return Structured(func(w io.Writer) structform.Visitor {
		return cborl.NewVisitor(w)
	}, fields, opts...)
}

func Structured(
	makeEncoder func(io.Writer) structform.Visitor,
	fields []diag.Field,
	opts ...gotype.FoldOption,
) Factory {
	return func(out io.Writer) (Layout, error) {
		logCtx := diag.NewContext(nil, nil)
		logCtx.AddFields(fields...)

		l := &structLayout{
			out:         out,
			fields:      logCtx,
			makeEncoder: makeEncoder,
			typeOpts:    opts,
		}
		l.reset()
		return l, nil
	}
}

func (l *structLayout) reset() {
	l.buf.Reset()
	visitor := l.makeEncoder(&l.buf)
	l.types, _ = gotype.NewIterator(visitor, l.typeOpts...)
	l.visitor = visitor
}

func (l *structLayout) UseContext() bool { return true }

func (l *structLayout) Log(msg backend.Message) {
	var userCtx, stdCtx *diag.Context

	if msg.Context.Len() > 0 {
		userCtx = msg.Context.User()
		stdCtx = msg.Context.Standardized()
	}

	file := msg.Caller.File()

	ctx := diag.NewContext(stdCtx, nil)
	ctx.AddFields([]diag.Field{
		ecs.Log.Level(msg.Level.String()),

		ecs.Log.Origin.File.Name(file),
		ecs.Log.Origin.File.Line(msg.Caller.Line()),

		ecs.Message(msg.Message),
	}...)
	if msg.Name != "" {
		ctx.AddField(ecs.Log.Logger(msg.Name))
	}

	if userCtx.Len() > 0 {
		ctx.AddField(diag.Any("fields", &userCtx))
	}

	// Add error values to the context. So to guarantee an error value is not
	// missed we use fully qualified names here.
	switch len(msg.Causes) {
	case 0:
		break
	case 1:
		cause := msg.Causes[0]
		if errCtx := buildErrCtx(cause); errCtx.Len() > 0 {
			ctx.AddField(diag.Any("error.ctx", &errCtx))
		}
		ctx.AddField(diag.String("error.message", cause.Error()))

		if file, line := sderr.At(cause); file != "" {
			ctx.AddField(diag.String("error.at.file", file))
			ctx.AddField(diag.Int("error.at.line", line))
		}

		n := sderr.NumCauses(cause)
		switch n {
		case 0:
			// nothing
		case 1:
			ctx.AddField(diag.Any("error.cause", errorVal{sderr.Unwrap(cause)}))

		default:
			ctx.AddField(diag.Any("error.causes", multiErrOf{cause}))
		}

	default:
		ctx.AddField(diag.Any("error.causes", multiErr{msg.Causes}))
	}

	// link predefined fields
	if l.fields.Len() > 0 {
		ctx = diag.NewContext(l.fields, ctx)
	}

	v := (*structVisitor)(l)
	if err := v.Process(ctx); err != nil {
		l.reset()
	} else {
		l.out.Write(l.buf.Bytes())
		l.buf.Reset()
	}
}

func (v *structVisitor) Process(ctx *diag.Context) error {
	if err := v.Begin(); err != nil {
		return err
	}
	if err := ctx.VisitStructured(v); err != nil {
		return err
	}
	return v.End()
}

func (v *structVisitor) Begin() error { return v.visitor.OnObjectStart(-1, structform.AnyType) }
func (v *structVisitor) End() error   { return v.visitor.OnObjectFinished() }

func (v structVisitor) OnObjStart(key string) error {
	if err := v.visitor.OnKey(key); err != nil {
		return err
	}
	return v.visitor.OnObjectStart(-1, structform.AnyType)
}

func (v structVisitor) OnObjEnd() error {
	return v.visitor.OnObjectFinished()
}

func (v structVisitor) OnValue(key string, val diag.Value) error {
	var err error

	if err = v.visitor.OnKey(key); err != nil {
		return err
	}

	val.Reporter.Ifc(&val, func(ifc interface{}) {
		switch val := ifc.(type) {
		case *diag.Context:
			if err = v.Begin(); err != nil {
				return
			}
			if err = val.VisitStructured(v); err != nil {
				return
			}
			err = v.End()

		case errorVal: // error cause
			err = v.OnErrorValue(val.err, false)

		case multiErrOf:
			err = v.OnMultiErrValueIter(val.err)

		case multiErr:
			err = v.OnMultiErr(val.errs)

		default:
			err = v.types.Fold(ifc)
		}
	})

	return err
}

func (v structVisitor) OnErrorValue(err error, withCtx bool) error {
	if err := v.Begin(); err != nil {
		return err
	}

	if file, line := sderr.At(err); file != "" {
		if err := v.visitor.OnKey("at"); err != nil {
			return err
		}
		if err := v.Begin(); err != nil {
			return err
		}
		if err := v.visitor.OnKey("file"); err != nil {
			return err
		}
		if err := v.visitor.OnString(file); err != nil {
			return err
		}
		if err := v.visitor.OnKey("line"); err != nil {
			return err
		}
		if err := v.visitor.OnInt(line); err != nil {
			return err
		}
		if err := v.End(); err != nil {
			return err
		}
	}

	if withCtx {
		ctx := buildErrCtx(err)
		if ctx.Len() > 0 {
			if err := v.visitor.OnKey("ctx"); err != nil {
				return err
			}
			if err := v.Begin(); err != nil {
				return err
			}
			if err := ctx.VisitStructured(v); err != nil {
				return err
			}
			if err := v.End(); err != nil {
				return err
			}
		}
	}

	n := sderr.NumCauses(err)
	switch n {
	case 0:
		// nothing to do

	case 1:
		// add cause
		cause := sderr.Cause(err, 0)
		if cause != nil {
			if err := v.OnValue("cause", diag.ValAny(errorVal{cause})); err != nil {
				return err
			}
		}

	default:
		if err := v.OnValue("causes", diag.ValAny(multiErrOf{err})); err != nil {
			return err
		}

	}

	if err := v.visitor.OnKey("message"); err != nil {
		return err
	}
	if err := v.visitor.OnString(err.Error()); err != nil {
		return err
	}

	return v.End()
}

func (v structVisitor) OnMultiErrValueIter(parent error) error {
	if err := v.visitor.OnArrayStart(-1, structform.AnyType); err != nil {
		return err
	}

	n := sderr.NumCauses(parent)
	for i := 0; i < n; i++ {
		cause := sderr.Cause(parent, i)
		if cause != nil {
			if err := v.OnErrorValue(cause, true); err != nil {
				return err
			}
		}
	}

	return v.visitor.OnArrayFinished()
}

func (v structVisitor) OnMultiErr(errs []error) error {
	if err := v.visitor.OnArrayStart(-1, structform.AnyType); err != nil {
		return err
	}

	for _, err := range errs {
		if err != nil {
			if err := v.OnErrorValue(err, true); err != nil {
				return err
			}
		}
	}

	return v.visitor.OnArrayFinished()
}

func buildErrCtx(err error) (errCtx *diag.Context) {
	var linkedCtx *diag.Context

	causeCtx := sderr.Context(err)
	if causeCtx.Len() > 0 {
		linkedCtx = linkLinearErrCtx(causeCtx, err)
	} else {
		linkedCtx = linkLinearErrCtx(linkedCtx, err)
	}

	stdCtx := linkedCtx.Standardized()
	errCtx = diag.NewContext(stdCtx, nil)

	if userCtx := linkedCtx.User(); userCtx.Len() > 0 {
		errCtx.AddField(diag.Any("fields", &userCtx))
	}

	return errCtx
}

// linkLinearErrCtx links all error context in a linear chain. Stops if a
// multierror is discovered.
func linkLinearErrCtx(ctx *diag.Context, err error) *diag.Context {
	for err != nil {
		n := sderr.NumCauses(err)
		if n != 1 {
			return ctx
		}

		cause := sderr.Unwrap(err)
		causeCtx := sderr.Context(cause)
		if causeCtx.Len() > 0 {
			ctx = diag.NewContext(ctx, causeCtx)
		}

		err = cause
	}
	return ctx
}
