// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"database/sql"
	"fmt"

	"github.com/godror/godror"
	"github.com/godror/godror/dsn"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

// ConnectionDetails contains all possible data that can be used to create a connection with
// an Oracle db
type ConnectionDetails struct {
	Username string   `config:"username"`
	Password string   `config:"password"`
	Hosts    []string `config:"hosts"    validate:"required"`
}

// HostParser parsers the host value as a URL
var HostParser = parse.URLHostParserBuilder{
	DefaultScheme: "oracle",
}.Build()

func init() {
	// Register the ModuleFactory function for the "oracle" module.
	if err := mb.Registry.AddModule("oracle", newModule); err != nil {
		panic(err)
	}
}

// NewConnection returns a connection already established with Oracle
func NewConnection(c *ConnectionDetails) (*sql.DB, error) {
	params, err := godror.ParseDSN(c.Hosts[0])
	if err != nil {
		return nil, fmt.Errorf("error trying to parse URL in field 'hosts': %w", err)
	}

	// If username and password are given in separate fields in the configuration then use them to authenticate
	if params.Username == "" {
		params.Username = c.Username
	}

	if params.Password.Secret() == "" {
		params.StandaloneConnection = true
		params.Password = dsn.NewPassword(c.Password)
	}

	db, err := sql.Open("godror", params.StringWithPassword())
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

// newModule adds validation that hosts is non-empty, a requirement to use the
// Oracle module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := ConnectionDetails{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("error parsing config module: %w", err)
	}

	return &base, nil
}
