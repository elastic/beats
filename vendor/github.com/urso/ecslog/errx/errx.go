package errx

// Provide common functions for iterating and querying properties of errors.
//
// Can query/iterate errors from:
// - github.com/elastic/go-txfile/txerr
// - github.com/hashicorp/go-multierror
// - github.com/pkg/errors
// - github.com/urso/ecslog/errx

import (
	"reflect"

	"github.com/urso/ecslog/ctxtree"
)

type errCoder interface {
	ErrCode() error
}

// causer interface allows us to iterate errors with unique causes.
// The interface is compatible to github.com/pkg/errors.
type causer interface {
	Cause() error
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

func NumCauses(in error) int {
	switch err := in.(type) {
	case causer:
		if err.Cause() == nil {
			return 0
		}
		return 1

	case multiCauser:
		return err.NumCauses()

	case wrappedError:
		return len(err.WrappedErrors())

	default:
		return 0
	}
}

func Cause(in error, i int) error {
	switch err := in.(type) {
	case causer:
		if i > 0 {
			panic("index out of bounds")
		}
		return err.Cause()

	case multiCauser:
		return err.Cause(i)

	case wrappedError:
		return err.WrappedErrors()[i]

	default:
		return nil
	}
}

func At(err error) (string, int) {
	poser, ok := err.(interface{ At() (string, int) })
	if !ok {
		return "", -1
	}

	return poser.At()
}

func ErrContext(err error) *ctxtree.Ctx {
	ctxer, ok := err.(interface{ Context() *ctxtree.Ctx })
	if !ok {
		return nil
	}

	return ctxer.Context()
}

func Is(errCode error, in error) bool {
	return FindCode(in, errCode) != nil
}

func Collect(in error, pred func(error) bool) []error {
	var errs []error
	Walk(in, func(err error) {
		if pred(err) {
			errs = append(errs, err)
		}
	})
	return errs
}

func Contains(in error, pred func(err error) bool) bool {
	return Find(in, pred) != nil
}

// FindErrWith returns the first error in the error tree, that matches the
// given predicate.
func Find(in error, pred func(err error) bool) error {
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

func CollectType(in, sample error) []error { return Collect(in, PredType(sample)) }
func ContainsType(in, sample error) bool   { return Contains(in, PredType(sample)) }
func FindType(in, sample error) error      { return Find(in, PredType(sample)) }
func PredType(sample error) func(error) bool {
	sampleType := reflect.TypeOf(sample).Elem()
	return func(current error) bool {
		t := reflect.TypeOf(current).Elem()
		return t == sampleType
	}
}

func CollectValue(in, val error) []error { return Collect(in, PredValue(val)) }
func ContainsValue(in, val error) bool   { return Contains(in, PredValue(val)) }
func FindValue(in, val error) error      { return Find(in, PredValue(val)) }
func PredValue(val error) func(error) bool {
	return func(err error) bool {
		return err == val
	}
}

func CollectCode(in, code error) []error { return Collect(in, PredCode(code)) }
func ContainsCode(in, code error) bool   { return Contains(in, PredCode(code)) }
func FindCode(in, code error) error      { return Find(in, PredCode(code)) }
func PredCode(code error) func(error) bool {
	return func(current error) bool {
		if err, ok := current.(errCoder); ok {
			return err.ErrCode() == code
		}
		return false
	}
}

// Walk walks the complete error tree.
func Walk(in error, fn func(error)) {
	Iter(in, func(err error) bool {
		fn(err)
		return true
	})
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
