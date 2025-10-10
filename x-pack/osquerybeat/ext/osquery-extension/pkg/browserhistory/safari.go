// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

func safariParser(ctx context.Context, queryContext table.QueryContext, browserName, profilePath string, log func(m string, kvs ...any)) ([]*row, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", filepath.Join(profilePath, "History.db"))
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		log("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open Safari history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere := buildSafariTimestampWhere(queryContext)

	query := fmt.Sprintf(`
		SELECT 
			hi.url,
			hi.domain_expansion,
			hi.visit_count,
			hv.title,
			hv.visit_time,
			hv.load_successful,
			hi.id as item_id,
			hv.id as visit_id
		FROM history_items hi
		LEFT JOIN history_visits hv ON hi.id = hv.history_item
		WHERE hi.url IS NOT NULL%s
		ORDER BY hv.visit_time DESC
	`, timestampWhere)

	log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Safari history: %w", err)
	}
	defer rows.Close()

	user := extractUserFromPath(profilePath, log)
	profileName := extractSafariProfileName(profilePath, log)

	var entries []*row
	rowCount := 0
	for rows.Next() {
		rowCount++

		var (
			url             sql.NullString
			domainExpansion sql.NullString
			visitCount      sql.NullInt64
			title           sql.NullString
			visitTime       sql.NullInt64
			loadSuccessful  sql.NullInt64
			itemID          sql.NullInt64
			visitID         sql.NullInt64
		)

		err := rows.Scan(
			&url,
			&domainExpansion,
			&visitCount,
			&title,
			&visitTime,
			&loadSuccessful,
			&itemID,
			&visitID,
		)
		if err != nil {
			log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entry := newHistoryRow("safari", browserName, user, profileName, profilePath, safariTimeToUnix(visitTime.Int64))
		entry.URL = stringFromNullString(url)
		entry.Title = stringFromNullString(title)
		entry.VisitID = decimalStringFromNullInt(visitID)
		entry.VisitCount = decimalStringFromNullInt(visitCount)
		entry.UrlID = decimalStringFromNullInt(itemID)
		entry.SfDomainExpansion = stringFromNullString(domainExpansion)
		entry.SfLoadSuccessful = boolStringFromNullInt(loadSuccessful)

		entries = append(entries, entry)
	}

	log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", profilePath)
	return entries, rows.Err()
}

func extractSafariProfileName(profilePath string, log func(m string, kvs ...any)) string {
	// Modern Safari profiles: /Users/username/Library/Safari/Profiles/ProfileName
	// Legacy Safari: /Users/username/Library/Safari

	// Normalize path separators
	normalizedPath := filepath.ToSlash(profilePath)
	parts := strings.Split(normalizedPath, "/")

	// Look for the Profiles directory in the path
	for i, part := range parts {
		if part == "Profiles" && i+1 < len(parts) {
			profileName := parts[i+1]
			log("extracted Safari profile name from Profiles directory", "name", profileName)
			return profileName
		}
	}

	log("using default Safari profile name")
	return "Default"
}

// Unix timestamps are in seconds since January 1, 1970 UTC
// Safari timestamps are in seconds since January 1, 2001 UTC (Mac OS X epoch)
func safariTimeToUnix(time int64) int64 {
	if time == 0 {
		return 0
	}

	const epochOffset = 978307200
	return time + epochOffset
}

func unixToSafariTime(unixTime int64) int64 {
	if unixTime == 0 {
		return 0
	}
	// Convert from Unix epoch (1970) to Mac OS X epoch (2001)
	const epochOffset = 978307200
	return unixTime - epochOffset
}

// buildSafariTimestampWhere creates WHERE clause for Safari
func buildSafariTimestampWhere(queryContext table.QueryContext) string {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return ""
	}

	var conditions []string
	for _, constraint := range constraints {
		safariTime := unixToSafariTime(constraint.Value)

		switch constraint.Operator {
		case table.OperatorEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time = %d", safariTime))
		case table.OperatorGreaterThan:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time > %d", safariTime))
		case table.OperatorLessThan:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time < %d", safariTime))
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time >= %d", safariTime))
		case table.OperatorLessThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time <= %d", safariTime))
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")"
	}
	return ""
}
