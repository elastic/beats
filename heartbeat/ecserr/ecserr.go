package ecserr

import (
	"fmt"
)

// ECSErr represents an error per the ECS specification
type ECSErr struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
	// StackTrace is optional, since it's more rare, it's nicer to
	// have it JSON serialize to null.
	// The other fields are not pointers since they should be there most of
	// the time.
	StackTrace *string `json:"stack_trace"`
}

func NewECSErr(typ string, code string, message string) *ECSErr {
	return NewECSErrWithStack(typ, code, message, nil)
}

func NewECSErrWithStack(typ string, code string, message string, stackTrace *string) *ECSErr {
	return &ECSErr{
		Type:       typ,
		Code:       code,
		Message:    message,
		StackTrace: stackTrace,
	}
}

func (e *ECSErr) Error() string {
	// We don't get fancy here because we
	// want to allow wrapped errors to invoke this without duplicating fields
	// see wrappers.go for more info on how we set the final errors value for
	// events.
	return e.Message
}

func (e *ECSErr) String() string {
	// This can be fancy, see note in Error()
	return fmt.Sprintf("error %s (type='%s', code='%s')", e.Message, e.Type, e.Code)
}

const (
	ETYPE_IO = "io"
)

type SynthErrType string

func NewBadCmdStatusErr(exitCode int, cmd string) *ECSErr {
	return NewECSErr(
		ETYPE_IO,
		"BAD_CMD_STATUS",
		fmt.Sprintf("command '%s' exited unexpectedly with code: %d", cmd, exitCode),
	)
}
