// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"errors"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
)

func getCustomDataDirFilters(queryContext table.QueryContext) ([]string, error) {
	clist, ok := queryContext.Constraints["custom_data_dir"]
	if !ok || len(clist.Constraints) == 0 {
		return nil, nil
	}

	var results []string
	for _, c := range clist.Constraints {
		switch c.Operator {
		case table.OperatorEquals, table.OperatorGlob:
			results = append(results, c.Expression)
		case table.OperatorLike:
			// Convert SQL LIKE pattern to filepath.Match pattern
			pattern := strings.ReplaceAll(c.Expression, "%", "*")
			results = append(results, pattern)
		case table.OperatorRegexp:
			return nil, errors.New("regexp operator not supported for custom_data_dir")
		}
	}
	return results, nil
}

type filter struct {
	field string
	value string
}

func getConstraintFilters(queryContext table.QueryContext, fieldName string) []filter {
	clist, ok := queryContext.Constraints[fieldName]
	if !ok || len(clist.Constraints) == 0 {
		return nil
	}
	var results []filter
	for _, c := range clist.Constraints {
		switch c.Operator {
		case table.OperatorEquals, table.OperatorGlob, table.OperatorRegexp:
			results = append(results, filter{field: fieldName, value: c.Expression})
		case table.OperatorLike:
			// Convert SQL LIKE pattern to filepath.Match pattern
			pattern := strings.ReplaceAll(c.Expression, "%", "*")
			results = append(results, filter{field: fieldName, value: pattern})
		}
	}
	return results
}

type timestampConstraint struct {
	Operator table.Operator
	Value    int64 // Unix timestamp in seconds
}

func getTimestampConstraints(queryContext table.QueryContext) []timestampConstraint {
	clist, ok := queryContext.Constraints["timestamp"]
	if !ok || len(clist.Constraints) == 0 {
		return nil
	}

	constraints := getDatetimeConstraints(queryContext)
	for _, c := range clist.Constraints {
		// Parse and validate timestamp value
		osqueryTimestamp, err := strconv.ParseInt(c.Expression, 10, 64)
		if err != nil {
			continue // Skip invalid timestamp values
		}

		constraints = append(constraints, timestampConstraint{
			Operator: c.Operator,
			Value:    osqueryTimestamp,
		})
	}
	return constraints
}

func getDatetimeConstraints(queryContext table.QueryContext) []timestampConstraint {
	clist, ok := queryContext.Constraints["datetime"]
	if !ok || len(clist.Constraints) == 0 {
		return nil
	}
	var constraints []timestampConstraint
	for _, c := range clist.Constraints {
		t, err := time.Parse(time.RFC3339, c.Expression)
		if err != nil {
			continue
		}

		constraints = append(constraints, timestampConstraint{
			Operator: c.Operator,
			Value:    t.Unix(),
		})
	}
	return constraints
}

func matchesProfileFilters(profile *profile, filters []filter) bool {
	if !matchesFiltersForField("browser", profile.browser, filters) {
		return false
	}
	if !matchesFiltersForField("user", profile.user, filters) {
		return false
	}
	if !matchesFiltersForField("profile_name", profile.name, filters) {
		return false
	}
	return true
}

func matchesFiltersForField(field, value string, filters []filter) bool {
	var fieldFilters []filter
	for _, filter := range filters {
		if filter.field == field {
			fieldFilters = append(fieldFilters, filter)
		}
	}
	if len(fieldFilters) == 0 {
		return true
	}
	for _, filter := range fieldFilters {
		// Check for exact match
		if value == filter.value {
			return true
		}
		// Check for glob pattern match
		if matched, _ := filepath.Match(filter.value, value); matched {
			return true
		}
		// Check for regexp pattern match
		if re, err := regexp.Compile(filter.value); err == nil {
			if re.MatchString(value) {
				return true
			}
		}
	}
	return false
}
