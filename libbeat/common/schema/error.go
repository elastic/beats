package schema

import "fmt"

const (
	RequiredType ErrorType = iota
	OptionalType ErrorType = iota
)

type ErrorType int

type Error struct {
	key       string
	message   string
	errorType ErrorType
}

func NewError(key string, message string) *Error {
	return &Error{
		key:       key,
		message:   message,
		errorType: RequiredType,
	}
}

func (err *Error) SetType(errorType ErrorType) {
	err.errorType = errorType
}

func (err *Error) IsType(errorType ErrorType) bool {
	return err.errorType == errorType
}

func (err *Error) Error() string {
	return fmt.Sprintf("Missing field: %s, Error: %s", err.key, err.message)
}

type KeyNotFoundError struct {
	Key string
	Err error
}

func (err *KeyNotFoundError) Error() string {
	msg := fmt.Sprintf("Key `%s` not found", err.Key)
	if err.Err != nil {
		msg += ": " + err.Err.Error()
	}
	return msg
}
