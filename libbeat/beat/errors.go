package beat

import "fmt"

var (
	// GracefulExit is an error that signals to exit with a code of 0.
	GracefulExit = ExitError{}
)

// ExitError is an error type that can be returned to set a specific exit code.
type ExitError struct {
	ExitCode int
	Cause    error
}

func (e ExitError) Error() string {
	if e.Cause != nil {
		return e.Cause.Error()
	}

	return ""
}

// NewExitError returns a new ExitError.
func NewExitError(code int, format string, args ...interface{}) error {
	return ExitError{
		ExitCode: code,
		Cause:    fmt.Errorf(format, args),
	}
}
