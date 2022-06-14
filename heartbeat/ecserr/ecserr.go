package ecserr

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
)

type ECSErr struct {
	ID         uuid.UUID `json:"id"`
	Message    string    `json:"message"`
	Code       string    `json:"code"`
	StackTrace string    `json:"stack_trace"`
	Type       string    `json:"type"`
}

func NewECSErr(typ string, code string, message string, stackTrace ...string) *ECSErr {
	id, _ := uuid.NewV4()
	return &ECSErr{
		ID:         id,
		Type:       typ,
		Code:       code,
		Message:    message,
		StackTrace: strings.Join(stackTrace, "\n"),
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
	return fmt.Sprintf("error %s (type='%s', code='%s', id='%s' stacktrace='%s')", e.Message, e.Type, e.Code, e.ID.String(), e.StackTrace)
}

const (
	ETYPE_IO = "io"
)

type SynthErrType string

func NewBadCmdStatusErr(exitCode int, cmd string) *ECSErr {
	return NewECSErr(
		"IO",
		"BAD_CMD_STATUS",
		fmt.Sprintf("command '%s' exited unexpectedly with code: %d", cmd, exitCode),
	)
}
