// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticbrowserhistory "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_browser_history"
)

var _ historyParser = &safariParser{}

type safariParser struct {
	location searchLocation
	profiles []*profile
	log      *logger.Logger
}

func newSafariParser(ctx context.Context, location searchLocation, log *logger.Logger) historyParser {
	profiles := getSafariProfiles(ctx, location, log)
	if len(profiles) > 0 {
		return &safariParser{
			location: location,
			profiles: profiles,
			log:      log,
		}
	}
	return nil
}

func inferSafariBrowserName(path string) string {
	normalized := filepath.ToSlash(path)
	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		lower := strings.ToLower(segment)
		if strings.Contains(lower, "safari") {
			return strings.ReplaceAll(lower, " ", "_")
		}
	}
	return "safari_custom"
}

func (parser *safariParser) parse(ctx context.Context, queryContext table.QueryContext, allFilters []filters.Filter) ([]elasticbrowserhistory.Result, error) {
	var (
		merr   error
		visits []elasticbrowserhistory.Result
	)
	for _, profile := range parser.profiles {
		// Check if profile matches the filters
		if !matchesProfile(profile, allFilters) {
			continue
		}
		vs, err := parser.parseProfile(ctx, queryContext, profile)
		if err != nil {
			merr = errors.Join(merr, err)
			continue
		}
		visits = append(visits, vs...)
	}
	return visits, merr
}

func (parser *safariParser) parseProfile(ctx context.Context, queryContext table.QueryContext, profile *profile) ([]elasticbrowserhistory.Result, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", profile.HistoryPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log.Errorf("failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open Safari history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere, params := buildSafariTimestampWhere(queryContext)

	query := `
	       SELECT 
		       hi.url,
		       hi.domain_expansion,
		       hv.title,
		       hv.visit_time,
		       hv.load_successful,
		       hi.id as item_id,
		       hv.id as visit_id
	       FROM history_items hi
	       LEFT JOIN history_visits hv ON hi.id = hv.history_item
	       WHERE hi.url IS NOT NULL` + timestampWhere + `
	       ORDER BY hv.visit_time DESC
       `

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		parser.log.Errorf("failed to execute query: %v", err)
		return nil, fmt.Errorf("failed to query Safari history: %w", err)
	}
	defer rows.Close()

	var entries []elasticbrowserhistory.Result
	rowCount := 0
	for rows.Next() {
		rowCount++

		var (
			url             sql.NullString
			domainExpansion sql.NullString
			title           sql.NullString
			visitTime       sql.NullFloat64
			loadSuccessful  sql.NullBool
			itemID          sql.NullInt64
			visitID         sql.NullInt64
		)

		err := rows.Scan(
			&url,
			&domainExpansion,
			&title,
			&visitTime,
			&loadSuccessful,
			&itemID,
			&visitID,
		)
		if err != nil {
			parser.log.Errorf("failed to scan row %d: %v", rowCount, err)
			continue
		}

		entry := newResult("safari", profile, safariTimeToUnix(visitTime.Float64))
		entry.Url = url.String
		entry.Title = title.String
		entry.Scheme, entry.Hostname, entry.Domain = extractSchemeHostAndTLDPPlusOne(url.String)
		entry.VisitId = visitID.Int64
		entry.UrlId = itemID.Int64
		entry.SfDomainExpansion = domainExpansion.String
		entry.SfLoadSuccessful = loadSuccessful.Bool

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func getSafariProfiles(ctx context.Context, location searchLocation, log *logger.Logger) []*profile {
	var profiles []*profile
	user := extractUserFromPath(location.path, log)

	// Recursively search for History.db files
	historyPaths := findFilesRecursively(location.path, "History.db", log)

	for _, historyPath := range historyPaths {
		profilePath := filepath.Dir(historyPath)
		profileName := resolveProfileName(ctx, profilePath)

		log.Infof("detected safari History.db file: %s", historyPath)

		profile := &profile{
			Name:        profileName,
			User:        user,
			Browser:     location.browser,
			ProfilePath: profilePath,
			HistoryPath: historyPath,
		}
		if location.isCustom {
			profile.Browser = inferSafariBrowserName(profile.ProfilePath)
			profile.CustomDataDir = location.path
		}
		profiles = append(profiles, profile)
	}

	return profiles
}

// resolveProfileName tries to extract the profile name from SafariTabs.db bookmarks table.
// Falls back to the base of the profile path.
func resolveProfileName(ctx context.Context, profilePath string) string {
	base := filepath.Base(profilePath)
	if tabsPath := findClosestSafariTabsDB(profilePath); tabsPath != "" {
		connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", tabsPath)
		db, err := sql.Open("sqlite3", connectionString)
		if err == nil {
			defer db.Close()
			row := db.QueryRowContext(ctx, "SELECT title FROM bookmarks WHERE external_uuid = ?", base)
			var title string
			if err := row.Scan(&title); err == nil && title != "" {
				return title
			}
		}
	}
	if strings.Contains(base, "Safari") {
		return "Default Profile"
	}
	return base
}

// findClosestSafariTabsDB recursively searches up the directory tree for SafariTabs.db
func findClosestSafariTabsDB(startPath string) string {
	dir := startPath
	for {
		tabsPath := filepath.Join(dir, "SafariTabs.db")
		if _, err := os.Stat(tabsPath); err == nil {
			return tabsPath
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// Unix timestamps are in seconds since January 1, 1970 UTC
// Safari timestamps are in seconds since January 1, 2001 UTC (Mac OS X epoch)
func safariTimeToUnix(time float64) int64 {
	if time == 0 {
		return 0
	}

	const epochOffset = 978307200
	return int64(time + epochOffset)
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
func buildSafariTimestampWhere(queryContext table.QueryContext) (string, []any) {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return "", nil
	}

	var conditions []string
	var params []any
	for _, constraint := range constraints {
		safariTime := unixToSafariTime(constraint.Value)
		const secondRange = 1

		switch constraint.Operator {
		case table.OperatorEquals:
			lower := safariTime
			upper := safariTime + secondRange
			conditions = append(conditions, "hv.visit_time >= ? AND hv.visit_time < ?")
			params = append(params, lower, upper)
		case table.OperatorGreaterThan:
			threshold := safariTime + secondRange
			conditions = append(conditions, "hv.visit_time >= ?")
			params = append(params, threshold)
		case table.OperatorLessThan:
			conditions = append(conditions, "hv.visit_time < ?")
			params = append(params, safariTime)
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, "hv.visit_time >= ?")
			params = append(params, safariTime)
		case table.OperatorLessThanOrEquals:
			upper := safariTime + secondRange
			conditions = append(conditions, "hv.visit_time < ?")
			params = append(params, upper)
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")", params
	}
	return "", nil
}
