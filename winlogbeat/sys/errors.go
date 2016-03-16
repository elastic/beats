package sys

// InsufficientBufferError indicates the buffer passed to a system call is too
// small.
type InsufficientBufferError struct {
	Cause        error
	RequiredSize int // Size of the buffer that is required.
}

// Error returns the cause of the insufficient buffer error.
func (e InsufficientBufferError) Error() string {
	return e.Cause.Error()
}
