package common

import (
	"fmt"
)

// ErrInputNotFinished struct for reporting errors related to not finished inputs
type ErrInputNotFinished struct {
	      state string
	}

// Error method of ErrInputNotFinished
func (e *ErrInputNotFinished) Error() string {
	return fmt.Sprintf("Can only start an input when all related states are finished: %+v", e.state)
}
