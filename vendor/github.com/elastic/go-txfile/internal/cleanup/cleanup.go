// Package cleanup provides common helpers for common cleanup patterns on defer
//
// Use the helpers with `defer`. For example use IfNot with `defer`, such that
// cleanup functions will be executed if `check` is false, no matter if an
// error has been returned or an panic has occured.
//
//     initOK := false
//     defer cleanup.IfNot(&initOK, func() {
//       cleanup
//     })
//
//     ... // init structures...
//
//     initOK = true // notify handler cleanup code must not be executed
//
package cleanup

// If will run the cleanup function if the bool value is true.
func If(check *bool, cleanup func()) {
	if *check {
		cleanup()
	}
}

// IfNot will run the cleanup function if the bool value is false.
func IfNot(check *bool, cleanup func()) {
	if !(*check) {
		cleanup()
	}
}

// IfPred will run the cleanup function if pred returns true.
func IfPred(pred func() bool, cleanup func()) {
	if pred() {
		cleanup()
	}
}

// IfNotPred will run the cleanup function if pred returns false.
func IfNotPred(pred func() bool, cleanup func()) {
	if !pred() {
		cleanup()
	}
}

// WithError returns a cleanup function calling a custom handler if an error occured.
func WithError(fn func(error), cleanup func() error) func() {
	return func() {
		if err := cleanup(); err != nil {
			fn(err)
		}
	}
}

// IgnoreError silently ignores errors in the cleanup function.
func IgnoreError(cleanup func() error) func() {
	return func() { _ = cleanup() }
}
