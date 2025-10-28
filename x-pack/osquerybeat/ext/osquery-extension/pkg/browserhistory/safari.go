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
	"go.uber.org/multierr"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

var _ historyParser = &safariParser{}

type safariParser struct {
	location searchLocation
	profiles []*profile
	log      *logger.Logger
}

func newSafariParser(location searchLocation, log *logger.Logger) historyParser {
	profiles := getSafariProfiles(location, log)
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

func (parser *safariParser) parse(ctx context.Context, queryContext table.QueryContext, filters []filter) ([]*visit, error) {
	var (
		merr   error
		visits []*visit
	)
	for _, profile := range parser.profiles {
		if !matchesProfileFilters(profile, filters) {
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
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", profile.historyPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log.Errorf("failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open Safari history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere := buildSafariTimestampWhere(queryContext)

	query := fmt.Sprintf(`
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
		WHERE hi.url IS NOT NULL%s
		ORDER BY hv.visit_time DESC
	`, timestampWhere)

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		parser.log.Errorf("failed to execute query: %v", err)
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

		entry := newVisit("safari", profile, safariTimeToUnix(visitTime.Float64))
		entry.URL = url.String
		entry.Title = title.String
		entry.Scheme, entry.Domain = extractSchemeAndDomain(url.String)
		entry.VisitID = visitID.Int64
		entry.UrlID = itemID.Int64
		entry.SfDomainExpansion = domainExpansion.String
		entry.SfLoadSuccessful = loadSuccessful.Bool

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

func getSafariProfiles(location searchLocation, log *logger.Logger) []*profile {
	var profiles []*profile
	user := extractUserFromPath(location.path, log)

	// Recursively search for History.db files
	historyPaths := findFilesRecursively(location.path, "History.db", log)

	for _, historyPath := range historyPaths {
		profilePath := filepath.Dir(historyPath)
		profileName := filepath.Base(profilePath)

		// If the profile name is "Safari", use "Default" instead
		if profileName == "Safari" {
			profileName = "Default"
		}

		log.Infof("detected safari History.db file: %s", historyPath)

		profile := &profile{
			name:        profileName,
			user:        user,
			browser:     location.browser,
			profilePath: profilePath,
			historyPath: historyPath,
		}
		if location.isCustom {
			profile.browser = inferSafariBrowserName(profile.profilePath)
			profile.customDataDir = location.path
		}
		profiles = append(profiles, profile)
	}

	return profiles
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
func buildSafariTimestampWhere(queryContext table.QueryContext) string {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return ""
	}

	var conditions []string
	for _, constraint := range constraints {
		safariTime := unixToSafariTime(constraint.Value)
		const secondRange = 1

		switch constraint.Operator {
		case table.OperatorEquals:
			lower := safariTime
			upper := safariTime + secondRange
			conditions = append(conditions, fmt.Sprintf("hv.visit_time >= %d AND hv.visit_time < %d", lower, upper))
		case table.OperatorGreaterThan:
			threshold := safariTime + secondRange
			conditions = append(conditions, fmt.Sprintf("hv.visit_time >= %d", threshold))
		case table.OperatorLessThan:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time < %d", safariTime))
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_time >= %d", safariTime))
		case table.OperatorLessThanOrEquals:
			upper := safariTime + secondRange
			conditions = append(conditions, fmt.Sprintf("hv.visit_time < %d", upper))
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")"
	}
	return ""
}
