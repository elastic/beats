// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"errors"
)

// Common schedule errors
var (
	ErrStartDateAfterEndDate = errors.New("start date must be before end date")
	ErrOutsideScheduleWindow = errors.New("current time is outside schedule window")
)

// Common schedule constants
const (
	// DefaultSplay is the default splay duration if not specified (disabled)
	DefaultSplay = 0
)
