// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"fmt"
	"regexp"
	"strings"
)

// CursorPlaceholder is the user-facing placeholder in SQL queries.
// Users write :cursor in their queries, and it gets translated to
// the appropriate database-specific placeholder.
const CursorPlaceholder = ":cursor"

// cursorRegex matches the :cursor placeholder with word boundary.
// This prevents matching :cursor_value or similar.
var cursorRegex = regexp.MustCompile(`:cursor\b`)

// ValidateQueryHasCursor checks if the query contains exactly one cursor placeholder.
// Returns an error if:
//   - No :cursor placeholder is found
//   - More than one :cursor placeholder is found
func ValidateQueryHasCursor(query string) error {
	matches := cursorRegex.FindAllString(query, -1)
	if len(matches) == 0 {
		return fmt.Errorf("query must contain %s placeholder when cursor is enabled", CursorPlaceholder)
	}
	if len(matches) > 1 {
		return fmt.Errorf("query must contain exactly one %s placeholder, found %d", CursorPlaceholder, len(matches))
	}
	return nil
}

// TranslateQuery replaces :cursor with the driver-specific placeholder.
// The cursor value is passed as a parameterized query argument (SQL injection safe).
//
// Driver placeholder mapping:
//   - PostgreSQL, CockroachDB: $1
//   - MySQL: ?
//   - Oracle: :cursor_val
//   - MSSQL: @p1
func TranslateQuery(query, driver string) string {
	placeholder := getDriverPlaceholder(driver)
	return cursorRegex.ReplaceAllLiteralString(query, placeholder)
}

// getDriverPlaceholder returns the appropriate placeholder syntax for the given driver.
func getDriverPlaceholder(driver string) string {
	switch strings.ToLower(driver) {
	case "postgres", "postgresql", "cockroachdb", "cockroach":
		return "$1"
	case "mysql":
		return "?"
	case "oracle", "godror":
		return ":cursor_val"
	case "mssql", "sqlserver":
		return "@p1"
	default:
		// Default to positional placeholder (works for most databases)
		return "?"
	}
}

// CountPlaceholders returns the number of :cursor placeholders in the query.
// This is useful for validation and debugging.
func CountPlaceholders(query string) int {
	return len(cursorRegex.FindAllString(query, -1))
}
