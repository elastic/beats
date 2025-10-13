// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

var _ historyParser = &firefoxParser{}

type firefoxParser struct {
	browserName string
	profiles    []*profile
	log         func(m string, kvs ...any)
}

func newFirefoxParser(browserName, basePath string, log func(m string, kvs ...any)) historyParser {
	var profiles []*profile

	// First, try to parse profiles.ini
	profilesIniPath := filepath.Join(basePath, "profiles.ini")
	if file, err := os.Open(profilesIniPath); err == nil {
		defer file.Close()
		profiles = getFirefoxProfiles(file, basePath, log)
		log("parsed profiles from profiles.ini", "count", len(profiles), "path", profilesIniPath)
	} else {
		log("profiles.ini not found, trying fallback", "path", profilesIniPath, "error", err)
		profiles = getFirefoxProfilesFallback(basePath, log)
	}

	if len(profiles) > 0 {
		return &firefoxParser{
			browserName: browserName,
			profiles:    profiles,
			log:         log,
		}
	}

	return nil
}

func (parser *firefoxParser) parse(ctx context.Context, queryContext table.QueryContext, profileFilters []string) ([]*visit, error) {
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

func (parser *firefoxParser) parseProfile(ctx context.Context, queryContext table.QueryContext, profile *profile) ([]*visit, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", profile.historyPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open Firefox history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere := buildFirefoxTimestampWhere(queryContext)

	query := fmt.Sprintf(`
		SELECT 
			p.url,
			p.title,
			p.visit_count,
			p.hidden,
			p.frecency,
			p.typed,
			p.id as place_id,
			hv.visit_date,
			hv.visit_type,
			hv.id as visit_id,
			hv.from_visit,
			hv.session,
			hv.source as visit_source,
			ref_p.url as referring_url
		FROM moz_places p
		JOIN moz_historyvisits hv ON p.id = hv.place_id
		LEFT JOIN moz_historyvisits ref_hv ON hv.from_visit = ref_hv.id
		LEFT JOIN moz_places ref_p ON ref_hv.place_id = ref_p.id
		WHERE p.visit_count > 0%s
		ORDER BY hv.visit_date DESC
	`, timestampWhere)

	parser.log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		parser.log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Firefox history: %w", err)
	}
	defer rows.Close()

	var entries []*visit
	rowCount := 0
	for rows.Next() {
		rowCount++

		var (
			url          sql.NullString
			title        sql.NullString
			visitCount   sql.NullInt64
			isHidden     sql.NullInt64
			ffFrecency   sql.NullInt64
			typedCount   sql.NullInt64
			urlID        sql.NullInt64
			visitTime    sql.NullInt64
			transition   sql.NullInt64
			visitID      sql.NullInt64
			fromVisitID  sql.NullInt64
			ffSessionID  sql.NullInt64
			visitSource  sql.NullInt64
			referringURL sql.NullString
		)

		err := rows.Scan(
			&url,
			&title,
			&visitCount,
			&isHidden,
			&ffFrecency,
			&typedCount,
			&urlID,
			&visitTime,
			&transition,
			&visitID,
			&fromVisitID,
			&ffSessionID,
			&visitSource,
			&referringURL,
		)
		if err != nil {
			parser.log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entry := newVisit("firefox", parser.browserName, profile, firefoxTimeToUnix(visitTime.Int64))
		entry.URL = url.String
		entry.Title = title.String
		entry.TransitionType = mapFirefoxTransitionType(transition)
		entry.ReferringURL = referringURL.String
		entry.VisitID = visitID.Int64
		entry.FromVisitID = fromVisitID.Int64
		entry.VisitCount = int(visitCount.Int64)
		entry.TypedCount = int(typedCount.Int64)
		entry.VisitSource = mapFirefoxVisitSource(visitSource)
		entry.IsHidden = func(v int64) bool { return v != 0 }(isHidden.Int64)
		entry.UrlID = urlID.Int64
		entry.FfSessionID = int(ffSessionID.Int64)
		entry.FfFrecency = int(ffFrecency.Int64)

		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func getFirefoxProfiles(file io.Reader, basePath string, log func(m string, kvs ...any)) []*profile {
	var profiles []*profile
	scanner := bufio.NewScanner(file)
	var currentProfile *profile
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[Profile") && strings.HasSuffix(line, "]") {
			currentProfile = &profile{}
		} else if currentProfile != nil {
			parts := strings.SplitN(line, "=", 2)
			switch parts[0] {
			case "Name":
				currentProfile.name = parts[1]
			case "Path":
				currentProfile.profilePath = parts[1]
				if !filepath.IsAbs(currentProfile.profilePath) {
					currentProfile.profilePath = filepath.Join(basePath, currentProfile.profilePath)
				}
			}
			if currentProfile.name != "" && currentProfile.profilePath != "" {
				historyPath := filepath.Join(currentProfile.profilePath, "places.sqlite")
				if _, err := os.Stat(historyPath); err != nil {
					continue
				}
				log("detected firefox places.sqlite file", "path", historyPath)
				profiles = append(profiles, &profile{
					name:        currentProfile.name,
					user:        extractUserFromPath(basePath, log),
					profilePath: currentProfile.profilePath,
					historyPath: historyPath,
					searchPath:  basePath,
				})
				currentProfile = nil
			}
		}
	}
	if len(profiles) > 0 {
		return profiles
	}
	return nil
}

// getFirefoxProfilesFallback scans the Profiles directory for Firefox profiles when profiles.ini is not available
func getFirefoxProfilesFallback(basePath string, log func(m string, kvs ...any)) []*profile {
	var profiles []*profile

	// Look for "Profiles" folder and scan its subdirectories
	profilesDir := filepath.Join(basePath, "Profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		log("Profiles directory not found", "path", profilesDir, "error", err)
		return nil
	}

	log("found Profiles directory, scanning subdirectories", "path", profilesDir)
	for _, entry := range entries {
		if entry.IsDir() {
			profilePath := filepath.Join(profilesDir, entry.Name())
			historyPath := filepath.Join(profilePath, "places.sqlite")

			if _, err := os.Stat(historyPath); err == nil {
				log("detected firefox places.sqlite file in fallback", "path", historyPath)
				profiles = append(profiles, &profile{
					name:        entry.Name(), // Use directory name as profile name
					user:        extractUserFromPath(basePath, log),
					profilePath: profilePath,
					historyPath: historyPath,
					searchPath:  basePath,
				})
			}
		}
	}

	log("fallback profile discovery complete", "count", len(profiles))
	return profiles
}

// Unix timestamps are in seconds since January 1, 1970 UTC
// Firefox timestamps are in microseconds since January 1, 1970 UTC
func firefoxTimeToUnix(firefoxTime int64) int64 {
	if firefoxTime == 0 {
		return 0
	}
	return firefoxTime / 1000000
}

func unixToFirefoxTime(unixTime int64) int64 {
	if unixTime == 0 {
		return 0
	}
	// Convert seconds to microseconds
	return unixTime * 1000000
}

// mapFirefoxVisitSource maps Firefox visit source values to human-readable strings
// Based on Firefox's actual source field from moz_historyvisits table
// Reference: Firefox forensics analysis and source code investigation
func mapFirefoxVisitSource(source sql.NullInt64) string {
	if !source.Valid {
		return "" // null
	}

	switch source.Int64 {
	case 0:
		return "source_organic" // Normal browsing/navigation
	case 1:
		return "source_imported" // Imported from another browser
	case 2:
		return "source_synced" // Firefox Sync
	case 3:
		return "source_temporary" // Temporary/private browsing artifacts
	default:
		return "source_unknown"
	}
}

// mapFirefoxTransitionType maps Firefox visit types to human-readable strings
// Firefox uses visit_type column with forensically-relevant categorization
// Reference: cyberengage.org Firefox forensics guide and Mozilla source code
func mapFirefoxTransitionType(transitionType sql.NullInt64) string {
	if !transitionType.Valid {
		return ""
	}

	switch transitionType.Int64 {
	case 1:
		return "LINK" // User clicked a link
	case 2:
		return "TYPED" // User typed URL (forensically significant)
	case 3:
		return "BOOKMARK" // From bookmark (indicates user intent)
	case 4:
		return "EMBED" // Embedded content
	case 5:
		return "REDIRECT_PERMANENT" // 301 redirect
	case 6:
		return "REDIRECT_TEMPORARY" // 302/307 redirect
	case 7:
		return "DOWNLOAD" // Download activity (forensically significant)
	case 8:
		return "FRAMED_LINK" // Link within iframe
	case 9:
		return "RELOAD" // Page reload
	default:
		return "UNKNOWN"
	}
}

// buildFirefoxTimestampWhere creates WHERE clause for Firefox
func buildFirefoxTimestampWhere(queryContext table.QueryContext) string {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return ""
	}

	var conditions []string
	for _, constraint := range constraints {
		firefoxTime := unixToFirefoxTime(constraint.Value)

		switch constraint.Operator {
		case table.OperatorEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_date = %d", firefoxTime))
		case table.OperatorGreaterThan:
			conditions = append(conditions, fmt.Sprintf("hv.visit_date > %d", firefoxTime))
		case table.OperatorLessThan:
			conditions = append(conditions, fmt.Sprintf("hv.visit_date < %d", firefoxTime))
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_date >= %d", firefoxTime))
		case table.OperatorLessThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("hv.visit_date <= %d", firefoxTime))
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")"
	}
	return ""
}
