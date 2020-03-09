// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0

package sderr

import (
	"reflect"

	"github.com/urso/diag"
)

// errCheckIs is an optional interface that allows users and errors to
// customize the behavior of `Is`, such the custom error types can be matched,
// or private fields can be checked.
type errCheckIs interface {
	Is(error) bool
}

// errConv is used by `As` to allow for some customaized conversation from err
// to target in As.
type errConv interface {
	As(interface{}) bool
}

// causer interface allows us to iterate errors with unique causes.
// The interface is compatible to github.com/pkg/errors.
type causer interface {
	Cause() error
}

type unwrapper interface {
	Unwrap() error
}

// multiCauser interface allows us to efficiently iterate a set of causes
// leading to the current error.
// The interface is compatible to github.com/urso/ecslog/errx
type multiCauser interface {
	NumCauses() int
	Cause(i int) error
}

// wrappedError is compatible to github.com/hashicorp/go-multierror
type wrappedError interface {
	WrappedErrors() []error
}

var tError = reflect.TypeOf((*error)(nil)).Elem()

// At returns the file name and line the error originated from (if present)
func At(err error) (string, int) {
	if pe, ok := err.(interface{ At() (string, int) }); ok {
		return pe.At()
	}
	return "", 0
}

// Trace returns the stack trace, if the error value contains one.
func Trace(err error) StackTrace {
	if se, ok := err.(interface{ StackTrace() StackTrace }); ok {
		return se.StackTrace()
	}
	return nil
}

// Context returns the errors diagnostic context, if the direct error value has a context.
func Context(err error) *diag.Context {
	if ce, ok := err.(interface{ Context() *diag.Context }); ok {
		return ce.Context()
	}
	return nil
}

// Is walks the complete error tree, trying to find an error that matches
// target.
//
// An error is matched with target, if the error is equal to the target,
// if error implements Is(error) bool that returns true for target, or
// target implements Is(error) bool such that Is(error) returne true.
//
// Use `Find` to return the actual error value that would match.
func Is(err, target error) bool {
	if err == nil {
		return err == target
	}

	return Find(err, target) != nil
}

// IsIf walkt the complete error try, trying to match an error
// with the given predicate.
//
// Use `FindIf` to return the actual error value that would match.
func IsIf(err error, pred func(error) bool) bool {
	if err == nil {
		return pred(err)
	}

	return FindIf(err, pred) != nil
}

// As finds the first error in the error tree that matches target, and if so, sets
// target to that error value and returns true.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// As will panic if target is not a non-nil pointer to either a type that implements
// error, or to any interface type. As returns false if err is nil.
func As(err error, target interface{}) bool {
	if target == nil {
		panic("errors: target cannot be nil")
	}

	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("errors: target must be a non-nil pointer")
	}

	if e := typ.Elem(); e.Kind() != reflect.Interface && !e.Implements(tError) {
		panic("errors: *target must be interface or implement error")
	}

	targetType := typ.Elem()
	assigned := false
	Iter(err, func(cur error) bool {
		if reflect.TypeOf(cur).AssignableTo(targetType) {
			val.Elem().Set(reflect.ValueOf(cur))
			assigned = true
		} else if x, ok := err.(errConv); ok && x.As(target) {
			assigned = true
		}

		// continue searching in error tree until we found a matching error
		return !assigned
	})

	return assigned
}

// Find walks the complete error tree, trying to find an error that matches
// target. The first error matching target is returned.
//
// An error is matched with target, if the error is equal to the target,
// if error implements Is(error) bool that returns true for target, or
// target implements Is(error) bool such that Is(error) returne true.
func Find(err, target error) error {
	isComparable := reflect.TypeOf(target).Comparable()
	var targetCheck errCheckIs
	if tmp, ok := target.(errCheckIs); ok {
		targetCheck = tmp
	}

	return FindIf(err, func(cur error) bool {
		if isComparable && cur == target {
			return true
		}

		if errCheck, ok := cur.(errCheckIs); ok && errCheck.Is(target) {
			return true
		}
		if targetCheck != nil && targetCheck.Is(cur) {
			return true
		}

		return false
	})
}

// FindIf returns the first error in the error tree, that matches the
// given predicate.
func FindIf(in error, pred func(err error) bool) error {
	var found error
	Iter(in, func(err error) bool {
		matches := pred(err)
		if matches {
			found = err
			return false
		}
		return true
	})

	return found
}

// NumCauses returns the number of direct errors the error value wraps.
func NumCauses(in error) int {
	switch err := in.(type) {
	case wrappedError:
		return len(err.WrappedErrors())
	case multiCauser:
		return err.NumCauses()
	case causer:
		if err.Cause() == nil {
			return 0
		}
		return 1
	case unwrapper:
		if err.Unwrap() == nil {
			return 0
		}
		return 1
	default:
		return 0
	}
}

// Cause returns the i-th cause from the error value.
func Cause(in error, i int) error {
	switch err := in.(type) {
	case multiCauser:
		return err.Cause(i)

	case wrappedError:
		return err.WrappedErrors()[i]

	case causer:
		if i > 0 {
			panic("index out of bounds")
		}
		return err.Cause()

	case unwrapper:
		if i > 0 {
			panic("index out of bounds")
		}
		return err.Unwrap()

	default:
		return nil
	}
}

// Cause returns the first wrapped cause from the error value.
func Unwrap(in error) error {
	switch err := in.(type) {
	case unwrapper:
		return err.Unwrap()
	case causer:
		return err.Cause()
	case wrappedError:
		errs := err.WrappedErrors()
		if len(errs) == 0 {
			return nil
		}
		return errs[0]
	case multiCauser:
		if err.NumCauses() == 0 {
			return nil
		}
		return err.Cause(0)
	default:
		return nil
	}
}

// Walk walks the complete error tree.
func Walk(in error, fn func(error)) {
	Iter(in, func(err error) bool {
		fn(err)
		return true
	})
}

// Collect returns all errors in the error tree that matches the pred.
func Collect(in error, pred func(error) bool) []error {
	var errs []error
	Walk(in, func(err error) {
		if pred(err) {
			errs = append(errs, err)
		}
	})
	return errs
}

// CollectType returns all errors that are convertible to the error type of sample.
func CollectType(in, sample error) []error { return Collect(in, PredType(sample)) }

// IsType checks if any error in the error tree is convertible to type of sample.
func IsType(in, sample error) bool { return IsIf(in, PredType(sample)) }

// FindType finds the first error the is convertible to the error type of sample.
func FindType(in, sample error) error { return FindIf(in, PredType(sample)) }

// PredType creates a predicate checking if the type of an error value matches
// the type of sample.
//
// The predicate checks if the error type is equal or convertible to the sample
// type.
func PredType(sample error) func(error) bool {
	sampleType := reflect.TypeOf(sample).Elem()
	return func(current error) bool {
		t := reflect.TypeOf(current).Elem()
		return t == sampleType || t.ConvertibleTo(sampleType)
	}
}

// WalkEach walks every single error value in the given array of errors.
func WalkEach(errs []error, fn func(error)) {
	for _, err := range errs {
		Walk(err, fn)
	}
}

// Iter iterates the complete error tree calling fn on each error value found.
// The user function fn can stop the iteration by returning false.
func Iter(in error, fn func(err error) bool) {
	doIter(in, fn)
}

func doIter(in error, fn func(err error) bool) bool {
	for {
		if in == nil {
			return true // continue searching
		}

		// call fn and back-propagate search decision
		if cont := fn(in); !cont {
			return cont
		}

		switch err := in.(type) {
		case causer:
			in = err.Cause()

		case unwrapper:
			in = err.Unwrap()

		case multiCauser:
			num := err.NumCauses()
			switch num {
			case 0:
				return true

			case 1:
				in = err.Cause(0)

			default:
				for i := 0; i < num; i++ {
					if cont := doIter(err.Cause(i), fn); !cont {
						return false
					}
				}
				return true
			}

		case wrappedError:
			for _, cause := range err.WrappedErrors() {
				if cont := doIter(cause, fn); !cont {
					return false
				}
			}
			return true

		default:
			return true
		}
	}
}
