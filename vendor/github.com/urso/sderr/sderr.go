// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package sderr

import (
	"context"
	"errors"

	"github.com/urso/diag"
)

var emptyBuilder = &Builder{}

// New returns an error that formats as the given text. Each call to New
// returns a distinct error value even if the text is identical.
func New(msg string) error {
	return errors.New(msg)
}

func With(fields ...interface{}) *Builder {
	return emptyBuilder.With(fields...)
}

func WithStack() *Builder {
	return emptyBuilder.WithStack()
}

func WithDiagnosticContext(ctx *diag.Context) *Builder {
	return emptyBuilder.WithDiagnosticContext(ctx)
}

func WithDiagnotics(ctx context.Context) *Builder {
	return emptyBuilder.WithDiagnotics(ctx)
}

func Errf(msg string, vs ...interface{}) error {
	return emptyBuilder.doErrf(1, msg, vs)
}

func Wrap(cause error, msg string, vs ...interface{}) error {
	return emptyBuilder.doWrap(1, cause, msg, vs)
}

func WrapAll(causes []error, msg string, vs ...interface{}) error {
	return emptyBuilder.doWrapAll(1, causes, msg, vs)
}
