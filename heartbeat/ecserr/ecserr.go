// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ecserr

import (
	"fmt"
	"time"
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

func NewCmdTimeoutStatusErr(timeout time.Duration, cmd string) *ECSErr {
	return NewECSErr(
		ETYPE_IO,
		"CMD_TIMEOUT",
		fmt.Sprintf("command '%s' did not exit before extended timeout: %s", cmd, timeout.String()),
	)
}

func NewSyntheticsCmdCouldNotStartErr(reason error) *ECSErr {
	return NewECSErr(
		ETYPE_IO,
		"SYNTHETICS_CMD_COULD_NOT_START",
		fmt.Sprintf("could not start command not found: %s", reason),
	)
}
