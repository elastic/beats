// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package query

import (
	"fmt"

	"github.com/elastic/beats/v7/metricbeat/helper/sql"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// ParseDSN parses the host for the oracle driver, the only driver allowed in
// FIPS builds. Oracle Net cryptography runs in the customer-installed Oracle
// client library, not in Go, so FIPS compliance depends on that client being
// configured with a FIPS 140-3 validated provider. The other drivers rely on
// cryptography outside the FIPS boundary.
func ParseDSN(mod mb.Module, host string) (_ mb.HostData, fetchErr error) {
	defer func() {
		fetchErr = sql.SanitizeError(fetchErr, host)
	}()

	config := ConnectionDetails{}
	if err := mod.UnpackConfig(&config); err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing config file: %w", err)
	}

	if config.Driver != "oracle" {
		return mb.HostData{}, fmt.Errorf("driver %q is not supported: only \"oracle\" is supported in FIPS mode", config.Driver)
	}

	return oracleParseDSN(config, host)
}
