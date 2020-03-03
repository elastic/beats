package ctxtool

import (
	"context"
	"time"
)

type cancelContext struct {
	canceller
}

// AutoCancel collects cancel functions to be executed at the end of the
// function scope.
//
// Example:
//   var ac AutoCancel
//   defer ac.Cancel()
//   ctx := ac.With(context.WithCancel(context.Background()))
//   ctx := ac.With(context.WithTimeout(ctx, 5 * time.Second))
//   ... // do something with ctx
type AutoCancel struct {
	funcs []context.CancelFunc
}

// Cancel calls all registered cancel functions in reverse order.
func (ac *AutoCancel) Cancel() {
	for _, fn := range ac.funcs {
		defer fn()
	}
}

// Add adds a new cancel function to the AutoCancel. The function will be run
// before any other already registered cancel function.
func (ac *AutoCancel) Add(fn context.CancelFunc) {
	ac.funcs = append(ac.funcs, fn)
}

// With is used to wrap a Context constructer call that returns a context and a
// cancel function.  The cancel function is automatically added to AutoCancel
// and the original context is returned as is.
func (ac *AutoCancel) With(ctx context.Context, cancel context.CancelFunc) context.Context {
	ac.Add(cancel)
	return ctx
}

// FromCanceller creates a new context from a canceller. If a contex is passed,
// then Deadline and Value will be ignored.
func FromCanceller(c canceller) context.Context {
	return cancelContext{c}
}

func (c cancelContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (c cancelContext) Value(key interface{}) interface{} {
	return nil
}
