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
	Timestamp      time.Time `osquery:"timestamp" format:"unix" desc:"Unix timestamp of the visit"`
	Datetime       time.Time `osquery:"datetime" format:"rfc3339" tz:"UTC" desc:"Datetime of the visit in RFC3339 format"`
	UrlID          int64     `osquery:"url_id" desc:"Unique identifier for the URL"`
	Scheme         string    `osquery:"scheme" desc:"URL scheme (e.g., http, https)"`
	Domain         string    `osquery:"domain" desc:"Domain of the visited URL"`
	Hostname       string    `osquery:"hostname" desc:"Hostname of the visited URL"`
	URL            string    `osquery:"url" desc:"Full URL of the visited page"`
	Title          string    `osquery:"title" desc:"Title of the visited page"`
	Browser        string    `osquery:"browser" desc:"Browser used for the visit"`
	Parser         string    `osquery:"parser" desc:"Parser used to extract the visit"`
	User           string    `osquery:"user" desc:"User associated with the visit"`
	ProfileName    string    `osquery:"profile_name" desc:"Profile name associated with the visit"`
	TransitionType string    `osquery:"transition_type" desc:"Type of transition for the visit"`
	ReferringURL   string    `osquery:"referring_url" desc:"Referring URL for the visit"`
	VisitID        int64     `osquery:"visit_id" desc:"Unique identifier for the visit"`
	FromVisitID    int64     `osquery:"from_visit_id" desc:"Identifier for the originating visit"`
	VisitSource    string    `osquery:"visit_source" desc:"Source of the visit"`
	IsHidden       bool      `osquery:"is_hidden" desc:"Whether the visit is hidden"`
	HistoryPath    string    `osquery:"history_path" desc:"Path to the history database"`

	// Chromium-specific fields (Chrome, Edge, Brave, etc.)
	ChVisitDurationMs int64 `osquery:"ch_visit_duration_ms" desc:"Duration of the visit in milliseconds"`

	// Firefox-specific fields
	FfSessionID int `osquery:"ff_session_id" desc:"Firefox session tracking"`
	FfFrecency  int `osquery:"ff_frecency" desc:"Firefox user interest algorithm"`

	// Safari-specific fields
	SfDomainExpansion string `osquery:"sf_domain_expansion" desc:"Safari domain classification"`
	SfLoadSuccessful  bool   `osquery:"sf_load_successful" desc:"Whether page loaded successfully"`

	CustomDataDir string `osquery:"custom_data_dir" desc:"Custom data directory if applicable"`
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
