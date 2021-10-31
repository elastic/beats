// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package sderr

import (
	"context"
	"fmt"

	"github.com/urso/diag"
	"github.com/urso/diag/ctxfmt"
)

type Builder struct {
	ctx       *diag.Context
	withStack bool
}

func (b *Builder) With(fields ...interface{}) *Builder {
	ctx := diag.NewContext(b.ctx, nil)
	ctx.AddAll(fields...)
	return &Builder{ctx: ctx, withStack: b.withStack}
}

func (b *Builder) WithStack() *Builder {
	var ctx *diag.Context
	if b.ctx.Len() > 0 {
		ctx = diag.NewContext(b.ctx, nil)
	}
	return &Builder{ctx: ctx, withStack: true}
}

func (b *Builder) WithDiagnosticContext(ctx *diag.Context) *Builder {
	merged := diag.NewContext(b.ctx, ctx)
	return &Builder{ctx: diag.NewContext(merged, nil), withStack: b.withStack}
}

func (b *Builder) WithDiagnotics(ctx context.Context) *Builder {
	dc, _ := diag.DiagnosticsFrom(ctx)
	return b.WithDiagnosticContext(dc)
}

func (b *Builder) Errf(msg string, vs ...interface{}) error {
	return b.doErrf(1, msg, vs)
}

func (b *Builder) doErrf(skip int, msg string, vs []interface{}) error {
	val, causes := b.makeErrValue(skip+1, msg, vs)
	switch len(causes) {
	case 0:
		return &val
	case 1:
		return &wrappedErrValue{errValue: val, cause: causes[0]}
	default:
		return &multiErrValue{errValue: val, causes: causes}
	}
}

func (b *Builder) Wrap(cause error, msg string, vs ...interface{}) error {
	return b.doWrap(1, cause, msg, vs)
}

func (b *Builder) doWrap(skip int, cause error, msg string, vs []interface{}) error {
	val, extra := b.makeErrValue(skip+1, msg, vs)
	if len(extra) > 0 {
		if cause != nil {
			extra = append(extra, cause)
		}

		if len(extra) == 1 {
			return &wrappedErrValue{errValue: val, cause: extra[0]}
		}
		return &multiErrValue{errValue: val, causes: extra}
	}

	if cause == nil {
		return &val
	}

	return &wrappedErrValue{errValue: val, cause: cause}
}

func (b *Builder) WrapAll(causes []error, msg string, vs ...interface{}) error {
	return b.doWrapAll(1, causes, msg, vs)
}

func (b *Builder) doWrapAll(skip int, causes []error, msg string, vs []interface{}) error {
	if len(causes) == 0 {
		return nil
	}

	val, extra := b.makeErrValue(skip+1, msg, vs)
	if len(extra) > 0 {
		causes = append(extra, causes...)
	}

	return &multiErrValue{errValue: val, causes: causes}
}

func (b *Builder) makeErrValue(skip int, msg string, vs []interface{}) (errValue, []error) {
	var ctx *diag.Context
	var causes []error

	errorMessage, _ := ctxfmt.Sprintf(func(key string, idx int, val interface{}) {
		if ctx == nil {
			ctx = diag.NewContext(b.ctx, nil)
		}

		if field, ok := (val).(diag.Field); ok {
			if key != "" {
				ctx.Add(fmt.Sprintf("%v.%v", key, field.Key), field.Value)
			} else {
				ctx.AddField(field)
			}
			return
		}

		switch v := val.(type) {
		case diag.Value:
			ctx.Add(ensureKey(key, idx), v)
		case error:
			causes = append(causes, v)
			if key != "" {
				ctx.AddField(diag.String(key, v.Error()))
			}
		default:
			ctx.AddField(diag.Any(ensureKey(key, idx), val))
		}

	}, msg, vs...)

	if ctx == nil {
		ctx = b.ctx
	}

	var stack StackTrace
	if b.withStack {
		stack = makeStackTrace(skip + 1)
	}
	return errValue{at: getCaller(skip + 1), msg: errorMessage, ctx: ctx, stack: stack}, causes
}

func ensureKey(key string, idx int) string {
	if key == "" {
		return fmt.Sprintf("%v", idx)
	}
	return key
}
