// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package query

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/godror/godror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/mysql"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
)

type testFetchConfig struct {
	config    config
	Host      string
	Assertion func(t *testing.T, event beat.Event)
}

func TestMySQL(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")
	cfg := testFetchConfig{
		config: config{
			Driver:         "mysql",
			Query:          "select table_schema, table_name, engine, table_rows from information_schema.tables where table_rows > 0;",
			ResponseFormat: tableResponseFormat,
			RawData: rawData{
				Enabled: true,
			},
		},
		Host:      mysql.GetMySQLEnvDSN(service.Host()),
		Assertion: assertFieldNotContains("service.address", ":test@"),
	}
	t.Run("fetch", func(t *testing.T) {
		testFetch(t, cfg)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, cfg, "./_meta/data_mysql_tables.json")
	})

	cfg = testFetchConfig{
		config: config{
			Driver:         "mysql",
			Query:          "show status;",
			ResponseFormat: variableResponseFormat,
			RawData: rawData{
				Enabled: true,
			},
		},
		Host:      mysql.GetMySQLEnvDSN(service.Host()),
		Assertion: assertFieldNotContains("service.address", ":test@"),
	}

	t.Run("fetch", func(t *testing.T) {
		testFetch(t, cfg)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, cfg, "./_meta/data_mysql_variables.json")
	})
}

func TestPostgreSQL(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()

	cfg := testFetchConfig{
		config: config{
			Driver:         "postgres",
			Query:          "select * from pg_stat_database",
			ResponseFormat: tableResponseFormat,
		},
		Host:      fmt.Sprintf("user=%s password=%s sslmode=disable host=%s port=%s", user, password, host, port),
		Assertion: assertFieldNotContains("service.address", "password="+password),
	}

	t.Run("fetch", func(t *testing.T) {
		testFetch(t, cfg)
	})

	cfg = testFetchConfig{
		config: config{
			Driver:         "postgres",
			Query:          "select * from pg_stat_database where datname='postgres'",
			ResponseFormat: tableResponseFormat,
			RawData: rawData{
				Enabled: true,
			},
		},
		Host:      fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
		Assertion: assertFieldNotContains("service.address", ":"+password+"@"),
	}

	t.Run("fetch with URL", func(t *testing.T) {
		testFetch(t, cfg)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, cfg, "./_meta/data_postgres_tables.json")
	})

	cfg = testFetchConfig{
		config: config{
			Driver:         "postgres",
			Query:          "select name, setting from pg_settings",
			ResponseFormat: variableResponseFormat,
			RawData: rawData{
				Enabled: true,
			},
		},
		Host:      fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
		Assertion: assertFieldNotContains("service.address", ":"+password+"@"),
	}

	t.Run("fetch with URL", func(t *testing.T) {
		testFetch(t, cfg)
	})

	t.Run("data", func(t *testing.T) {
		testData(t, cfg, "./_meta/data_postgres_variables.json")
	})

	t.Run("raw_data", func(t *testing.T) {
		t.Run("variable mode", func(t *testing.T) {
			cfg = testFetchConfig{
				config: config{
					Driver:         "postgres",
					Query:          "select name, setting from pg_settings",
					ResponseFormat: variableResponseFormat,
					RawData: rawData{
						Enabled: true,
					},
				},
				Host: fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
				Assertion: func(t *testing.T, event beat.Event) {
					value, err := event.GetValue("sql.query")
					assert.NoError(t, err)
					require.NotEmpty(t, value)
				},
			}

			t.Run("fetch with URL", func(t *testing.T) {
				testFetch(t, cfg)
			})

		})

		t.Run("table mode", func(t *testing.T) {
			cfg = testFetchConfig{
				config: config{
					Driver:         "postgres",
					Query:          "select * from pg_settings",
					ResponseFormat: tableResponseFormat,
					RawData: rawData{
						Enabled: true,
					},
				},
				Host: fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
				Assertion: func(t *testing.T, event beat.Event) {
					value, err := event.GetValue("sql.query")
					assert.NoError(t, err)
					require.NotEmpty(t, value)
				},
			}

			t.Run("fetch with URL", func(t *testing.T) {
				testFetch(t, cfg)
			})

		})

		t.Run("merged mode", func(t *testing.T) {
			cfg = testFetchConfig{
				config: config{
					Driver: "postgres",
					Queries: []query{
						query{Query: "SELECT blks_hit FROM pg_stat_database limit 1;", ResponseFormat: "table"},
						query{Query: "SELECT blks_read FROM pg_stat_database limit 1;", ResponseFormat: "table"},
					},
					ResponseFormat: tableResponseFormat,
					RawData: rawData{
						Enabled: true,
					},
					MergeResults: true,
				},
				Host: fmt.Sprintf("postgres://%s:%s@%s:%s/?sslmode=disable", user, password, host, port),
				Assertion: func(t *testing.T, event beat.Event) {
					// Ensure both merged fields are there in a single event.
					value1, err1 := event.GetValue("sql.metrics.blks_hit")
					assert.NoError(t, err1)
					require.NotEmpty(t, value1)
					value2, err2 := event.GetValue("sql.metrics.blks_read")
					assert.NoError(t, err2)
					require.NotEmpty(t, value2)
				},
			}

			t.Run("fetch with URL", func(t *testing.T) {
				testFetch(t, cfg)
			})

		})
	})
}

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
	values := map[string]interface{}{
		"module":           "sql",
		"metricsets":       []string{"query"},
		"hosts":            []string{cfg.Host},
		"driver":           cfg.config.Driver,
		"sql_query":        cfg.config.Query,
		"sql_queries":      cfg.config.Queries,
		"raw_data.enabled": cfg.config.RawData.Enabled,
		"merge_results":    cfg.config.MergeResults,
	}
	if cfg.config.ResponseFormat != "" {
		values["sql_response_format"] = cfg.config.ResponseFormat
	}
	return values
}

func assertFieldNotContains(field, s string) func(t *testing.T, event beat.Event) {
	return func(t *testing.T, event beat.Event) {
		value, err := event.GetValue(field)
		assert.NoError(t, err)
		require.NotEmpty(t, value.(string))
		require.NotContains(t, value.(string), s)
	}
}

func assertFieldContainsFloat64(field string, limit float64) func(t *testing.T, event beat.Event) {
	return func(t *testing.T, event beat.Event) {
		value, err := event.GetValue("sql.metrics.hit_ratio")
		assert.NoError(t, err)
		require.GreaterOrEqual(t, value.(float64), limit)
	}
}

func GetOracleConnectionDetails(t *testing.T, host string, port string) string {
	params, err := godror.ParseDSN(GetOracleConnectString(host, port))
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
