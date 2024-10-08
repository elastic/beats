package pgbouncer

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
	"github.com/stretchr/testify/assert"
)

func TestNewMetricSet(t *testing.T) {
	base := mb.BaseMetricSet{}
	metricSet, err := NewMetricSet(base)
	assert.NoError(t, err)
	assert.NotNil(t, metricSet)
}

func TestDBConnection(t *testing.T) {
	db, err := connectDatabase(t)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}
	defer db.Close()
	ctx := context.Background()
	metricSet := MetricSet{
		db: db,
	}
	conn, err := metricSet.DB(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, conn, "The database connection should not be nil")
	if conn != nil {
		defer conn.Close()
	}
}

func TestQueryStats(t *testing.T) {
	db, err := connectDatabase(t)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}

	defer db.Close()
	metricSet := MetricSet{
		db: db,
	}
	ctx := context.Background()
	query := "SHOW STATS;"
	results, err := metricSet.QueryStats(ctx, query)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.NotEmpty(t, results)
}

func TestClose(t *testing.T) {
	db, err := connectDatabase(t)
	if err != nil {
		t.Fatalf("Error opening database: %s", err)
	}

	metricSet := MetricSet{
		db: db,
	}

	err = metricSet.Close()
	assert.NoError(t, err)

	err = db.Ping()
	assert.Error(t, err)
}

func connectDatabase(t *testing.T) (*sql.DB, error) {
	service := compose.EnsureUp(t, "pgbouncer")
	config := getConfig(service.Host())

	dsn := fmt.Sprintf("postgres://%s:%s@%s",
		config["username"].(string),
		config["password"].(string),
		config["hosts"].([]string)[0],
	)

	db, err := sql.Open("postgres", dsn)
	return db, err
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "pgbouncer",
		"metricsets": []string{"stats"},
		"hosts":      []string{fmt.Sprintf("localhost:6432/pgbouncer?sslmode=disable")},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
