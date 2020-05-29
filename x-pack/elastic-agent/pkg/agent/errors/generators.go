// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package errors

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

// M creates a meta entry for an error
func M(key string, val interface{}) MetaRecord {
	return MetaRecord{key: key,
		val: val,
	}
}

// New constructs an Agent Error based on provided parameteres.
// Accepts:
// - string for error message [0..1]
// - error for inner error [0..1]
// - ErrorType for defining type [0..1]
// - MetaRecords for enhancing error with metadata [0..*]
// If optional arguments are provided more than once (message, error, type), then
// last argument overwrites previous ones.
func New(args ...interface{}) error {
	agentErr := agentError{}
	agentErr.meta = make(map[string]interface{})

	for _, arg := range args {
		switch arg := arg.(type) {
		case string:
			agentErr.msg = arg
		case error:
			agentErr.err = arg
		case ErrorType:
			agentErr.errType = arg
		case MetaRecord:
			agentErr.meta[arg.key] = arg.val
		}
	}

	if agentErr.err == nil {
		agentErr.err = errors.New("unknown error")

		if _, file, line, ok := runtime.Caller(1); ok {
			agentErr.err = errors.Wrapf(agentErr.err, fmt.Sprintf("%s[%d]", file, line))
		}
	}

	return agentErr
}
