// multierror is a simple Go package for combining multiple errors together.
package multierror

import (
	"bytes"
	"fmt"
)

// The Errors type wraps a slice of errors
type Errors []error

// Returns a MultiError struct containing this Errors instance, or nil
// if there are zero errors contained.
func (e Errors) Err() error {
	if len(e) == 0 {
		return nil
	}

	return &MultiError{Errors: e}
}

// The MultiError type implements the error interface, and contains the
// Errors used to construct it.
type MultiError struct {
	Errors Errors
}

// Returns a concatenated string of the contained errors
func (m *MultiError) Error() string {
	var buf bytes.Buffer

	if len(m.Errors) == 1 {
		buf.WriteString("1 error: ")
	} else {
		fmt.Fprintf(&buf, "%d errors: ", len(m.Errors))
	}

	for i, err := range m.Errors {
		if i != 0 {
			buf.WriteString("; ")
		}

		buf.WriteString(err.Error())
	}

	return buf.String()
}
