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

/*
Package mysql is Metricbeat module for MySQL server.
*/
package mysql

import (
	"database/sql"

	"github.com/elastic/beats/v8/metricbeat/mb"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

func init() {
	// Register the ModuleFactory function for the "mysql" module.
	if err := mb.Registry.AddModule("mysql", NewModule); err != nil {
		panic(err)
	}
}

func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

// ParseDSN creates a DSN (data source name) string by parsing the host.
// It validates the resulting DSN and returns an error if the DSN is invalid.
//
//   Format:  [username[:password]@][protocol[(address)]]/
//   Example: root:test@tcp(127.0.0.1:3306)/
func ParseDSN(mod mb.Module, host string) (mb.HostData, error) {
	c := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{}
	if err := mod.UnpackConfig(&c); err != nil {
		return mb.HostData{}, err
	}

	config, err := mysql.ParseDSN(host)
	if err != nil {
		return mb.HostData{}, errors.Wrapf(err, "error parsing mysql host")
	}

	if config.User == "" {
		config.User = c.Username
	}

	if config.Passwd == "" {
		config.Passwd = c.Password
	}

	// Add connection timeouts to the DSN.
	if timeout := mod.Config().Timeout; timeout > 0 {
		config.Timeout = timeout
		config.ReadTimeout = timeout
		config.WriteTimeout = timeout
	}

	noCredentialsConfig := *config
	noCredentialsConfig.User = ""
	noCredentialsConfig.Passwd = ""

	return mb.HostData{
		URI:          config.FormatDSN(),
		SanitizedURI: noCredentialsConfig.FormatDSN(),
		Host:         config.Addr,
		User:         config.User,
		Password:     config.Passwd,
	}, nil
}

// NewDB returns a new mysql database handle. The dsn value (data source name)
// must be valid, otherwise an error will be returned.
//
//   DSN Format: [username[:password]@][protocol[(address)]]/
func NewDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "sql open failed")
	}
	return db, nil
}
