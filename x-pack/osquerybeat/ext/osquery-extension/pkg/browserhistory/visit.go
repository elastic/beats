// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"net/url"
	"time"
)

type visit struct {
	// Universal fields (available across all browsers)
	Timestamp      time.Time `osquery:"timestamp" format:"unix"`
	Datetime       time.Time `osquery:"datetime" format:"rfc3339" tz:"UTC"`
	UrlID          int64     `osquery:"url_id"`
	Scheme         string    `osquery:"scheme"`
	Domain         string    `osquery:"domain"`
	URL            string    `osquery:"url"`
	Title          string    `osquery:"title"`
	Browser        string    `osquery:"browser"`
	Parser         string    `osquery:"parser"`
	User           string    `osquery:"user"`
	ProfileName    string    `osquery:"profile_name"`
	TransitionType string    `osquery:"transition_type"`
	ReferringURL   string    `osquery:"referring_url"`
	VisitID        int64     `osquery:"visit_id"`
	FromVisitID    int64     `osquery:"from_visit_id"`
	VisitSource    string    `osquery:"visit_source"`
	IsHidden       bool      `osquery:"is_hidden"`
	HistoryPath    string    `osquery:"history_path"`

	// Chromium-specific fields (Chrome, Edge, Brave, etc.)
	ChVisitDurationMs int64 `osquery:"ch_visit_duration_ms"` // Only available in Chromium-based browsers

	// Firefox-specific fields
	FfSessionID int `osquery:"ff_session_id"` // Firefox session tracking
	FfFrecency  int `osquery:"ff_frecency"`   // Firefox user interest algorithm

	// Safari-specific fields
	SfDomainExpansion string `osquery:"sf_domain_expansion"` // Safari domain classification
	SfLoadSuccessful  bool   `osquery:"sf_load_successful"`  // Whether page loaded successfully

	CustomDataDir string `osquery:"custom_data_dir"` // Custom data directory if applicable
}

func newVisit(parser string, profile *profile, timestamp int64) *visit {
	t := time.Unix(timestamp, 0)
	return &visit{
		Timestamp:     t,
		Datetime:      t,
		Browser:       profile.browser,
		Parser:        parser,
		User:          profile.user,
		ProfileName:   profile.name,
		HistoryPath:   profile.historyPath,
		CustomDataDir: profile.customDataDir,
	}
}

func extractSchemeAndDomain(rawURL string) (string, string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	return parsedURL.Scheme, parsedURL.Hostname()
}
