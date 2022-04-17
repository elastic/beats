// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"database/sql"

	"github.com/godror/godror"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"

	"github.com/pkg/errors"
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
	params, err := godror.ParseConnString(c.Hosts[0])
	if err != nil {
		return nil, errors.Wrap(err, "error trying to parse connection string in field 'hosts'")
	}

	if params.Username == "" {
		params.Username = c.Username
	}

	if params.Password == "" {
		params.Password = c.Password
	}

	db, err := sql.Open("godror", params.StringWithPassword())
	if err != nil {
		return nil, errors.Wrap(err, "could not open database")
	}

	// Check the connection before executing all queries to reduce the number
	// of connection errors that we might encounter.
	if err = db.Ping(); err != nil {
		err = errors.Wrap(err, "error doing ping to database")
	}

	return db, err
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// Oracle module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := ConnectionDetails{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, errors.Wrap(err, "error parsing config module")
	}

	return &base, nil
}
