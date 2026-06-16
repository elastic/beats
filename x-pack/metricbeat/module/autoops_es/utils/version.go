// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package utils

import (
	"fmt"
)

type VersionMismatchError struct {
	ExpectedVersion string
	ActualVersion   string
}

func (e *VersionMismatchError) Error() string {
	return fmt.Sprintf("version mismatch: expected %s, got %s", e.ExpectedVersion, e.ActualVersion)
}
