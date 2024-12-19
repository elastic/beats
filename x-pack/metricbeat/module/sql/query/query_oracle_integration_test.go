// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && oracle

package query

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	// Drivers
	_ "github.com/godror/godror"

	"github.com/godror/godror/dsn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
)

func TestOracle(t *testing.T) {
	t.Skip("Flaky test: test containers fail over attempt to bind port 5500 https://github.com/elastic/beats/issues/35105")
	service := compose.EnsureUp(t, "oracle")
	host, port, _ := net.SplitHostPort(service.Host())
	cfg := testFetchConfig{
		config: config{
			Driver:         "oracle",
			Query:          `SELECT name, physical_reads, db_block_gets, consistent_gets, 1 - (physical_reads / (db_block_gets + consistent_gets)) "Hit_Ratio" FROM V$BUFFER_POOL_STATISTICS`,
			ResponseFormat: tableResponseFormat,
			RawData: rawData{
				Enabled: true,
			},
		},
		Host:      GetOracleConnectionDetails(t, host, port),
		Assertion: assertFieldContainsFloat64("Hit_Ratio", 0.0),
	}
	t.Run("fetch", func(t *testing.T) {
		testFetch(t, cfg)
	})
}

func assertFieldContainsFloat64(field string, limit float64) func(t *testing.T, event beat.Event) {
	return func(t *testing.T, event beat.Event) {
		value, err := event.GetValue("sql.metrics.hit_ratio")
		assert.NoError(t, err)
		require.GreaterOrEqual(t, value.(float64), limit)
	}
}

func GetOracleConnectionDetails(t *testing.T, host string, port string) string {
	params, err := dsn.Parse(GetOracleConnectString(host, port))
	require.Empty(t, err)
	return params.StringWithPassword()
}

// GetOracleEnvServiceName returns the service name to use with Oracle testing server or the value of the environment variable ORACLE_SERVICE_NAME if not empty
func GetOracleEnvServiceName() string {
	serviceName := os.Getenv("ORACLE_SERVICE_NAME")
	if len(serviceName) == 0 {
		serviceName = "ORCLCDB.localdomain"
	}
	return serviceName
}

// GetOracleEnvUsername returns the username to use with Oracle testing server or the value of the environment variable ORACLE_USERNAME if not empty
func GetOracleEnvUsername() string {
	username := os.Getenv("ORACLE_USERNAME")
	if len(username) == 0 {
		username = "sys"
	}
	return username
}

// GetOracleEnvUsername returns the port of the Oracle server or the value of the environment variable ORACLE_PASSWORD if not empty
func GetOracleEnvPassword() string {
	password := os.Getenv("ORACLE_PASSWORD")
	if len(password) == 0 {
		password = "Oradoc_db1" // #nosec
	}
	return password
}

func GetOracleConnectString(host string, port string) string {
	time.Sleep(300 * time.Second)
	connectString := os.Getenv("ORACLE_CONNECT_STRING")
	if len(connectString) == 0 {
		connectString = fmt.Sprintf("%s/%s@%s:%s/%s as sysdba", GetOracleEnvUsername(), GetOracleEnvPassword(), host, port, GetOracleEnvServiceName())
	}
	return connectString
}
