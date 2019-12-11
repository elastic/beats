// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/x-pack/libbeat/management/api"
)

// ErrorType is type of error that the events endpoint understand.
type ErrorType string

// ConfigError is the type of error send when an unpack or a blacklist happen.
var ConfigError = ErrorType("CONFIG")

// ErrorEvent is the event type when an error happen.
var ErrorEvent = api.EventType("ERROR")

// Error is a config error to be reported to kibana.
type Error struct {
	Type ErrorType
	UUID uuid.UUID
	Err  error
}

// EventType returns a ErrorEvent.
func (e *Error) EventType() api.EventType {
	return ErrorEvent
}

// MarshalJSON transform an error into a JSON document.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		UUID    string `json:"uuid"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}{
		UUID:    e.UUID.String(),
		Type:    string(e.Type),
		Message: e.Err.Error(),
	})
}

// UnmarshalJSON unmarshals a event of the type Error.
func (e *Error) UnmarshalJSON(b []byte) error {
	res := &struct {
		UUID    string `json:"uuid,omitempty"`
		Type    string `json:"type"`
		Message string `json:"message"`
	}{}

	if err := json.Unmarshal(b, res); err != nil {
		return err
	}

	uuid, err := uuid.FromString(res.UUID)
	if err != nil {
		return err
	}
	*e = Error{
		Type: ErrorType(res.Type),
		UUID: uuid,
		Err:  errors.New(res.Message),
	}
	return nil
}

func (e *Error) Error() string {
	return e.Err.Error()
}

// Errors contains mutiples config error.
type Errors []*Error

// Errors makes sure we can display the error in the logger.
func (er *Errors) Error() string {
	var s strings.Builder
	if len(*er) == 1 {
		s.WriteString("1 error: ")
	} else {
		s.WriteString(strconv.Itoa(len(*er)))
		s.WriteString(" errors: ")
	}
	for idx, err := range *er {
		if idx != 0 {
			s.WriteString("; ")
		}
		s.WriteString(err.Error())
	}
	return s.String()
}

// IsEmpty returns true when we don't have any errors.
func (er *Errors) IsEmpty() bool {
	return len(*er) == 0
}

// NewConfigError wraps an error to be a management error of a specific ConfigError Type
func NewConfigError(err error) *Error {
	return &Error{Type: ConfigError, Err: err}
}
