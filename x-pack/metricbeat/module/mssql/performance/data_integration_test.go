// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"net/url"
	"testing"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	mtest "github.com/elastic/beats/x-pack/metricbeat/module/mssql/testing"
)

func TestData(t *testing.T) {
	t.Skip("Skipping `data.json` generation test")

	_, config, err := getHostURI()
	if err != nil {
		t.Fatal("error getting config information", err.Error())
	}

	f := mbtest.NewReportingMetricSetV2(t, config)
	events, errs := mbtest.ReportingFetchV2(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)

	if err = mbtest.WriteEventsReporterV2(f, t, ""); err != nil {
		t.Fatal("write", err)
	}
}

func getHostURI() (string, map[string]interface{}, error) {
	config := mtest.GetConfig("performance")

	host, ok := config["hosts"].([]string)
	if !ok {
		return "", nil, errors.New("error getting host name information")
	}

	username, ok := config["username"].(string)
	if !ok {
		return "", nil, errors.New("error getting username information")
	}

	password, ok := config["password"].(string)
	if !ok {
		return "", nil, errors.New("error getting password information")
	}

	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(username, password),
		Host:   host[0],
	}

	return u.String(), config, nil
}
