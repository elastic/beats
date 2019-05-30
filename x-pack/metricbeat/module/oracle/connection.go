// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"database/sql"
	"fmt"

	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	// Register driver.
	_ "gopkg.in/goracle.v2"

	"github.com/pkg/errors"
)

// ConnectionDetails contains all possible data that can be used to create a connection with
// an Oracle db
type ConnectionDetails struct {
	Username    string   `config:"username"    validate:"nonzero"`
	Password    string   `config:"password"    validate:"nonzero"`
	Hosts       []string `config:"hosts"    validate:"nonzero"`
	ServiceName string   `config:"service_name"    validate:"nonzero"`
	Prefix      string   `config:"sid_connection_prefix"`
	Suffix      string   `config:"sid_connection_suffix"`
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
	sid := fmt.Sprintf("%s%s/%s@%s/%s%s", c.Prefix, c.Username, c.Password, c.Hosts[0], c.ServiceName, c.Suffix)
	db, err := sql.Open("goracle", sid)
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
