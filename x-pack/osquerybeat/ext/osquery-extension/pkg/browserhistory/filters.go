// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
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

// matchesProfile checks if a profile matches the given filters
func matchesProfile(profile *profile, allFilters []filters.Filter) bool {
	for _, filter := range allFilters {
		// Only check profile-related filters
		switch filter.ColumnName {
		case "browser", "user", "profile_name":
			if !filter.Matches(profile) {
				return false
			}
		}
	}
	return true
}
