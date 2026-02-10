// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"errors"
	"time"
)

// Common schedule errors
var (
	ErrStartDateAfterEndDate = errors.New("start date must be before end date")
	ErrOutsideScheduleWindow = errors.New("current time is outside schedule window")
	ErrIntervalTooShort      = errors.New("schedule interval must be at least 1 day")
)

// Common schedule constants
const (
	// MaxSplay is the maximum allowed splay duration (12 hours)
	// Since minimum interval is 1 day, 12h splay is always safe (at most 50% of interval)
	MaxSplay = 12 * time.Hour

	// DefaultSplay is the default splay duration if not specified (disabled)
	DefaultSplay = 0

	// MinInterval is the minimum allowed interval between executions
	// Production: 24 * time.Hour (1 day) - ensures splay (0-12h max) is always safe
	MinInterval = 24 * time.Hour
)

