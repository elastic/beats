// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package query

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/metricbeat/mb"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/paths"
)

func newFIPSModule(t *testing.T, driver, host string) ([]mb.MetricSet, error) {
	t.Helper()

	c, err := conf.NewConfigFrom(map[string]any{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{host},
		"driver":              driver,
		"username":            "scott",
		"password":            "tiger",
		"sql_query":           "SELECT 1 FROM DUAL",
		"sql_response_format": "table",
	})
	require.NoError(t, err, "building the module config should not fail")

	_, metricsets, err := mb.NewModule(c, mb.Registry, beat.Info{
		Paths:  &paths.Path{Data: t.TempDir()},
		Logger: logptest.NewTestingLogger(t, ""),
	})
	return metricsets, err
}

func TestFIPSParseDSN_AcceptsOracle(t *testing.T) {
	metricsets, err := newFIPSModule(t, "oracle", "oracle://localhost:1521/ORCLCDB.localdomain?sysdba=1")
	require.NoError(t, err, "the oracle driver must be usable in FIPS builds")
	require.Len(t, metricsets, 1, "expected one sql/query metricset")

	host := metricsets[0].HostData()
	assert.Equal(t, "scott", host.User, "username from the module config should be applied")
	assert.Equal(t, "tiger", host.Password, "password from the module config should be applied")
	assert.Equal(t, "localhost:1521/ORCLCDB.localdomain", host.SanitizedURI, "sanitized URI should be the connect string only")
	assert.NotContains(t, host.SanitizedURI, "tiger", "sanitized URI must not leak the password")
}

func TestFIPSParseDSN_RejectsOtherDrivers(t *testing.T) {
	tests := []struct {
		driver string
		host   string
	}{
		{driver: "mysql", host: "root:test@tcp(localhost:3306)/"},
		{driver: "postgres", host: "postgres://user:secret@localhost:5432/db"}, //nolint:gosec // fake credentials, checked for redaction below
		{driver: "mssql", host: "sqlserver://user:secret@localhost:1433?database=db"},
		{driver: "godror", host: "oracle://localhost:1521/ORCLCDB.localdomain"},
	}

	for _, tc := range tests {
		t.Run(tc.driver, func(t *testing.T) {
			_, err := newFIPSModule(t, tc.driver, tc.host)
			require.Error(t, err, "driver %q must be rejected in FIPS builds", tc.driver)
			assert.Contains(t, err.Error(), `only "oracle" is supported in FIPS mode`, "error should name the supported driver")
			assert.NotContains(t, err.Error(), "secret", "error must not leak credentials from the host")
		})
	}
}
