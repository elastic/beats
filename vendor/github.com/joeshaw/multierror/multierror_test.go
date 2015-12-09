package multierror

import (
	"fmt"
	"testing"
)

func TestZeroErrors(t *testing.T) {
	var e Errors
	err := e.Err()
	if err != nil {
		t.Error("An empty Errors Err() method should return nil")
	}
}

func TestNonZeroErrors(t *testing.T) {
	var e Errors
	e = append(e, fmt.Errorf("An error"))
	err := e.Err()
	if err == nil {
		t.Error("A nonempty Errors Err() method should not return nil")
	}

	merr, ok := err.(*MultiError)
	if !ok {
		t.Error("Errors Err() method should return a *MultiError")
	}

	if len(merr.Errors) != 1 {
		t.Error("The MultiError Errors field was not of length 1")
	}

	if merr.Errors[0] != e[0] {
		t.Error("The Error in merr.Errors was not the original error instance provided")
	}

	if merr.Error() != "1 error: An error" {
		t.Error("MultiError (single) string was not as expected")
	}

	e = append(e, fmt.Errorf("Another error"))
	merr = e.Err().(*MultiError)
	if merr.Error() != "2 errors: An error; Another error" {
		t.Error("MultiError (multiple) string was not as expected")
	}
}
