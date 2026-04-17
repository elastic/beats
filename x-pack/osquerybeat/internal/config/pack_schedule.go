// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package config

import (
	"errors"
	"fmt"
)

var (
	// ErrConflictingScheduleModes is returned when a query would use both native
	// interval-based scheduling (interval > 0) and rrule_schedule.
	ErrConflictingScheduleModes = errors.New("query cannot use both native interval scheduling and rrule_schedule")

	// ErrPackConflictingScheduleDefaults is returned when a pack defines both
	// default_native_schedule.interval and an enabled default_rrule_schedule.
	ErrPackConflictingScheduleDefaults = errors.New("pack cannot define both default_native_schedule.interval and default_rrule_schedule")

	// ErrPackMixedScheduleModes is returned when queries in a pack without schedule
	// defaults use different schedule modes (native interval vs rrule_schedule vs unscheduled).
	ErrPackMixedScheduleModes = errors.New("pack queries must share the same schedule mode (all native interval, all rrule_schedule, or all unscheduled)")

	// ErrPackQueryViolatesPackScheduleDefault is returned when a query's schedule
	// does not match the pack's default_native_schedule or default rrule_schedule.
	ErrPackQueryViolatesPackScheduleDefault = errors.New("query schedule conflicts with the pack's schedule default")

	// ErrPackNativeScheduleMetadataWithoutInterval is returned when default_native_schedule
	// sets start_date without a positive interval.
	ErrPackNativeScheduleMetadataWithoutInterval = errors.New("default_native_schedule cannot set start_date without a positive interval")
)

// UsesNativeSchedule reports whether the query is intended to run on osquery's
// native scheduler (requires a positive interval in the rendered config).
func (q Query) UsesNativeSchedule() bool {
	return q.Interval > 0
}

// ValidatePackScheduleDefaults ensures the pack does not mix native and RRULE defaults.
func ValidatePackScheduleDefaults(pack Pack) error {
	rruleOn := pack.DefaultRRuleSchedule != nil && pack.DefaultRRuleSchedule.IsEnabled()
	nativeOn := pack.DefaultNativeSchedule.Interval > 0
	if rruleOn && nativeOn {
		return ErrPackConflictingScheduleDefaults
	}
	if pack.DefaultNativeSchedule.Interval <= 0 && pack.DefaultNativeSchedule.StartDate != "" {
		return ErrPackNativeScheduleMetadataWithoutInterval
	}
	return nil
}

// ValidateQueryScheduleMode ensures a query does not combine native interval with RRULE.
func ValidateQueryScheduleMode(q Query) error {
	if q.UsesNativeSchedule() && q.RRuleSchedule.IsEnabled() {
		return ErrConflictingScheduleModes
	}
	return nil
}

type packQueryScheduleMode int

const (
	packQueryScheduleUnscheduled packQueryScheduleMode = iota
	packQueryScheduleNative
	packQueryScheduleRRule
)

func queryPackScheduleMode(q Query) packQueryScheduleMode {
	if q.UsesNativeSchedule() {
		return packQueryScheduleNative
	}
	if q.RRuleSchedule.IsEnabled() {
		return packQueryScheduleRRule
	}
	return packQueryScheduleUnscheduled
}

// ValidatePackQueriesAfterMerge checks the pack's queries after MergeQueryWithPackScheduleDefaults.
// When the pack sets default_native_schedule.interval, every query must use native scheduling.
// When the pack sets an enabled default rrule_schedule, every query must use RRULE scheduling.
// Otherwise, all queries must share one mode: native, RRULE, or unscheduled.
func ValidatePackQueriesAfterMerge(pack Pack) error {
	hasNativeDefault := pack.DefaultNativeSchedule.Interval > 0
	hasRRuleDefault := pack.DefaultRRuleSchedule != nil && pack.DefaultRRuleSchedule.IsEnabled()

	for qname, q := range pack.Queries {
		if hasNativeDefault && !q.UsesNativeSchedule() {
			return fmt.Errorf("%w: query %q must use native interval scheduling (pack sets default_native_schedule)", ErrPackQueryViolatesPackScheduleDefault, qname)
		}
		if hasRRuleDefault && !q.RRuleSchedule.IsEnabled() {
			return fmt.Errorf("%w: query %q must use rrule_schedule (pack sets default_rrule_schedule)", ErrPackQueryViolatesPackScheduleDefault, qname)
		}
	}

	modes := make(map[packQueryScheduleMode]struct{})
	for _, q := range pack.Queries {
		modes[queryPackScheduleMode(q)] = struct{}{}
	}
	if len(modes) > 1 {
		return ErrPackMixedScheduleModes
	}
	return nil
}

// MergeQueryWithPackScheduleDefaults applies pack-level schedule defaults to a query.
// Query-level fields take precedence for filling fields, but the merged pack must still
// pass ValidatePackQueriesAfterMerge (same schedule mode for all queries, aligned with pack defaults).
// Returns an error if the merged query violates ValidateQueryScheduleMode.
func MergeQueryWithPackScheduleDefaults(pack Pack, q Query) (Query, error) {
	if err := ValidateQueryScheduleMode(q); err != nil {
		return q, err
	}

	queryHasRRule := q.RRuleSchedule.IsEnabled()
	queryHasNative := q.UsesNativeSchedule()

	if !queryHasRRule {
		if q.Interval == 0 && pack.DefaultNativeSchedule.Interval > 0 {
			q.Interval = pack.DefaultNativeSchedule.Interval
		}
		if q.StartDate == "" && pack.DefaultNativeSchedule.StartDate != "" {
			q.StartDate = pack.DefaultNativeSchedule.StartDate
		}
	}

	if !queryHasNative {
		if !q.RRuleSchedule.IsEnabled() && pack.DefaultRRuleSchedule != nil && pack.DefaultRRuleSchedule.IsEnabled() {
			cpy := *pack.DefaultRRuleSchedule
			q.RRuleSchedule = &cpy
		}
	}

	if q.SpaceID == "" && pack.DefaultSpaceID != "" {
		q.SpaceID = pack.DefaultSpaceID
	}

	if err := ValidateQueryScheduleMode(q); err != nil {
		return q, err
	}
	return q, nil
}
