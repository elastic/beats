// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"database/sql"
	"fmt"

	"github.com/godror/godror/dsn"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// ConnectionDetails contains all possible data that can be used to create a connection with
// an Oracle db
type ConnectionDetails struct {
	Username string        `config:"username"`
	Password string        `config:"password"`
	Patterns []interface{} `config:"patterns"`
}

// HostParser parses host and extracts connection information and returns it to HostData
// HostData can then be used to make connection to SQL
func HostParser(mod mb.Module, rawURL string) (mb.HostData, error) {
	params, err := dsn.Parse(rawURL)
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error trying to parse connection string in field 'hosts': %w", err)
	}

	config := ConnectionDetails{}
	if err := mod.UnpackConfig(&config); err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing config file: %w", err)
	}

	if params.Username == "" {
		params.Username = config.Username
	}

	if params.Password.Secret() == "" {
		params.StandaloneConnection = true
		params.Password = dsn.NewPassword(config.Password)
	}

	return mb.HostData{
		URI:          params.StringWithPassword(),
		SanitizedURI: params.ConnectString,
		Host:         params.String(),
		User:         params.Username,
		Password:     params.Password.Secret(),
	}, nil
}

func NewConnection(connString string) (*sql.DB, error) {
	db, err := sql.Open("godror", connString)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	// Check the connection before executing all queries to reduce the number
	// of connection errors that we might encounter.
	if err = db.Ping(); err != nil {
		err = fmt.Errorf("error doing ping to database: %w", err)
	}

	return db, err
}
