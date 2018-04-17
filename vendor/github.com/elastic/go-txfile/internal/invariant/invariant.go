// Package invariant provides helpers for checking and panicing on faulty invariants.
package invariant

import "fmt"

// Check will raise an error with the provided message in case b is false.
func Check(b bool, msg string) {
	if b {
		return
	}

	if msg == "" {
		panic("failing invariant")
	}
	panic(msg)
}

// Checkf will raise an error in case b is false. Checkf accept a fmt.Sprintf
// compatible format string with parameters.
func Checkf(b bool, msgAndArgs ...interface{}) {
	if b {
		return
	}

	switch len(msgAndArgs) {
	case 0:
		panic("failing invariant")
	case 1:
		panic(msgAndArgs[0].(string))
	default:
		panic(fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...))
	}
}

// CheckNot will raise an error with the provided message in case b is true.
func CheckNot(b bool, msg string) {
	Check(!b, msg)
}

// CheckNotf will raise an error with the provided message in case b is true.
// CheckNotf accept a fmt.Sprintf compatible format string with parameters.
func CheckNotf(b bool, msgAndArgs ...interface{}) {
	Checkf(!b, msgAndArgs...)
}

// Unreachable marks some code sequence that must never be executed.
func Unreachable(msg string) {
	panic(msg)
}

// Unreachablef marks some code sequence that must never be executed.
func Unreachablef(f string, vs ...interface{}) {
	panic(fmt.Sprintf(f, vs...))
}
