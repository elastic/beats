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
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/osquery/osquery-go/plugin/table"
)

type chromiumHistoryEntry struct {
	url               string
	title             string
	visitCount        int64
	typedCount        int64
	lastVisitTime     int64
	transitionType    int64
	visitChainID      int64
	priorVisitChainID int64
	visitDuration     int64
	referringURL      string
	path              string
	browserName       string
}

func chromiumParser(ctx context.Context, queryContext table.QueryContext, browser, pathPattern string, log func(m string, kvs ...any)) ([]map[string]string, error) {
	var results []map[string]string

	var merr *multierror.Error
	for _, profilePath := range expandProfilePaths(pathPattern, log) {
		log("processing profile", "browser", browser, "path", profilePath)

		entries, err := readChromiumHistoryFromPath(ctx, profilePath, browser, log)
		if err != nil {
			err = fmt.Errorf("failed to read history: %w", err)
			log(err.Error(), "path", profilePath)
			merr = multierror.Append(merr, err)
			continue
		}

		for _, entry := range entries {
			row := chromiumEntryToRow(entry)
			results = append(results, row)
		}
	}

	return results, nil
}

// expandProfilePaths expands the {profile} placeholder to find all available profiles
func expandProfilePaths(pathPattern string, log func(m string, kvs ...any)) []string {
	var profilePaths []string

	if !strings.Contains(pathPattern, "{profile}") {
		return []string{pathPattern}
	}

	globPattern := strings.ReplaceAll(pathPattern, "{profile}", "*")
	if matches, err := filepath.Glob(globPattern); err == nil {
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil && !info.IsDir() {
				log("valid profile file found", "path", match)
				profilePaths = append(profilePaths, match)
			} else if err != nil {
				log("error stating file", "path", match, "error", err)
			} else {
				log("skipping directory", "path", match)
			}
		}
	} else {
		log("glob pattern failed", "error", err)
	}

	return profilePaths
}

func readChromiumHistoryFromPath(ctx context.Context, historyPath, browserName string, log func(m string, kvs ...any)) ([]chromiumHistoryEntry, error) {
	// Open database as read-only with no lock
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", historyPath)
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		log("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open Chromium history database: %w", err)
	}
	defer db.Close()

	query := `
		SELECT 
			urls.url,
			urls.title,
			urls.visit_count,
			urls.typed_count,
			urls.last_visit_time,
			visits.transition,
			visits.visit_time,
			visits.from_visit,
			visits.visit_duration,
			ref_urls.url as referring_url
		FROM urls
		LEFT JOIN visits ON urls.id = visits.url
		LEFT JOIN urls ref_urls ON visits.from_visit = ref_urls.id
		ORDER BY visits.visit_time DESC
	`

	log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Chromium history: %w", err)
	}
	defer rows.Close()

	var entries []chromiumHistoryEntry
	rowCount := 0
	for rows.Next() {
		rowCount++
		var entry chromiumHistoryEntry
		var referringURL sql.NullString
		var transition sql.NullInt64
		var visitTime sql.NullInt64
		var fromVisit sql.NullInt64
		var visitDuration sql.NullInt64

		err := rows.Scan(
			&entry.url,
			&entry.title,
			&entry.visitCount,
			&entry.typedCount,
			&entry.lastVisitTime,
			&transition,
			&visitTime,
			&fromVisit,
			&visitDuration,
			&referringURL,
		)
		if err != nil {
			log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}
		if referringURL.Valid {
			entry.referringURL = referringURL.String
		}
		if transition.Valid {
			entry.transitionType = transition.Int64
		}
		if visitTime.Valid {
			entry.visitChainID = visitTime.Int64
		}
		if fromVisit.Valid {
			entry.priorVisitChainID = fromVisit.Int64
		}
		if visitDuration.Valid {
			entry.visitDuration = visitDuration.Int64
		}

		entry.path = historyPath
		entry.browserName = browserName

		entries = append(entries, entry)
	}

	log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", historyPath)
	return entries, rows.Err()
}

// chromiumEntryToRow converts a Chromium history entry to the standardized row format
func chromiumEntryToRow(entry chromiumHistoryEntry) map[string]string {
	// Convert Chromium timestamp (microseconds since Jan 1, 1601) to Unix timestamp
	unixTimestamp := chromiumTimeToUnix(entry.lastVisitTime)

	// Map Chromium transition types to human-readable strings
	transitionType := mapChromiumTransitionType(entry.transitionType)

	return map[string]string{
		"timestamp":            strconv.FormatInt(unixTimestamp, 10),
		"url":                  entry.url,
		"title":                entry.title,
		"browser":              entry.browserName,
		"transition_type":      transitionType,
		"referring_url":        entry.referringURL,
		"visit_chain_id":       strconv.FormatInt(entry.visitChainID, 10),
		"prior_visit_chain_id": strconv.FormatInt(entry.priorVisitChainID, 10),
		"visit_duration_ms":    strconv.FormatInt(entry.visitDuration/1000, 10), // Convert microseconds to milliseconds
		"visit_count":          strconv.FormatInt(entry.visitCount, 10),
		"typed_count":          strconv.FormatInt(entry.typedCount, 10),
		"source_path":          entry.path,
	}
}

// chromiumTimeToUnix converts Chromium timestamp to Unix timestamp
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

// Chromium transition qualifiers (bit flags in upper bits)
const (
	TransitionBlocked        = 0x00800000 // Navigation was blocked
	TransitionForwardBack    = 0x01000000 // User used back/forward button
	TransitionFromAddressBar = 0x02000000 // Navigation from address bar
	TransitionHomePage       = 0x04000000 // Navigation to home page
	TransitionFromAPI        = 0x08000000 // Navigation from browser API
	TransitionChainStart     = 0x10000000 // Start of navigation chain
	TransitionChainEnd       = 0x20000000 // End of navigation chain
	TransitionClientRedirect = 0x40000000 // Client-side redirect
	TransitionServerRedirect = 0x80000000 // Server-side redirect
)

// mapChromiumTransitionType maps Chromium transition types to human-readable strings
// Extracts both core transition type (lower 8 bits) and qualifiers (upper bits)
func mapChromiumTransitionType(transitionType int64) string {
	// Extract core transition type (lower 8 bits)
	coreType := transitionType & 0xFF

	// Extract qualifiers (upper bits)
	qualifiers := transitionType & 0xFFFFFF00

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
	if qualifiers&TransitionBlocked != 0 {
		quals = append(quals, "BLOCKED")
	}
	if qualifiers&TransitionForwardBack != 0 {
		quals = append(quals, "BACK_FORWARD")
	}
	if qualifiers&TransitionFromAddressBar != 0 {
		quals = append(quals, "FROM_ADDRESS_BAR")
	}
	if qualifiers&TransitionHomePage != 0 {
		quals = append(quals, "HOME_PAGE")
	}
	if qualifiers&TransitionFromAPI != 0 {
		quals = append(quals, "FROM_API")
	}
	if qualifiers&TransitionChainStart != 0 {
		quals = append(quals, "CHAIN_START")
	}
	if qualifiers&TransitionChainEnd != 0 {
		quals = append(quals, "CHAIN_END")
	}
	if qualifiers&TransitionClientRedirect != 0 {
		quals = append(quals, "CLIENT_REDIRECT")
	}
	if qualifiers&TransitionServerRedirect != 0 {
		quals = append(quals, "SERVER_REDIRECT")
	}

	// Combine core type with qualifiers
	if len(quals) > 0 {
		return typeStr + "|" + strings.Join(quals, "|")
	}
	return typeStr
}
