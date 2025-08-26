// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"sync"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	filterOldestTime = "oldestTimeFilter"
	filterStartTime  = "startTimeFilter"
)

// filterProvider exposes filters that needs to be applied on derived state.
// Once configured, retrieve filter applier using getApplierFunc.
type filterProvider struct {
	cfg *config

	staticFilters []filter
	once          sync.Once
}

func newFilterProvider(cfg *config) *filterProvider {
	fp := &filterProvider{
		cfg: cfg,
	}

	// derive static filters
	if cfg.StartTimestamp != "" {
		// note - errors should not occur as this has validated prior reaching here
		parse, _ := time.Parse(time.RFC3339, cfg.StartTimestamp)
		fp.staticFilters = append(fp.staticFilters, newStartTimestampFilter(parse))
	}

	return fp
}

// getApplierFunc returns aggregated filters valid at the time of retrival.
// Applier return true if state is valid for processing according to the underlying filter collection.
func (f *filterProvider) getApplierFunc() func(log *logp.Logger, s state) bool {
	filters := map[string]filter{}

	if f.cfg.IgnoreOlder != 0 {
		timeFilter := newOldestTimeFilter(f.cfg.IgnoreOlder, time.Now())
		filters[timeFilter.getID()] = timeFilter
	}

	for _, f := range f.staticFilters {
		filters[f.getID()] = f
	}

	f.once.Do(func() {
		// Ignore the oldest time filter once for initial startup only if start timestamp filter is defined
		// This allows users to ingest desired data from start time onwards.
		if filters[filterStartTime] != nil {
			delete(filters, filterOldestTime)
		}
	})

	return func(log *logp.Logger, s state) bool {
		for _, f := range filters {
			if !f.isValid(s) {
				log.Debugf("skipping processing of object '%s' by filter '%s'", s.Key, f.getID())
				return false
			}
		}

		return true
	}
}

// filter defines the fileter implementation contract
type filter interface {
	getID() string
	isValid(objState state) (valid bool)
}

// startTimestampFilter - filter out entries based on provided start time stamp, compared to filtering object's last modified times stamp.
type startTimestampFilter struct {
	id        string
	timeStart time.Time
}

func newStartTimestampFilter(start time.Time) *startTimestampFilter {
	return &startTimestampFilter{
		id:        filterStartTime,
		timeStart: start,
	}
}

func (s startTimestampFilter) isValid(objState state) bool {
	return s.timeStart.Before(objState.LastModified)
}

func (s startTimestampFilter) getID() string {
	return s.id
}

// oldestTimeFilter - filter out entries based on calculated oldest modified time, compared to filtering object's last modified times stamp.
type oldestTimeFilter struct {
	id         string
	timeOldest time.Time
}

func newOldestTimeFilter(timespan time.Duration, now time.Time) *oldestTimeFilter {
	return &oldestTimeFilter{
		id:         filterOldestTime,
		timeOldest: now.Add(-1 * timespan),
	}
}

func (s oldestTimeFilter) isValid(objState state) bool {
	return s.timeOldest.Before(objState.LastModified)
}

func (s oldestTimeFilter) getID() string {
	return s.id
}
