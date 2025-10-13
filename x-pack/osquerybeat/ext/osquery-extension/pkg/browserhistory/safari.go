// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

var _ historyParser = &safariParser{}

type safariParser struct {
	browserName string
	profiles    []*profile
	log         func(m string, kvs ...any)
}

func newSafariParser(browserName, basePath string, log func(m string, kvs ...any)) historyParser {
	profiles := getSafariProfiles(basePath, log)
	if len(profiles) > 0 {
		return &safariParser{
			browserName: browserName,
			profiles:    profiles,
			log:         log,
		}
	}
	return nil
}

func (parser *safariParser) parse(ctx context.Context, queryContext table.QueryContext, profileFilters []string) ([]*visit, error) {
	var (
		merr   error
		visits []*visit
	)
	for _, profile := range parser.profiles {
		if len(profileFilters) > 0 && !matchesFilters(profile.name, profileFilters) {
			continue
		}
		vs, err := parser.parseProfile(ctx, queryContext, profile)
		if err != nil {
			merr = multierr.Append(merr, err)
			continue
		}
		visits = append(visits, vs...)
	}
	return visits, merr
}

func (parser *safariParser) parseProfile(ctx context.Context, queryContext table.QueryContext, profile *profile) ([]*visit, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", filepath.Join(profile.path, "History.db"))
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log("failed to open database", "error", err)
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

	parser.log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		parser.log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Safari history: %w", err)
	}
	defer rows.Close()

	var entries []*visit
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
			parser.log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entry := newVisit("safari", parser.browserName, profile.user, profile.name, profile.path, safariTimeToUnix(visitTime.Int64))
		entry.URL = url.String
		entry.Title = title.String
		entry.VisitID = visitID.Int64
		entry.VisitCount = int(visitCount.Int64)
		entry.UrlID = itemID.Int64
		entry.SfDomainExpansion = domainExpansion.String
		entry.SfLoadSuccessful = func(v int64) bool { return v != 0 }(loadSuccessful.Int64)

		entries = append(entries, entry)
	}

	parser.log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", profile.path)
	return entries, rows.Err()
}

func getSafariProfiles(basePath string, log func(m string, kvs ...any)) []*profile {
	// Modern Safari profiles: /Users/username/Library/Safari/Profiles/ProfileName
	// Legacy Safari: /Users/username/Library/Safari

	user := extractUserFromPath(basePath, log)

	var profiles []*profile

	entries, err := os.ReadDir(filepath.Join(basePath, "Profiles"))
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		historyPath := filepath.Join(basePath, "Profiles", entry.Name(), "History.db")
		if _, err := os.Stat(historyPath); err != nil {
			return nil
		}
		log("detected safari History.db file", "path", historyPath)
		profiles = append(profiles, &profile{
			name: entry.Name(),
			path: filepath.Dir(historyPath),
			user: user,
		})
	}
	if len(profiles) > 0 {
		return profiles
	}
	historyPath := filepath.Join(basePath, "History.db")
	if _, err := os.Stat(historyPath); err != nil {
		return nil
	}
	profiles = append(profiles, &profile{
		name: "Default",
		path: filepath.Dir(historyPath),
		user: user,
	})
	return profiles
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
