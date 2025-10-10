// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"context"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
)

type parserFunc func(ctx context.Context, queryContext table.QueryContext, browserName, profilePath string, log func(m string, kvs ...any)) ([]*row, error)

type row struct {
	// Universal fields (available across all browsers)
	Timestamp      string `osquery:"timestamp"`
	Datetime       string `osquery:"datetime"`
	URL            string `osquery:"url"`
	Title          string `osquery:"title"`
	Browser        string `osquery:"browser"`
	Parser         string `osquery:"parser"`
	User           string `osquery:"user"`
	ProfileName    string `osquery:"profile_name"`
	ProfileFolder  string `osquery:"profile_folder"`
	TransitionType string `osquery:"transition_type"`
	ReferringURL   string `osquery:"referring_url"`
	VisitID        string `osquery:"visit_id"`
	FromVisitID    string `osquery:"from_visit_id"`
	UrlID          string `osquery:"url_id"`
	VisitCount     string `osquery:"visit_count"`
	TypedCount     string `osquery:"typed_count"`
	VisitSource    string `osquery:"visit_source"`
	IsHidden       string `osquery:"is_hidden"`
	SourcePath     string `osquery:"source_path"`

	// Chromium-specific fields (Chrome, Edge, Brave, etc.)
	ChVisitDurationMs string `osquery:"ch_visit_duration_ms"` // Only available in Chromium-based browsers

	// Firefox-specific fields
	FfSessionID string `osquery:"ff_session_id"` // Firefox session tracking
	FfFrecency  string `osquery:"ff_frecency"`   // Firefox user interest algorithm

	// Safari-specific fields
	SfDomainExpansion string `osquery:"sf_domain_expansion"` // Safari domain classification
	SfLoadSuccessful  string `osquery:"sf_load_successful"`  // Whether page loaded successfully
}

func newHistoryRow(parser, browserName, user, profileName, sourcePath string, timestamp int64) *row {
	return &row{
		Timestamp:     strconv.FormatInt(timestamp, 10),
		Datetime:      time.Unix(timestamp, 0).UTC().Format(time.RFC3339),
		Browser:       browserName,
		Parser:        parser,
		User:          user,
		ProfileName:   profileName,
		ProfileFolder: filepath.Base(sourcePath),
		SourcePath:    sourcePath,
	}
}

func (entry *row) toMap() map[string]string {
	result := make(map[string]string)

	v := reflect.ValueOf(entry).Elem()
	t := reflect.TypeOf(entry).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Get the osquery tag name
		tag := fieldType.Tag.Get("osquery")
		if tag == "" {
			continue // Skip fields without osquery tag
		}

		// Convert field value to string
		var value string
		switch field.Kind() {
		case reflect.String:
			value = field.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() == 0 {
				value = ""
			} else {
				value = strconv.FormatInt(field.Int(), 10)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if field.Uint() == 0 {
				value = ""
			} else {
				value = strconv.FormatUint(field.Uint(), 10)
			}
		default:
			value = ""
		}

		result[tag] = value
	}

	return result
}
