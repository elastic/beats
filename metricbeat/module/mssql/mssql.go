// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/elastic/beats/metricbeat/mb"
	"net/url"
)

func init() {
	if err := mb.Registry.AddModule("mssql", newModule); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
}

type Config struct {
	Host     string `config:"host" validate:"nonzero,required"`
	User     string `config:"user" validate:"nonzero,required"`
	Password string `config:"password" validate:"nonzero,required"`
	Port     int    `config:"port" validate:"nonzero,required"`
}

func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config Config
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{BaseMetricSet: base}, nil
}

// NewDB returns a *sql.DB instance created with the provided configuration. Useful to be called on each Fetch method
// from each metricset
func NewDB(config *Config) (*sql.DB, error) {
	u := &url.URL{
		Scheme: "sqlserver",
		User:   url.UserPassword(config.User, config.Password),
		Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
	}
	return sql.Open("sqlserver", u.String())
}

func newModule(base mb.BaseModule) (mb.Module, error) {
	var config Config
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}
