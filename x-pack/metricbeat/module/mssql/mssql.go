// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
)

const (
	urlScheme = "sqlserver"
)

// HostParser parsers the host value as a URL. The mssql driver expects either
// odbc:// or sqlserver:// schemes. It will automatically add the default port
// of 1433 if a port is not included in the host.
var HostParser = parse.URLHostParserBuilder{
	DefaultScheme: urlScheme,
}.Build()

func init() {
	// Register the ModuleFactory function for the "mssql" module.
	if err := mb.Registry.AddModule("mssql", newModule); err != nil {
		panic(err)
	}
}

// newModule adds validation that hosts is non-empty, a requirement to use the
// mssql module.
func newModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}
