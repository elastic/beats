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
Package postgresql is Metricbeat module for PostgreSQL server.
*/
package postgresql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

func init() {
	// Register the ModuleFactory function for the "postgresql" module.
	if err := mb.Registry.AddModule("postgresql", NewModule); err != nil {
		panic(err)
	}
}

//NewModule returns a new instance of the module
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

//ParseURL is the postgres host parser
func ParseURL(mod mb.Module, rawURL string) (mb.HostData, error) {
	c := struct {
		Username string `config:"username"`
		Password string `config:"password"`
	}{}
	if err := mod.UnpackConfig(&c); err != nil {
		return mb.HostData{}, err
	}

	if parts := strings.SplitN(rawURL, "://", 2); len(parts) != 2 {
		// Add scheme.
		rawURL = fmt.Sprintf("postgres://%s", rawURL)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing URL: %v", err)
	}

	parse.SetURLUser(u, c.Username, c.Password)

	if timeout := mod.Config().Timeout; timeout > 0 {
		q := u.Query()
		q.Set("connect_timeout", strconv.Itoa(int(timeout.Seconds())))
		u.RawQuery = q.Encode()
	}

	// https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-CONNSTRING
	connString, err := pq.ParseURL(u.String())
	if err != nil {
		return mb.HostData{}, err
	}

	h := parse.NewHostDataFromURL(u)

	// Store the connection string instead of URL to avoid the cost of sql.Open
	// parsing the URL on each call.
	h.URI = connString

	// Postgres URLs can use a host query param to specify the host. This is
	// used for unix domain sockets (postgres:///dbname?host=/var/lib/postgres).
	if host := u.Query().Get("host"); u.Host == "" && host != "" {
		h.Host = host
	}

	return h, nil
}

//QueryStats makes the database call for a given metric
func QueryStats(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "scanning columns")
	}
	vals := make([][]byte, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i := range vals {
		valPointers[i] = &vals[i]
	}

	results := []map[string]interface{}{}

	for rows.Next() {
		err = rows.Scan(valPointers...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			result[col] = string(vals[i])
		}

		logp.Debug("postgresql", "Result: %v", result)
		results = append(results, result)
	}
	return results, nil
}
