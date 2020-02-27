// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package collector

import (
	"testing"

	"github.com/elastic/beats/libbeat/logp"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestEventMapping(t *testing.T) {
	logp.TestingSetup()

	mbtest.TestDataFiles(t, "openmetrics", "collector")
}
