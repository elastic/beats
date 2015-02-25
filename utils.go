package main

import "fmt"

// bare-bone Error type, to make it easy to create
// our own exceptions with a string.
type GenericError struct {
	msg string
}

func (err GenericError) Error() string {
	return err.msg
}

// Convenience function for quickly returning GenericErrors
func MsgError(format string, v ...interface{}) error {
	return GenericError{
		msg: fmt.Sprintf(format, v...),
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
