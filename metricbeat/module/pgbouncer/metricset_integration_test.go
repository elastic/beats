// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build integration

package pgbouncer

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/postgresql"
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
		"hosts":      []string{"localhost:6432/pgbouncer?sslmode=disable"},
		"username":   "test",
		"password":   postgresql.GetEnvPassword(),
	}
}
