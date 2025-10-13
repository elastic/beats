// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/osquery/osquery-go/plugin/table"
)

func getProfileNameFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "profile_name", nil)
}

func getUserFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "user", nil)
}

func getBrowserFilters(queryContext table.QueryContext) []string {
	return getConstraintFilters(queryContext, "browser", defaultBrowsers)
}

func getConstraintFilters(queryContext table.QueryContext, fieldName string, validateAgainst []string) []string {
	clist, ok := queryContext.Constraints[fieldName]
	if !ok || len(clist.Constraints) == 0 {
		return nil
	}

	var results []string
	for _, c := range clist.Constraints {
		switch c.Operator {
		case table.OperatorEquals:
			results = append(results, c.Expression)
		case table.OperatorLike:
			// Convert SQL LIKE pattern to filepath.Match pattern
			pattern := strings.ReplaceAll(c.Expression, "%", "*")
			if validateAgainst != nil {
				for _, item := range validateAgainst {
					if matched, _ := filepath.Match(pattern, item); matched {
						results = append(results, item)
					}
				}
			} else {
				results = append(results, pattern)
			}
		case table.OperatorGlob:
			if validateAgainst != nil {
				for _, item := range validateAgainst {
					if matched, _ := filepath.Match(c.Expression, item); matched {
						results = append(results, item)
					}
				}
			} else {
				results = append(results, c.Expression)
			}
		case table.OperatorRegexp:
			// Compile and validate regexp pattern
			re, err := regexp.Compile(c.Expression)
			if err != nil {
				continue
			}
			if validateAgainst != nil {
				for _, item := range validateAgainst {
					if re.MatchString(item) {
						results = append(results, item)
					}
				}
			} else {
				// We store the original expression since we'll need to recompile it later
				results = append(results, c.Expression)
			}
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

	var constraints []timestampConstraint
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

func matchesFilters(name string, filters []string) bool {
	for _, filter := range filters {
		// Check for exact match
		if name == filter {
			return true
		}
		// Check for glob pattern match
		if matched, _ := filepath.Match(filter, name); matched {
			return true
		}
		// Check for regexp pattern match
		if re, err := regexp.Compile(filter); err == nil {
			if re.MatchString(name) {
				return true
			}
		}
	}
	return false
}
