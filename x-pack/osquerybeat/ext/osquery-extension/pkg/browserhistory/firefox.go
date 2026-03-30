// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/filters"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
	elasticbrowserhistory "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_browser_history"
)

var _ historyParser = &firefoxParser{}

type firefoxParser struct {
	location searchLocation
	profiles []*profile
	log      *logger.Logger
}

func newFirefoxParser(_ context.Context, location searchLocation, log *logger.Logger) historyParser {
	var profiles []*profile

	// First, recursively search for profiles.ini files
	profilesIniPaths := findFilesRecursively(location.path, "profiles.ini", log)

	for _, profilesIniPath := range profilesIniPaths {
		if file, err := os.Open(profilesIniPath); err == nil {
			defer file.Close()
			basePath := filepath.Dir(profilesIniPath)
			foundProfiles := getFirefoxProfiles(file, basePath, location, log)
			profiles = append(profiles, foundProfiles...)
			log.Infof("parsed profiles from profiles.ini: %d (path: %s)", len(foundProfiles), profilesIniPath)
		}
	}

	// If no profiles.ini found, try fallback method
	if len(profiles) == 0 {
		profiles = getFirefoxProfilesFallback(location, log)
	}

	if len(profiles) > 0 {
		return &firefoxParser{
			location: location,
			profiles: profiles,
			log:      log,
		}
	}

	return nil
}

func inferFirefoxBrowserName(path string) string {
	normalized := filepath.ToSlash(path)
	segments := strings.Split(normalized, "/")
	for i, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		lower := strings.ToLower(segment)
		if lower == "profiles" && i > 0 {
			return strings.ReplaceAll(strings.ToLower(segments[i-1]), " ", "_")
		}
	}
	return "firefox_custom"
}

func (parser *firefoxParser) parse(ctx context.Context, queryContext table.QueryContext, allFilters []filters.Filter) ([]elasticbrowserhistory.Result, error) {
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

func (parser *firefoxParser) parseProfile(ctx context.Context, queryContext table.QueryContext, profile *profile) ([]elasticbrowserhistory.Result, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", profile.HistoryPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log.Errorf("failed to open database: %v", err)
		return nil, fmt.Errorf("failed to open Firefox history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere, params := buildFirefoxTimestampWhere(queryContext)

	query := `
	       SELECT 
		       p.url,
		       p.title,
		       p.hidden,
		       p.frecency,
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
	       WHERE 1=1` + timestampWhere + `
	       ORDER BY hv.visit_date DESC
       `

	rows, err := db.QueryContext(ctx, query, params...)
	if err != nil {
		parser.log.Errorf("failed to execute query: %v", err)
		return nil, fmt.Errorf("failed to query Firefox history: %w", err)
	}
	defer rows.Close()

	var entries []elasticbrowserhistory.Result
	rowCount := 0
	for rows.Next() {
		rowCount++

		var (
			url          sql.NullString
			title        sql.NullString
			isHidden     sql.NullInt64
			ffFrecency   sql.NullInt64
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
			&isHidden,
			&ffFrecency,
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
			parser.log.Errorf("failed to scan row %d: %v", rowCount, err)
			continue
		}

		entry := newResult("firefox", profile, firefoxTimeToUnix(visitTime.Int64))
		entry.Url = url.String
		entry.Title = title.String
		entry.Scheme, entry.Hostname, entry.Domain = extractSchemeHostAndTLDPPlusOne(url.String)
		entry.TransitionType = mapFirefoxTransitionType(transition)
		entry.ReferringUrl = referringURL.String
		entry.VisitId = visitID.Int64
		entry.FromVisitId = fromVisitID.Int64
		entry.VisitSource = mapFirefoxVisitSource(visitSource)
		entry.IsHidden = func(v int64) bool { return v != 0 }(isHidden.Int64)
		entry.UrlId = urlID.Int64
		entry.FfSessionId = int32(ffSessionID.Int64)
		entry.FfFrecency = int32(ffFrecency.Int64)

		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func getFirefoxProfiles(file io.Reader, basePath string, location searchLocation, log *logger.Logger) []*profile {
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
				currentProfile.Name = parts[1]
			case "Path":
				currentProfile.ProfilePath = parts[1]
				if !filepath.IsAbs(currentProfile.ProfilePath) {
					currentProfile.ProfilePath = filepath.Join(basePath, currentProfile.ProfilePath)
				}
			}
			if currentProfile.Name != "" && currentProfile.ProfilePath != "" {
				historyPath := filepath.Join(currentProfile.ProfilePath, "places.sqlite")
				if _, err := os.Stat(historyPath); err != nil {
					continue
				}
				log.Infof("detected firefox places.sqlite file: %s", historyPath)
				profile := &profile{
					Name:        currentProfile.Name,
					User:        extractUserFromPath(basePath, log),
					Browser:     location.browser,
					ProfilePath: currentProfile.ProfilePath,
					HistoryPath: historyPath,
				}
				if location.isCustom {
					profile.Browser = inferFirefoxBrowserName(profile.ProfilePath)
					profile.CustomDataDir = location.path
				}
				profiles = append(profiles, profile)
				currentProfile = nil
			}
		}
	}
	if len(profiles) > 0 {
		return profiles
	}
	return nil
}

// getFirefoxProfilesFallback recursively searches for places.sqlite files when profiles.ini is not available
func getFirefoxProfilesFallback(location searchLocation, log *logger.Logger) []*profile {
	var profiles []*profile
	user := extractUserFromPath(location.path, log)

	// Recursively search for places.sqlite files
	placesPaths := findFilesRecursively(location.path, "places.sqlite", log)

	for _, placesPath := range placesPaths {
		profilePath := filepath.Dir(placesPath)
		profileName := filepath.Base(profilePath)

		log.Infof("detected firefox places.sqlite file in fallback: %s", placesPath)

		profile := &profile{
			Name:        profileName,
			User:        user,
			Browser:     location.browser,
			ProfilePath: profilePath,
			HistoryPath: placesPath,
		}
		if location.isCustom {
			profile.Browser = inferFirefoxBrowserName(profile.ProfilePath)
			profile.CustomDataDir = location.path
		}
		profiles = append(profiles, profile)
	}
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
func buildFirefoxTimestampWhere(queryContext table.QueryContext) (string, []any) {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return "", nil
	}

	var conditions []string
	var params []any
	for _, constraint := range constraints {
		firefoxTime := unixToFirefoxTime(constraint.Value)
		const microsPerSecond = 1000000

		switch constraint.Operator {
		case table.OperatorEquals:
			lower := firefoxTime
			upper := firefoxTime + microsPerSecond
			conditions = append(conditions, "hv.visit_date >= ? AND hv.visit_date < ?")
			params = append(params, lower, upper)
		case table.OperatorGreaterThan:
			threshold := firefoxTime + microsPerSecond
			conditions = append(conditions, "hv.visit_date >= ?")
			params = append(params, threshold)
		case table.OperatorLessThan:
			conditions = append(conditions, "hv.visit_date < ?")
			params = append(params, firefoxTime)
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, "hv.visit_date >= ?")
			params = append(params, firefoxTime)
		case table.OperatorLessThanOrEquals:
			upper := firefoxTime + microsPerSecond
			conditions = append(conditions, "hv.visit_date < ?")
			params = append(params, upper)
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")", params
	}
	return "", nil
}
