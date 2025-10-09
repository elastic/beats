// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
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

	query := `
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
		WHERE hi.url IS NOT NULL
		ORDER BY hv.visit_time DESC
	`

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
		entry := rawHistoryEntry{
			user:        user,
			profile:     profileName,
			path:        profilePath,
			browserName: browserName,
		}

		err := rows.Scan(
			&entry.url,
			&entry.sfDomainExpansion,
			&entry.visitCount,
			&entry.title,
			&entry.visitTime,
			&entry.sfLoadSuccessful,
			&entry.urlID,
			&entry.visitID,
		)
		if err != nil {
			log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entries = append(entries, safariEntryToRow(&entry))
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

func safariEntryToRow(entry *rawHistoryEntry) *row {
	return &row{
		Timestamp: formatNullInt64(entry.visitTime, func(value int64) string {
			return strconv.FormatInt(safariTimeToUnix(value), 10)
		}),
		URL:               stringFromNullString(entry.url),
		Title:             stringFromNullString(entry.title),
		Browser:           entry.browserName,
		Parser:            "safari",
		User:              entry.user,
		ProfileName:       entry.profile,
		VisitID:           decimalStringFromNullInt(entry.visitID),
		VisitCount:        decimalStringFromNullInt(entry.visitCount),
		SourcePath:        entry.path,
		UrlID:             decimalStringFromNullInt(entry.urlID),
		SfDomainExpansion: stringFromNullString(entry.sfDomainExpansion),
		SfLoadSuccessful:  boolStringFromNullInt(entry.sfLoadSuccessful),
	}
}

// safariTimeToUnix converts Safari timestamp to Unix timestamp
func safariTimeToUnix(time int64) int64 {
	if time == 0 {
		return 0
	}
	// Safari uses seconds since 2001-01-01 00:00:00 UTC (Mac OS X epoch)
	const epochOffset = 978307200
	return time + epochOffset
}
