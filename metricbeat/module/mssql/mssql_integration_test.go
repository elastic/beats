// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build integration

package mssql

import (
	"testing"

	_ "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestNewDB(t *testing.T) {
	//TODO
	// compose.EnsureUp(t, "mssql")

	// db, err := NewDB(GetMySQLEnvDSN())
	// assert.NoError(t, err)

	// err = db.Ping()
	// assert.NoError(t, err)
}
