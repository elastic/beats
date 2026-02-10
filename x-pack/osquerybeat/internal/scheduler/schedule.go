// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scheduler

import (
	"errors"
	"fmt"
	"time"

	"github.com/teambition/rrule-go"
)

// RRULE-specific errors
var (
	ErrInvalidRRule = errors.New("invalid rrule expression")
)

// RecurrenceSchedule represents an RRULE-based schedule with optional time window and splay
type RecurrenceSchedule struct {
	// RRule is the RFC 5545 recurrence rule string
	// Examples:
	//   "FREQ=DAILY" - every day
	//   "FREQ=WEEKLY;BYDAY=MO,WE,FR" - every Monday, Wednesday, Friday
	//   "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE" - every 2 weeks on Monday and Wednesday
	//   "FREQ=MONTHLY;BYMONTHDAY=1" - first day of every month
	RRule string `config:"rrule"`

	// StartDate is the earliest time the schedule can execute (required)
	// Also serves as DTSTART for the RRULE
	StartDate *time.Time `config:"start_date"`

	// EndDate is the latest time the schedule can execute (optional)
	EndDate *time.Time `config:"end_date"`

	// Splay is a random delay added to each execution
	// to prevent thundering herd problems when many agents share the same schedule
	Splay time.Duration `config:"splay"`

	// Parsed rrule object (not serialized)
	rule *rrule.RRule
}

// Parse validates and parses the RRULE expression
func (s *RecurrenceSchedule) Parse() error {
	if s.RRule == "" {
		return ErrInvalidRRule
	}
	if s.StartDate == nil {
		return fmt.Errorf("start_date is required for rrule schedules")
	}

	// Parse the RRULE
	rule, err := rrule.StrToRRule(s.RRule)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRRule, err)
	}

	// Use start_date as DTSTART for determinism
	rule.DTStart(*s.StartDate)

	// If EndDate is set in schedule, use it as UNTIL (unless UNTIL is already in the rule)
	// Note: rrule-go handles UNTIL in the parsed rule, so we just validate here
	if s.EndDate != nil && s.StartDate != nil && s.EndDate.Before(*s.StartDate) {
		return ErrStartDateAfterEndDate
	}

	s.rule = rule

	// Validate minimum interval
	if err := s.validateMinInterval(); err != nil {
		return err
	}

	return nil
}

// validateMinInterval ensures the schedule doesn't run more often than once per day
func (s *RecurrenceSchedule) validateMinInterval() error {
	if s.rule == nil {
		return nil
	}

	// Get the next 3 execution times and check intervals
	now := time.Now()
	times := s.rule.Between(now, now.Add(365*24*time.Hour), false)

	if len(times) < 2 {
		// Not enough occurrences to check interval - that's fine
		return nil
	}

	// Check interval between consecutive executions (up to first 3)
	checkCount := 3
	if len(times) < checkCount {
		checkCount = len(times)
	}

	for i := 1; i < checkCount; i++ {
		interval := times[i].Sub(times[i-1])
		if interval < MinInterval {
			return ErrIntervalTooShort
		}
	}

	return nil
}

// Next returns the next execution time after the given time, considering the schedule window
// Returns zero time if no next execution exists within the window
func (s *RecurrenceSchedule) Next(t time.Time) time.Time {
	if s.rule == nil {
		return time.Time{}
	}

	// If before start date, use start date as reference
	if s.StartDate != nil && t.Before(*s.StartDate) {
		t = s.StartDate.Add(-time.Second) // Subtract a second so After() can return start time if it matches
	}

	next := s.rule.After(t, false)

	// If after end date, return zero time
	if s.EndDate != nil && !next.IsZero() && next.After(*s.EndDate) {
		return time.Time{}
	}

	return next
}

// IsWithinWindow checks if the given time is within the schedule's time window
func (s *RecurrenceSchedule) IsWithinWindow(t time.Time) bool {
	if s.StartDate != nil && t.Before(*s.StartDate) {
		return false
	}
	if s.EndDate != nil && t.After(*s.EndDate) {
		return false
	}
	return true
}

// ExecutionIndex returns the 1-based execution number for the given time.
// Given start date X and RRULE Y, this is "which occurrence is t?" (e.g. 3 = 3rd execution).
// Returns (0, false) if the schedule has no rule, t is before the first occurrence, or t is outside the schedule window.
func (s *RecurrenceSchedule) ExecutionIndex(t time.Time) (n int, ok bool) {
	if s.rule == nil {
		return 0, false
	}
	if !s.IsWithinWindow(t) {
		return 0, false
	}
	start := s.rule.GetDTStart()
	if t.Before(start) {
		return 0, false
	}
	// All occurrences from DTSTART through t (inclusive); len = execution index (1-based)
	occurrences := s.rule.Between(start, t, true)
	if len(occurrences) == 0 {
		return 0, false
	}
	return len(occurrences), true
}

// GetMaxSplayDuration returns the configured splay duration
func (s *RecurrenceSchedule) GetMaxSplayDuration() time.Duration {
	return s.Splay
}

// ValidateSplay checks if splay configuration is valid
// Since minimum interval is 1 day and max splay is 12h, splay is always safe
func (s *RecurrenceSchedule) ValidateSplay() error {
	if s.Splay < 0 {
		return fmt.Errorf("splay cannot be negative: %v", s.Splay)
	}
	if s.Splay > MaxSplay {
		return fmt.Errorf("splay cannot exceed %v, got: %v", MaxSplay, s.Splay)
	}
	return nil
}

// IsActive returns true if the schedule has a valid rrule
func (s *RecurrenceSchedule) IsActive() bool {
	return s.RRule != "" && s.rule != nil
}

// String returns the RRULE expression
func (s *RecurrenceSchedule) String() string {
	return s.RRule
}

// Unpack implements the config unpacker interface
func (s *RecurrenceSchedule) Unpack(cfg map[string]interface{}) error {
	if v, ok := cfg["rrule"].(string); ok {
		s.RRule = v
	}

	if v, ok := cfg["start_date"].(string); ok && v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("invalid start_date: %w", err)
		}
		s.StartDate = &t
	}
	if s.StartDate == nil {
		return fmt.Errorf("start_date is required for rrule schedules")
	}

	if v, ok := cfg["end_date"].(string); ok && v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return fmt.Errorf("invalid end_date: %w", err)
		}
		s.EndDate = &t
	}

	// Parse splay - duration string (e.g., "30s", "5m", "2h")
	if v, ok := cfg["splay"].(string); ok && v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid splay duration '%s': %w", v, err)
		}
		if d < 0 {
			return fmt.Errorf("splay cannot be negative: %s", v)
		}
		if d > MaxSplay {
			return fmt.Errorf("splay cannot exceed %v, got: %s", MaxSplay, v)
		}
		s.Splay = d
	} else {
		// Default to 0s splay (disabled)
		s.Splay = DefaultSplay
	}

	return s.Parse()
}
