// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"
)

func firefoxParser(ctx context.Context, queryContext table.QueryContext, browserName, profilePath string, log func(m string, kvs ...any)) ([]*row, error) {
	connectionString := fmt.Sprintf("file:%s?mode=ro&cache=shared&immutable=1", filepath.Join(profilePath, "places.sqlite"))
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		log("failed to open database", "error", err)
		return nil, fmt.Errorf("failed to open Firefox history database: %w", err)
	}
	defer db.Close()

	query := `
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
		WHERE p.visit_count > 0
		ORDER BY hv.visit_date DESC
	`

	log("executing SQL query", "query", query)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log("failed to execute query", "error", err)
		return nil, fmt.Errorf("failed to query Firefox history: %w", err)
	}
	defer rows.Close()

	user := extractUserFromPath(profilePath, log)
	profileName := extractFirefoxProfileName(profilePath, log)

	var entries []*row
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
			log("failed to scan row", "rowNumber", rowCount, "error", err)
			continue
		}

		entry := newHistoryRow("firefox", browserName, user, profileName, profilePath)
		entry.Timestamp = formatNullInt64(visitTime, func(value int64) string {
			return strconv.FormatInt(value/1000000, 10)
		})
		entry.URL = stringFromNullString(url)
		entry.Title = stringFromNullString(title)
		entry.TransitionType = mapFirefoxTransitionType(transition)
		entry.ReferringURL = stringFromNullString(referringURL)
		entry.VisitID = decimalStringFromNullInt(visitID)
		entry.FromVisitID = decimalStringFromNullInt(fromVisitID)
		entry.VisitCount = decimalStringFromNullInt(visitCount)
		entry.TypedCount = decimalStringFromNullInt(typedCount)
		entry.VisitSource = mapFirefoxVisitSource(visitSource)
		entry.IsHidden = boolStringFromNullInt(isHidden)
		entry.UrlID = decimalStringFromNullInt(urlID)
		entry.FfSessionID = decimalStringFromNullInt(ffSessionID)
		entry.FfFrecency = decimalStringFromNullInt(ffFrecency)

		entries = append(entries, entry)
	}

	log("completed reading history", "totalRows", rowCount, "validEntries", len(entries), "historyPath", profilePath)
	return entries, rows.Err()
}

func extractFirefoxProfileName(profilePath string, log func(m string, kvs ...any)) string {
	profileFolderName := filepath.Base(profilePath)
	profilesDir := filepath.Dir(profilePath)
	firefoxDir := filepath.Dir(profilesDir)
	profilesIniPath := filepath.Join(firefoxDir, "profiles.ini")

	if file, err := os.Open(profilesIniPath); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)

		var currentProfile string
		var profileName string
		var profilePath string

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "[Profile") && strings.HasSuffix(line, "]") {
				currentProfile = line
				profileName = ""
				profilePath = ""
			} else if currentProfile != "" {
				if strings.HasPrefix(line, "Name=") {
					profileName = strings.TrimPrefix(line, "Name=")
				} else if strings.HasPrefix(line, "Path=") {
					profilePath = strings.TrimPrefix(line, "Path=")
					profilePath = strings.TrimPrefix(profilePath, "Profiles/")
				}
				if profileName != "" && profilePath != "" && profilePath == profileFolderName {
					log("extracted profile name from profiles.ini", "name", profileName, "folder", profileFolderName)
					return profileName
				}
			}
		}
	}
	log("using folder name as profile name", "name", profileFolderName)
	return profileFolderName
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
