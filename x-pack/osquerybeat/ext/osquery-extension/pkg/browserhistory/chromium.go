// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
	"go.uber.org/multierr"
)

var _ historyParser = &chromiumParser{}

type chromiumParser struct {
	browserName string
	profiles    []*profile
	log         func(m string, kvs ...any)
}

func newChromiumParser(browserName, basePath string, log func(m string, kvs ...any)) historyParser {
	profiles := getChromiumProfiles(basePath, log)
	if len(profiles) > 0 {
		return &chromiumParser{
			browserName: browserName,
			profiles:    profiles,
			log:         log,
		}
	}
	return nil
}

func (parser *chromiumParser) parse(ctx context.Context, queryContext table.QueryContext, profileFilters []string) ([]*visit, error) {
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

func (parser *chromiumParser) parseProfile(ctx context.Context, queryContext table.QueryContext, profile *profile) ([]*visit, error) {
	historyPath := filepath.Join(profile.path, "History")
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", historyPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		parser.log("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open Chromium history database: %w", err)
	}
	defer db.Close()

	// Build timestamp filtering
	timestampWhere := buildChromiumTimestampWhere(queryContext)

	query := fmt.Sprintf(`
		SELECT 
			urls.url,
			urls.title,
			urls.visit_count,
			urls.typed_count,
			urls.hidden,
			urls.id as url_id,
			visits.visit_time,
			visits.transition,
			visits.id as visit_id,
			visits.from_visit,
			visits.visit_duration,
			visit_source.source,
			ref_urls.url as referring_url
		FROM urls
		JOIN visits ON urls.id = visits.url
		LEFT JOIN visit_source ON visits.id = visit_source.id
		LEFT JOIN visits ref_visits ON visits.from_visit = ref_visits.id
		LEFT JOIN urls ref_urls ON ref_visits.url = ref_urls.id
		WHERE 1=1%s
		ORDER BY visits.visit_time DESC
	`, timestampWhere)

	parser.log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		parser.log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Chromium history: %w", err)
	}
	defer rows.Close()

	var entries []*visit
	rowCount := 0
	for rows.Next() {
		rowCount++

		var (
			url             sql.NullString
			title           sql.NullString
			visitCount      sql.NullInt64
			typedCount      sql.NullInt64
			isHidden        sql.NullInt64
			urlID           sql.NullInt64
			visitTime       sql.NullInt64
			transitionType  sql.NullInt64
			visitID         sql.NullInt64
			fromVisitID     sql.NullInt64
			chVisitDuration sql.NullInt64
			visitSource     sql.NullInt64
			referringURL    sql.NullString
		)

		err := rows.Scan(
			&url,
			&title,
			&visitCount,
			&typedCount,
			&isHidden,
			&urlID,
			&visitTime,
			&transitionType,
			&visitID,
			&fromVisitID,
			&chVisitDuration,
			&visitSource,
			&referringURL,
		)
		if err != nil {
			parser.log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entry := newVisit("chromium", parser.browserName, profile.user, profile.name, historyPath, chromiumTimeToUnix(visitTime.Int64))
		entry.URL = url.String
		entry.Title = title.String
		entry.TransitionType = mapChromiumTransitionType(transitionType)
		entry.ReferringURL = referringURL.String
		entry.VisitID = visitID.Int64
		entry.FromVisitID = fromVisitID.Int64
		entry.VisitCount = int(visitCount.Int64)
		entry.TypedCount = int(typedCount.Int64)
		entry.VisitSource = mapChromiumVisitSource(visitSource)
		entry.IsHidden = func(v int64) bool { return v != 0 }(isHidden.Int64)
		entry.UrlID = urlID.Int64
		entry.ChVisitDurationMs = chVisitDuration.Int64 / 1000

		entries = append(entries, entry)
	}

	parser.log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", historyPath)
	return entries, rows.Err()
}

type localState struct {
	Profile struct {
		InfoCache map[string]struct {
			Name string `json:"name"`
		} `json:"info_cache"`
	} `json:"profile"`
}

func getChromiumProfiles(basePath string, log func(m string, kvs ...any)) []*profile {
	var profiles []*profile

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		profilePath := filepath.Join(basePath, entry.Name())
		historyPath := filepath.Join(profilePath, "History")
		if _, err := os.Stat(historyPath); err != nil {
			continue
		}
		log("detected chromium History file", "path", historyPath)

		profile := &profile{
			name: entry.Name(),
			user: extractUserFromPath(profilePath, log),
			path: profilePath,
		}

		userDataDir := filepath.Dir(profilePath)
		localStatePath := filepath.Join(userDataDir, "Local State")
		data, err := os.ReadFile(localStatePath)
		if err != nil {
			profiles = append(profiles, profile)
			continue
		}

		var localState localState
		if err := json.Unmarshal(data, &localState); err == nil {
			if profileInfo, exists := localState.Profile.InfoCache[entry.Name()]; exists && profileInfo.Name != "" {
				profile.name = profileInfo.Name
			}
		}
		profiles = append(profiles, profile)
	}

	return profiles
}

// Unix timestamps are in seconds since January 1, 1970 UTC
// Chromium timestamps are in microseconds since January 1, 1601 UTC
func chromiumTimeToUnix(chromiumTime int64) int64 {
	if chromiumTime == 0 {
		return 0
	}
	// Difference between Jan 1, 1601 and Jan 1, 1970 in microseconds
	const epochDifference = 11644473600000000
	// Convert to seconds by dividing by 1,000,000 (microseconds to seconds)
	return (chromiumTime - epochDifference) / 1000000
}

func unixToChromiumTime(unixTime int64) int64 {
	if unixTime == 0 {
		return 0
	}
	// Difference between Jan 1, 1601 and Jan 1, 1970 in microseconds
	const epochDifference = 11644473600000000
	// Convert to microseconds and add epoch difference
	return (unixTime * 1000000) + epochDifference
}

// Chromium transition qualifiers (bit flags in upper bits)
const (
	transitionBlocked        = 0x00800000 // Navigation was blocked
	transitionForwardBack    = 0x01000000 // User used back/forward button
	transitionFromAddressBar = 0x02000000 // Navigation from address bar
	transitionHomePage       = 0x04000000 // Navigation to home page
	transitionFromAPI        = 0x08000000 // Navigation from browser API
	transitionChainStart     = 0x10000000 // Start of navigation chain
	transitionChainEnd       = 0x20000000 // End of navigation chain
	transitionClientRedirect = 0x40000000 // Client-side redirect
	transitionServerRedirect = 0x80000000 // Server-side redirect
)

// mapChromiumTransitionType maps Chromium transition types to human-readable strings
// Extracts both core transition type (lower 8 bits) and qualifiers (upper bits)
func mapChromiumTransitionType(transitionType sql.NullInt64) string {
	if !transitionType.Valid {
		return ""
	}

	value := transitionType.Int64
	// Extract core transition type (lower 8 bits)
	coreType := value & 0xFF

	// Extract qualifiers (upper bits)
	qualifiers := value & 0xFFFFFF00

	// Map core transition type
	var typeStr string
	switch coreType {
	case 0:
		typeStr = "LINK"
	case 1:
		typeStr = "TYPED"
	case 2:
		typeStr = "AUTO_BOOKMARK"
	case 3:
		typeStr = "AUTO_SUBFRAME"
	case 4:
		typeStr = "MANUAL_SUBFRAME"
	case 5:
		typeStr = "GENERATED"
	case 6:
		typeStr = "AUTO_TOPLEVEL"
	case 7:
		typeStr = "FORM_SUBMIT"
	case 8:
		typeStr = "RELOAD"
	case 9:
		typeStr = "KEYWORD"
	case 10:
		typeStr = "KEYWORD_GENERATED"
	default:
		typeStr = "UNKNOWN"
	}

	// Extract and append qualifiers for forensic analysis
	var quals []string
	if qualifiers&transitionBlocked != 0 {
		quals = append(quals, "BLOCKED")
	}
	if qualifiers&transitionForwardBack != 0 {
		quals = append(quals, "BACK_FORWARD")
	}
	if qualifiers&transitionFromAddressBar != 0 {
		quals = append(quals, "FROM_ADDRESS_BAR")
	}
	if qualifiers&transitionHomePage != 0 {
		quals = append(quals, "HOME_PAGE")
	}
	if qualifiers&transitionFromAPI != 0 {
		quals = append(quals, "FROM_API")
	}
	if qualifiers&transitionChainStart != 0 {
		quals = append(quals, "CHAIN_START")
	}
	if qualifiers&transitionChainEnd != 0 {
		quals = append(quals, "CHAIN_END")
	}
	if qualifiers&transitionClientRedirect != 0 {
		quals = append(quals, "CLIENT_REDIRECT")
	}
	if qualifiers&transitionServerRedirect != 0 {
		quals = append(quals, "SERVER_REDIRECT")
	}

	// Combine core type with qualifiers
	if len(quals) > 0 {
		return typeStr + "|" + strings.Join(quals, "|")
	}
	return typeStr
}

// mapChromiumVisitSource maps visit source ID to human-readable string
// Based on Chromium's VisitSource enum
func mapChromiumVisitSource(source sql.NullInt64) string {
	if !source.Valid {
		return "" // null
	}

	switch source.Int64 {
	case 0:
		return "synced" // SOURCE_SYNCED
	case 1:
		return "browsed" // SOURCE_BROWSED (local browsing)
	case 2:
		return "extension" // SOURCE_EXTENSION
	case 3:
		return "firefox_imported" // SOURCE_FIREFOX_IMPORTED
	case 4:
		return "ie_imported" // SOURCE_IE_IMPORTED
	case 5:
		return "safari_imported" // SOURCE_SAFARI_IMPORTED
	default:
		return "source_unknown"
	}
}

// buildChromiumTimestampWhere creates WHERE clause for Chromium-based browsers
func buildChromiumTimestampWhere(queryContext table.QueryContext) string {
	constraints := getTimestampConstraints(queryContext)
	if len(constraints) == 0 {
		return ""
	}

	var conditions []string
	for _, constraint := range constraints {
		chromiumTime := unixToChromiumTime(constraint.Value)

		switch constraint.Operator {
		case table.OperatorEquals:
			conditions = append(conditions, fmt.Sprintf("visits.visit_time = %d", chromiumTime))
		case table.OperatorGreaterThan:
			conditions = append(conditions, fmt.Sprintf("visits.visit_time > %d", chromiumTime))
		case table.OperatorLessThan:
			conditions = append(conditions, fmt.Sprintf("visits.visit_time < %d", chromiumTime))
		case table.OperatorGreaterThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("visits.visit_time >= %d", chromiumTime))
		case table.OperatorLessThanOrEquals:
			conditions = append(conditions, fmt.Sprintf("visits.visit_time <= %d", chromiumTime))
		}
	}

	if len(conditions) > 0 {
		return " AND (" + strings.Join(conditions, " AND ") + ")"
	}
	return ""
}
