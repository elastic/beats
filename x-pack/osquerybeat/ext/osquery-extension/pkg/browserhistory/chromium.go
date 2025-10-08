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
	"strconv"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

func chromiumParser(ctx context.Context, queryContext table.QueryContext, profilePath, browserName string, log func(m string, kvs ...any)) ([]*row, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", filepath.Join(profilePath, "History"))
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
			urls.hidden,
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
		ORDER BY visits.visit_time DESC
	`

	log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Chromium history: %w", err)
	}
	defer rows.Close()

	user := extractUserFromPath(profilePath, func(m string, kvs ...any) {})
	profileName := extractChromiumProfileName(profilePath, func(m string, kvs ...any) {})

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
			&entry.title,
			&entry.visitCount,
			&entry.typedCount,
			&entry.isHidden,
			&entry.lastVisitTime,
			&entry.transitionType,
			&entry.visitID,
			&entry.fromVisitID,
			&entry.chVisitDuration,
			&entry.visitSource,
			&entry.referringURL,
		)
		if err != nil {
			log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}
		entries = append(entries, chromiumEntryToRow(entry))
	}

	log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", profilePath)
	return entries, rows.Err()
}

func extractChromiumProfileName(profilePath string, log func(m string, kvs ...any)) string {
	profileFolderName := filepath.Base(profilePath)
	userDataDir := filepath.Dir(profilePath)
	localStatePath := filepath.Join(userDataDir, "Local State")
	if data, err := os.ReadFile(localStatePath); err == nil {
		var localState struct {
			Profile struct {
				InfoCache map[string]struct {
					Name string `json:"name"`
				} `json:"info_cache"`
			} `json:"profile"`
		}

		if err := json.Unmarshal(data, &localState); err == nil {
			if profileInfo, exists := localState.Profile.InfoCache[profileFolderName]; exists && profileInfo.Name != "" {
				log("extracted profile name from Local State", "name", profileInfo.Name, "folder", profileFolderName)
				return profileInfo.Name
			}
		}
	}
	log("using folder name as profile name", "name", profileFolderName)
	return profileFolderName
}

func chromiumEntryToRow(entry rawHistoryEntry) *row {
	return &row{
		Timestamp: formatNullInt64(entry.lastVisitTime, func(value int64) string {
			return strconv.FormatInt(chromiumTimeToUnix(value), 10)
		}),
		URL:            stringFromNullString(entry.url),
		Title:          stringFromNullString(entry.title),
		Browser:        entry.browserName,
		Parser:         "chromium",
		User:           entry.user,
		ProfileName:    entry.profile,
		TransitionType: mapChromiumTransitionType(entry.transitionType),
		ReferringURL:   stringFromNullString(entry.referringURL),
		VisitID:        decimalStringFromNullInt(entry.visitID),
		FromVisitID:    decimalStringFromNullInt(entry.fromVisitID),
		VisitCount:     decimalStringFromNullInt(entry.visitCount),
		TypedCount:     decimalStringFromNullInt(entry.typedCount),
		VisitSource:    mapChromiumVisitSource(entry.visitSource),
		IsHidden:       boolStringFromNullInt(entry.isHidden),
		SourcePath:     entry.path,
		ChVisitDurationMs: formatNullInt64(entry.chVisitDuration, func(value int64) string {
			return strconv.FormatInt(value/1000, 10)
		}),
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
