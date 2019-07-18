// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcp

import (
	"errors"
)

var (
	errWrongContext        = errors.New("invalid type, expecting function context")
	errMissingFunctionName = errors.New("missing function operation name")
)

type functionContext struct {
	name *string
}
