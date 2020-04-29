// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"errors"
	"time"
)

var (
	errWrongContext   = errors.New("invalid type, expecting stack context")
	errMissingStackID = errors.New("missing stack id")
)

type stackContext struct {
	ID        *string
	StartedAt time.Time
}

func newStackContext() *stackContext {
	return &stackContext{StartedAt: time.Now()}
}
