package gotype

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	structform "github.com/elastic/go-structform"
	stunsafe "github.com/elastic/go-structform/internal/unsafe"
)

type userFoldFn func(unsafe.Pointer, structform.ExtVisitor) error

func makeUserFoldFn(fn reflect.Value) (userFoldFn, error) {
	t := fn.Type()

	if fn.Kind() != reflect.Func {
		return nil, errors.New("function type required")
	}

	if t.NumIn() != 2 {
		return nil, fmt.Errorf("function '%v' must accept 2 arguments", t.Name())
	}
	if t.NumOut() != 1 || t.Out(0) != tError {
		return nil, fmt.Errorf("function '%v' does not return errors", t.Name())
	}

	ta0 := t.In(0)
	if ta0.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("first argument in function '%v' must be a pointer", t.Name())
	}

	ta1 := t.In(1)
	if ta1 != tExtVisitor {
		return nil, fmt.Errorf("second arument in function '%v' must be structform.ExtVisitor", t.Name())
	}

	fptr := *((*userFoldFn)(stunsafe.UnsafeFnPtr(fn)))
	return fptr, nil
}

func liftUserPtrFn(f userFoldFn) reFoldFn {
	return func(c *foldContext, v reflect.Value) error {
		if v.IsNil() {
			return f(nil, c.visitor)
		}
		return f(stunsafe.ReflValuePtr(v.Elem()), c.visitor)
	}
}

func liftUserValueFn(f userFoldFn) reFoldFn {
	return func(c *foldContext, v reflect.Value) error {
		return f(stunsafe.ReflValuePtr(v), c.visitor)
	}
}
