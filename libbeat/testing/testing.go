package testing

// Driver for testing, manages test flow and controls output
type Driver interface {
	// Run tests under a given namespace
	Run(name string, f func(Driver))

	// Info reports some value retrieved while testing to the user
	Info(field, value string)

	// Warn shows a warning to the user
	Warn(field string, reason string)

	// Error will report an error on the given field if err != nil, will report OK if not
	Error(field string, err error)

	// Fatal behaves like error but stops current goroutine on error
	Fatal(field string, err error)

	// Shows given result to the user
	Result(data string)
}

// Testable is optionally implemented by clients that support self testing.
// Test method will test current settings work for this output.
type Testable interface {
	Test(Driver)
}
