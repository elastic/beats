// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package performance

import (
	"testing"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v9/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/v9/x-pack/metricbeat/module/mssql/testing"
)

func TestData(t *testing.T) {
	t.Skip("Skipping `data.json` generation test")

	f := mbtest.NewReportingMetricSetV2(t, mtest.GetConfig("performance"))

	err := mbtest.WriteEventsReporterV2(f, t, "")
	assert.NoError(t, err)
}
