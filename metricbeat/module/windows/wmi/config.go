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

// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package wmi

import (
	"fmt"

	wmiquery "github.com/microsoft/wmi/pkg/base/query"
)

type Config struct {
	IncludeQueries bool          `config:"wmi.include_queries"` // Whether to include the query in the document
	IncludeNull    bool          `config:"wmi.include_null"`    // Whether to include or not nil properties
	Host           string        `config:"wmi.host"`            // Remote WMI Host
	User           string        `config:"wmi.username"`        // Username for the Remote WMI
	Password       string        `config:"wmi.password"`        // Password for the Remote WMI
	Namespace      string        `config:"wmi.namespace"`       // Namespace for the queries
	Queries        []QueryConfig `config:"wmi.queries"`         // List of query definitions
}

type QueryConfig struct {
	QueryStr string
	Class    string   `config:"class"`
	Fields   []string `config:"fields"`
	Where    string   `config:"where"`
}

func NewDefaultConfig() Config {
	return Config{
		IncludeQueries: false,
		IncludeNull:    false,
		Host:           "localhost",
		Namespace:      WMIDefaultNamespace,
	}
}

func (c *Config) ValidateConnectionParameters() error {
	if c.User != "" && c.Password == "" {
		return fmt.Errorf("if user is set, password should be set")
	} else if c.User == "" && c.Password != "" {
		return fmt.Errorf("if password is set, user should be set")
	}
	return nil
}

func (qc *QueryConfig) compileQuery() {
	// Let us normalize the case where the array is ['*']
	// To the Empty Array
	if len(qc.Fields) == 1 && qc.Fields[0] == "*" {
		qc.Fields = []string{}
	}

	query := wmiquery.NewWmiQueryWithSelectList(qc.Class, qc.Fields, []string{}...)
	queryStr := query.String()
	// Concatenating the where clause manually, because the library supports only a subset of where clauses
	// while we want to leverage all filtering capabilities
	if qc.Where != "" {
		queryStr += " WHERE " + qc.Where
	}
	qc.QueryStr = queryStr
}

func (c *Config) CompileQueries() error {
	if len(c.Queries) == 0 {
		return fmt.Errorf("at least a query is needed")
	}

	for i := range c.Queries {
		c.Queries[i].compileQuery()
	}
	return nil
}
