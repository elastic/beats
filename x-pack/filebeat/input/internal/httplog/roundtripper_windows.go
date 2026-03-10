// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package httplog

import (
	"errors"

	"golang.org/x/sys/windows"
)

func isInvalidWindowsName(err error) bool {
	return errors.Is(err, windows.ERROR_INVALID_NAME)
}
