// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"database/sql"
	"fmt"

	// Register driver.
	_ "github.com/denisenkom/go-mssqldb"
)

// NewConnection returns a connection already established with MSSQL
func NewConnection(uri string) (*sql.DB, error) {
	db, err := sql.Open("sqlserver", uri)
	if err != nil {
		return nil, fmt.Errorf("could not create db instance: %w", err)
	}

	// Check the connection before executing all queries to reduce the number
	// of connection errors that we might encounter.
	if err = db.Ping(); err != nil {
		err = fmt.Errorf("error doing ping to db: %w", err)
	}

	return db, err
}
