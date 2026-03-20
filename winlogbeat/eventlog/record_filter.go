// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build windows

package eventlog

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type eventIDRange struct {
	start uint32
	end   uint32
}

func (r eventIDRange) contains(id uint32) bool {
	return id >= r.start && id <= r.end
}

type recordFilter struct {
	ignoreOlder time.Duration
	providers   map[string]struct{}
	levels      map[uint8]struct{}
	includeIDs  []eventIDRange
	excludeIDs  []eventIDRange
}

func newRecordFilter(q query) (*recordFilter, error) {
	f := &recordFilter{
		ignoreOlder: q.IgnoreOlder,
	}

	if len(q.Provider) > 0 {
		f.providers = make(map[string]struct{}, len(q.Provider))
		for _, provider := range q.Provider {
			if provider != "" {
				f.providers[provider] = struct{}{}
			}
		}
	}

	if q.Level != "" {
		levels, err := parseLevels(q.Level)
		if err != nil {
			return nil, err
		}
		f.levels = levels
	}

	includeIDs, excludeIDs, err := parseEventIDRanges(q.EventID)
	if err != nil {
		return nil, err
	}
	f.includeIDs = includeIDs
	f.excludeIDs = excludeIDs

	return f, nil
}

func (f *recordFilter) match(r *Record) bool {
	if f == nil || r == nil {
		return true
	}

	if f.ignoreOlder > 0 && !r.TimeCreated.SystemTime.IsZero() {
		if time.Since(r.TimeCreated.SystemTime) > f.ignoreOlder {
			return false
		}
	}

	if len(f.providers) > 0 {
		if _, ok := f.providers[r.Provider.Name]; !ok {
			return false
		}
	}

	if len(f.levels) > 0 {
		if _, ok := f.levels[r.LevelRaw]; !ok {
			return false
		}
	}

	eventID := r.EventIdentifier.ID
	for _, ex := range f.excludeIDs {
		if ex.contains(eventID) {
			return false
		}
	}

	if len(f.includeIDs) == 0 {
		return true
	}
	for _, in := range f.includeIDs {
		if in.contains(eventID) {
			return true
		}
	}
	return false
}

func parseLevels(raw string) (map[uint8]struct{}, error) {
	levels := make(map[uint8]struct{})

	add := func(v ...uint8) {
		for _, n := range v {
			levels[n] = struct{}{}
		}
	}

	for _, expr := range strings.Split(raw, ",") {
		expr = strings.ToLower(strings.TrimSpace(expr))
		switch expr {
		case "verbose", "5":
			add(5)
		case "information", "info", "4":
			add(0, 4)
		case "warning", "warn", "3":
			add(3)
		case "error", "err", "2":
			add(2)
		case "critical", "crit", "1":
			add(1)
		case "0":
			add(0)
		default:
			return nil, fmt.Errorf("invalid level ('%s') for query", raw)
		}
	}

	return levels, nil
}

func parseEventIDRanges(raw string) ([]eventIDRange, []eventIDRange, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil, nil
	}

	var includes []eventIDRange
	var excludes []eventIDRange

	for _, component := range strings.Split(raw, ",") {
		component = strings.TrimSpace(component)
		if component == "" {
			return nil, nil, fmt.Errorf("invalid event ID query component ('%s')", component)
		}

		exclude := strings.HasPrefix(component, "-")
		body := component
		if exclude {
			body = strings.TrimSpace(strings.TrimPrefix(component, "-"))
		}

		rng, err := parseEventIDRange(body, component)
		if err != nil {
			return nil, nil, err
		}

		if exclude {
			excludes = append(excludes, rng)
		} else {
			includes = append(includes, rng)
		}
	}

	return includes, excludes, nil
}

func parseEventIDRange(expr, original string) (eventIDRange, error) {
	parts := strings.Split(expr, "-")
	switch len(parts) {
	case 1:
		v, err := parseEventID(parts[0], original)
		if err != nil {
			return eventIDRange{}, err
		}
		return eventIDRange{start: v, end: v}, nil
	case 2:
		start, err := parseEventID(parts[0], original)
		if err != nil {
			return eventIDRange{}, err
		}
		end, err := parseEventID(parts[1], original)
		if err != nil {
			return eventIDRange{}, err
		}
		if start >= end {
			return eventIDRange{}, fmt.Errorf("event ID range '%s' is invalid", original)
		}
		return eventIDRange{start: start, end: end}, nil
	default:
		return eventIDRange{}, fmt.Errorf("invalid event ID query component ('%s')", original)
	}
}

func parseEventID(raw, original string) (uint32, error) {
	raw = strings.TrimSpace(raw)
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid event ID query component ('%s')", original)
	}
	return uint32(v), nil
}
