// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"net/url"
	"time"

	elasticbrowserhistory "github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/tables/generated/elastic_browser_history"
	"golang.org/x/net/publicsuffix"
)

func newResult(parser string, profile *profile, timestamp int64) elasticbrowserhistory.Result {
	t := time.Unix(timestamp, 0)
	return elasticbrowserhistory.Result{
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
