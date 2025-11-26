// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"net/url"
	"time"

	"golang.org/x/net/publicsuffix"
)

type visit struct {
	// Universal fields (available across all browsers)
	Timestamp      time.Time `osquery:"timestamp" format:"unix"`
	Datetime       time.Time `osquery:"datetime" format:"rfc3339" tz:"UTC"`
	UrlID          int64     `osquery:"url_id"`
	Scheme         string    `osquery:"scheme"`
	Domain         string    `osquery:"domain"`
	Hostname       string    `osquery:"hostname"`
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
		Parser:        parser,
		Browser:       profile.Browser,
		User:          profile.User,
		ProfileName:   profile.Name,
		HistoryPath:   profile.HistoryPath,
		CustomDataDir: profile.CustomDataDir,
	}
}

func extractSchemeHostAndTLDPPlusOne(rawURL string) (string, string, string) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", "", ""
	}
	// EffectiveTLDPlusOne returns the "registrable domain".
	// e.g., "www.google.com" -> "google.com"
	// e.g., "blog.example.co.uk" -> "example.co.uk"
	// e.g., "my-project.github.io" -> "my-project.github.io"
	eTLDPlusOne, _ := publicsuffix.EffectiveTLDPlusOne(parsedURL.Hostname())
	return parsedURL.Scheme, parsedURL.Hostname(), eTLDPlusOne
}
