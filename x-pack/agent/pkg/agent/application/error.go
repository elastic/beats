// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import "github.com/pkg/errors"

var (
	// ErrInvalidPeriod is returned when a reload period interval is not valid
	ErrInvalidPeriod = errors.New("period must be higher than zero")

	// ErrInvalidMgmtMode is returned when an unknown mode is selected.
	ErrInvalidMgmtMode = errors.New("invalid management mode")
)
