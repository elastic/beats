// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package query

import (
	"net/url"

	"github.com/go-sql-driver/mysql"

	"github.com/elastic/beats/v8/metricbeat/mb"
)

// ParseDSN tries to parse the host
func ParseDSN(mod mb.Module, host string) (mb.HostData, error) {
	// TODO: Add support for `username` and `password` as module options

	sanitized := sanitize(host)

	return mb.HostData{
		URI:          host,
		SanitizedURI: sanitized,
		Host:         sanitized,
	}, nil
}

func sanitize(host string) string {
	// Host is a standard URL
	if url, err := url.Parse(host); err == nil && len(url.Host) > 0 {
		return url.Host
	}

	// Host is a MySQL DSN
	if config, err := mysql.ParseDSN(host); err == nil {
		return config.Addr
	}

	// TODO: Add support for PostgreSQL connection strings and other formats

	return "(redacted)"
}
