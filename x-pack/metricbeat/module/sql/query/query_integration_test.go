// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package query

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/tests/compose"
	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mysql"
	"github.com/elastic/beats/metricbeat/module/postgresql"
)

type testFetchConfig struct {
	Driver            string
	Query             string
	Host              string
	SQLResponseFormat string

	Assertion func(t *testing.T, event beat.Event)
}

func TestMySQL(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")
	config := testFetchConfig{
		Driver:            "mysql",
		Query:             "select table_schema, table_name, engine, table_rows from information_schema.tables where table_rows > 0;",
		Host:              mysql.GetMySQLEnvDSN(service.Host()),
		SQLResponseFormat: table,
		Assertion:         assertFieldNotContains("service.address", ":test@"),
	}

	t.Run("fetch", func(t *testing.T) {
		testFetch(t, config)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, config, "")
	})

	config = testFetchConfig{
		Driver:            "mysql",
		Query:             "show status;",
		Host:              mysql.GetMySQLEnvDSN(service.Host()),
		SQLResponseFormat: variable,
		Assertion:         assertFieldNotContains("service.address", ":test@"),
	}

	t.Run("fetch", func(t *testing.T) {
		testFetch(t, config)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, config, "")
	})
}

func TestPostgreSQL(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()

	config := testFetchConfig{
		Driver:            "postgres",
		Query:             "select * from pg_stat_database",
		Host:              fmt.Sprintf("user=%s password=%s sslmode=disable host=%s port=%s", user, password, host, port),
		SQLResponseFormat: table,
		Assertion:         assertFieldNotContains("service.address", "password="+password),
	}

	t.Run("fetch", func(t *testing.T) {
		testFetch(t, config)
	})

	config = testFetchConfig{
		Driver:            "postgres",
		Query:             "select * from pg_stat_database",
		Host:              fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
		SQLResponseFormat: table,
		Assertion:         assertFieldNotContains("service.address", ":"+password+"@"),
	}

	t.Run("fetch with URL", func(t *testing.T) {
		testFetch(t, config)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, config, "./_meta/data_postgres.json")
	})

	config = testFetchConfig{
		Driver:            "postgres",
		Query:             "select name, setting from pg_settings",
		Host:              fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
		SQLResponseFormat: variable,
		Assertion:         assertFieldNotContains("service.address", ":"+password+"@"),
	}

	t.Run("fetch with URL", func(t *testing.T) {
		testFetch(t, config)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, config, "./_meta/data_postgres.json")
	})
}

func testFetch(t *testing.T, cfg testFetchConfig) {
	m := mbtest.NewFetcher(t, getConfig(cfg))
	events, errs := m.FetchEvents()
	require.Empty(t, errs)
	require.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", m.Module().Name(), m.Name(), events[0])

	if cfg.Assertion != nil {
		for _, event := range events {
			cfg.Assertion(t, m.StandardizeEvent(event, mb.AddMetricSetInfo))
		}
	}
}

func testData(t *testing.T, cfg testFetchConfig, postfix string) {
	m := mbtest.NewFetcher(t, getConfig(cfg))
	m.WriteEvents(t, postfix)
}

func getConfig(cfg testFetchConfig) map[string]interface{} {
	return map[string]interface{}{
		"module":              "sql",
		"metricsets":          []string{"query"},
		"hosts":               []string{cfg.Host},
		"driver":              cfg.Driver,
		"sql_query":           cfg.Query,
		"sql_response_format": cfg.SQLResponseFormat,
	}
}

func assertFieldNotContains(field, s string) func(t *testing.T, event beat.Event) {
	return func(t *testing.T, event beat.Event) {
		value, err := event.GetValue(field)
		assert.NoError(t, err)
		require.NotEmpty(t, value.(string))
		require.NotContains(t, value.(string), s)
	}
}
