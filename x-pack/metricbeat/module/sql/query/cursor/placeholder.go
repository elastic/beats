// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cursor

import (
	"fmt"
	"strings"
)

// CursorPlaceholder is the user-facing placeholder in SQL queries.
// Users write :cursor in their queries, and it gets translated to
// the appropriate database-specific placeholder.
const CursorPlaceholder = ":cursor"

// ValidateQueryHasCursor checks if the query contains exactly one cursor
// placeholder in executable SQL (outside strings and comments).
// Returns an error if no placeholder is found or more than one is found.
func ValidateQueryHasCursor(query string) error {
	count := CountPlaceholders(query)
	if count == 0 {
		return fmt.Errorf("query must contain %s placeholder when cursor is enabled", CursorPlaceholder)
	}
	if count > 1 {
		return fmt.Errorf("query must contain exactly one %s placeholder, found %d", CursorPlaceholder, count)
	}
	return nil
}

// TranslateQuery replaces the :cursor placeholder in executable SQL with the
// driver-specific parameterized placeholder. Occurrences inside quoted strings,
// identifiers, and SQL comments are left untouched.
//
// Driver placeholder mapping:
//   - PostgreSQL, CockroachDB: $1
//   - MySQL: ?
//   - Oracle: :cursor_val
//   - MSSQL: @p1
func TranslateQuery(query, driver string) string {
	placeholder := getDriverPlaceholder(driver)
	positions := findPlaceholderPositions(query)
	if len(positions) == 0 {
		return query
	}

	// Pre-size: original length adjusted for each replacement.
	sizeDelta := len(placeholder) - len(CursorPlaceholder)
	var b strings.Builder
	b.Grow(len(query) + sizeDelta*len(positions))

	last := 0
	for _, pos := range positions {
		b.WriteString(query[last:pos])
		b.WriteString(placeholder)
		last = pos + len(CursorPlaceholder)
	}
	b.WriteString(query[last:])
	return b.String()
}

// getDriverPlaceholder returns the parameterized placeholder syntax for the
// given driver name.
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
		return "?"
	}
}

// CountPlaceholders returns the number of :cursor placeholders in executable
// SQL (outside quoted strings and comments).
func CountPlaceholders(query string) int {
	return len(findPlaceholderPositions(query))
}

// Scanner states for findPlaceholderPositions.
const (
	stateNormal = iota
	stateSingleQuote
	stateDoubleQuote
	stateBacktick
	stateLineComment
	stateBlockComment
)

// findPlaceholderPositions returns the byte offsets of every :cursor
// placeholder that appears in executable SQL. The scanner skips content
// inside:
//   - Single-quoted strings ('...'), with '' escape handling
//   - Double-quoted identifiers ("..."), with "" escape handling
//   - Backtick-quoted identifiers (`...`), with `` escape handling
//   - Line comments (-- ...)
//   - Block comments (/* ... */)
//
// Limitation: MySQL's default sql_mode allows backslash escapes inside
// strings (e.g., 'it\'s :cursor'). This scanner does not handle backslash
// escapes and would incorrectly treat the :cursor inside such a string as
// a real placeholder. This is acceptable because operator-written queries
// rarely use backslash escapes, and MySQL's NO_BACKSLASH_ESCAPES mode
// disables them entirely. The standard SQL escape (doubled quotes:
// 'it''s :cursor') is handled correctly.
func findPlaceholderPositions(query string) []int {
	positions := make([]int, 0, 1)
	n := len(query)
	state := stateNormal

	for i := 0; i < n; i++ {
		ch := query[i]

		switch state {
		case stateNormal:
			if ch == '-' && i+1 < n && query[i+1] == '-' {
				state = stateLineComment
				i++
			} else if ch == '/' && i+1 < n && query[i+1] == '*' {
				state = stateBlockComment
				i++
			} else if ch == '\'' {
				state = stateSingleQuote
			} else if ch == '"' {
				state = stateDoubleQuote
			} else if ch == '`' {
				state = stateBacktick
			} else if ch == ':' && matchesPlaceholder(query, i, n) {
				positions = append(positions, i)
				i += len(CursorPlaceholder) - 1
			}

		case stateSingleQuote:
			if ch == '\'' {
				if i+1 < n && query[i+1] == '\'' {
					i++ // skip escaped quote
				} else {
					state = stateNormal
				}
			}

		case stateDoubleQuote:
			if ch == '"' {
				if i+1 < n && query[i+1] == '"' {
					i++ // skip escaped quote
				} else {
					state = stateNormal
				}
			}

		case stateBacktick:
			if ch == '`' {
				if i+1 < n && query[i+1] == '`' {
					i++ // skip escaped backtick
				} else {
					state = stateNormal
				}
			}

		case stateLineComment:
			if ch == '\n' || ch == '\r' {
				state = stateNormal
			}

		case stateBlockComment:
			if ch == '*' && i+1 < n && query[i+1] == '/' {
				state = stateNormal
				i++
			}
		}
	}

	return positions
}

// matchesPlaceholder reports whether query[i:] starts with ":cursor"
// followed by a non-word character (or end of string). The caller must
// ensure query[i] == ':' before calling.
func matchesPlaceholder(query string, i, n int) bool {
	end := i + len(CursorPlaceholder)
	if end > n {
		return false
	}
	if query[i:end] != CursorPlaceholder {
		return false
	}
	if end == n {
		return true
	}
	return !isWordChar(query[end])
}

// isWordChar reports whether ch is an ASCII letter, digit, or underscore.
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}
