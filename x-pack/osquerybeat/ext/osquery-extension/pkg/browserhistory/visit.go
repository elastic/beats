// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"time"
)

type visit struct {
	// Universal fields (available across all browsers)
	Timestamp      int64  `osquery:"timestamp"`
	Datetime       string `osquery:"datetime"`
	URL            string `osquery:"url"`
	Title          string `osquery:"title"`
	Browser        string `osquery:"browser"`
	Parser         string `osquery:"parser"`
	User           string `osquery:"user"`
	ProfileName    string `osquery:"profile_name"`
	TransitionType string `osquery:"transition_type"`
	ReferringURL   string `osquery:"referring_url"`
	VisitID        int64  `osquery:"visit_id"`
	FromVisitID    int64  `osquery:"from_visit_id"`
	UrlID          int64  `osquery:"url_id"`
	VisitCount     int    `osquery:"visit_count"`
	TypedCount     int    `osquery:"typed_count"`
	VisitSource    string `osquery:"visit_source"`
	IsHidden       bool   `osquery:"is_hidden"`
	HistoryPath    string `osquery:"history_path"`

	// Chromium-specific fields (Chrome, Edge, Brave, etc.)
	ChVisitDurationMs int64 `osquery:"ch_visit_duration_ms"` // Only available in Chromium-based browsers

	// Firefox-specific fields
	FfSessionID int `osquery:"ff_session_id"` // Firefox session tracking
	FfFrecency  int `osquery:"ff_frecency"`   // Firefox user interest algorithm

	// Safari-specific fields
	SfDomainExpansion string `osquery:"sf_domain_expansion"` // Safari domain classification
	SfLoadSuccessful  bool   `osquery:"sf_load_successful"`  // Whether page loaded successfully

	CustomDataDir string `osquery:"custom_data_dir"`
}

func newVisit(parser string, profile *profile, timestamp int64) *visit {
	v := &visit{
		Timestamp:     timestamp,
		Datetime:      time.Unix(timestamp, 0).UTC().Format(time.RFC3339),
		Browser:       profile.browser,
		Parser:        parser,
		User:          profile.user,
		ProfileName:   profile.name,
		HistoryPath:   profile.historyPath,
		CustomDataDir: profile.customDataDir,
	}
	return v
}
