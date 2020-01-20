// +build integration

package query

import (
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mysql"
	"github.com/elastic/beats/metricbeat/module/postgresql"

	// Drivers
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type testFetchConfig struct {
	Driver     string
	Query      string
	Host       string
	Datasource string
}

func TestFetchMySQL(t *testing.T) {
	service := compose.EnsureUp(t, "mysql")
	testFetch(t, testFetchConfig{
		Driver:     "mysql",
		Query:      "select now()",
		Host:       service.Host(),
		Datasource: mysql.GetMySQLEnvDSN(service.Host()),
	})
}

func TestFetchPostgreSQL(t *testing.T) {
	service := compose.EnsureUp(t, "postgresql")
	host, port, err := net.SplitHostPort(service.Host())
	require.NoError(t, err)

	user := postgresql.GetEnvUsername()
	password := postgresql.GetEnvPassword()

	testFetch(t, testFetchConfig{
		Driver:     "postgres",
		Query:      "select now()",
		Host:       service.Host(),
		Datasource: fmt.Sprintf("user=%s password=%s sslmode=disable host=%s port=%s", user, password, host, port),
	})
}

func TestData(t *testing.T) {
}

func testFetch(t *testing.T, cfg testFetchConfig) {
	m := mbtest.NewFetcher(t, getConfig(cfg))
	events, errs := m.FetchEvents()
	require.Empty(t, errs)
	require.NotEmpty(t, events)
	t.Logf("%s/%s event: %+v", m.Module().Name(), m.Name(), events[0])
}

func getConfig(cfg testFetchConfig) map[string]interface{} {
	return map[string]interface{}{
		"module":     "sql",
		"metricsets": []string{"query"},
		"hosts":      []string{cfg.Host},
		"driver":     cfg.Driver,
		"sql_query":  cfg.Query,
		"datasource": cfg.Datasource,
	}
}
