// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package diag

import "context"

type key int

// diagContextKey is the key for diag.Context values in context.Contexts.
// It is unexported; clients should use DiagnosticsFrom, and NewDiagnostics.
var diagContextKey key

// NewDiagnostics adds a diagnostics context to a context.Context value.
// The old diagnostic context will be shadowed if the context.Context already
// contains a diagnostics context.
func NewDiagnostics(ctx context.Context, dc *Context) context.Context {
	return context.WithValue(ctx, diagContextKey, dc)
}

// DiagnosticsFrom extracts a diagnostic context from context.Context.
func DiagnosticsFrom(ctx context.Context) (*Context, bool) {
	tmp := ctx.Value(diagContextKey)
	if tmp == nil {
		return nil, false
	}

	dc, ok := tmp.(*Context)
	return dc, ok
}

// PushFields adds a new diagnostics context with the given set of fields
// to a context.Context value. The new diagnostic context references the
// existing diagnostic context, if one exists (fields will be combined).
func PushFields(ctx context.Context, fields ...Field) context.Context {
	ctx, dc := extendDiagnostics(ctx)
	dc.AddFields(fields...)
	return ctx
}

// PushDiagnostics adds a new diagnostics context with the given fields to a
// context.Context value (see (*Context).AddAll). The new diagnostic context
// references the existing diagnostic context, if one exists (fields will be
// combined).
func PushDiagnostics(ctx context.Context, args ...interface{}) context.Context {
	ctx, dc := extendDiagnostics(ctx)
	dc.AddAll(args...)
	return ctx
}

func extendDiagnostics(ctx context.Context) (context.Context, *Context) {
	dc, ok := DiagnosticsFrom(ctx)
	if ok {
		dc = NewContext(dc, nil)
	} else {
		dc = NewContext(nil, nil)
	}

	return NewDiagnostics(ctx, dc), dc
}
