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

//go:build windows

package wmi

import (
	"fmt"
	"strings"
	"time"

	wmiquery "github.com/microsoft/wmi/pkg/base/query"
)

type Config struct {
	IncludeQueries               bool                     `config:"wmi.include_queries"`                 // Determines if the query string should be included in the output document
	IncludeNullProperties        bool                     `config:"wmi.include_null_properties"`         // Specifies whether to include properties with nil values in the final document
	IncludeEmptyStringProperties bool                     `config:"wmi.include_empty_string_properties"` // Specifies whether to include properties with empty string values in the final document
	Namespace                    string                   `config:"wmi.namespace"`                       // Default WMI namespace for executing queries, used if not overridden by individual query configurations
	Queries                      []QueryConfig            `config:"wmi.queries"`                         // List of WMI query configurations
	WarningThreshold             time.Duration            `config:"wmi.warning_threshold"`               // Maximum duration to wait for query results before logging a warning. The query will continue running in WMI but will no longer be awaited
	NamespaceQueryIndex          map[string][]QueryConfig // Internal structure indexing queries by namespace to reduce the number of WMI connections required per execution
	// Remote WMI Parameters
	// These parameters are intentionally hidden to discourage their use.
	// If you need access, please open a support ticket to request exposure.
	Host     string // Hostname of the remote WMI server
	Domain   string // Domain of the remote WMI Server
	User     string // Username for authentication on the remote WMI server
	Password string // Password for authentication on the remote WMI server
}

type QueryConfig struct {
	QueryStr   string   // The compiled query string generated internally (not user-configurable)
	Class      string   `config:"class"`      // WMI class to query (used in the FROM clause)
	Properties []string `config:"properties"` // List of properties to retrieve (used in the SELECT clause). If omitted, all properties of the class are fetched
	Where      string   `config:"where"`      // Custom WHERE clause to filter query results. The provided string is used directly in the query
	Namespace  string   `config:"namespace"`  // WMI namespace for the query. This takes precedence over the globally configured namespace
}

func NewDefaultConfig() Config {
	return Config{
		IncludeQueries:               false,
		IncludeNullProperties:        false,
		IncludeEmptyStringProperties: false,
		Host:                         "localhost",
		Namespace:                    WMIDefaultNamespace,
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
	if len(qc.Properties) == 1 && qc.Properties[0] == "*" {
		qc.Properties = []string{}
	}

	query := wmiquery.NewWmiQueryWithSelectList(qc.Class, qc.Properties, []string{}...)
	queryStr := query.String()
	// Concatenating the where clause manually, because the library supports only a subset of where clauses
	// while we want to leverage all filtering capabilities
	if qc.Where != "" {
		queryStr += " WHERE " + qc.Where
	}
	qc.QueryStr = queryStr
}

func (qc *QueryConfig) applyDefaultNamespace(defaultNamespace string) {
	if qc.Namespace == "" {
		qc.Namespace = defaultNamespace
	}
}

func (c *Config) CompileQueries() error {
	if len(c.Queries) == 0 {
		return fmt.Errorf("at least one query is needed")
	}

	for i := range c.Queries {
		c.Queries[i].compileQuery()
	}
	return nil
}

func (c *Config) ApplyDefaultNamespaceToQueries(defaultNamespace string) error {
	for i := range c.Queries {
		c.Queries[i].applyDefaultNamespace(defaultNamespace)
	}

	return nil
}

func (c *Config) BuildNamespaceQueryIndex() {
	c.NamespaceQueryIndex = make(map[string][]QueryConfig)
	for _, q := range c.Queries {
		// WMI namespaces are case-insensitive. We are building a case-insensitive map
		// to ensure that different variations of the namespace (e.g., "root\\cimv2" and "ROOT\\CIMV2")
		// are treated as the same and grouped together.
		namespace := strings.ToLower(q.Namespace)
		_, ok := c.NamespaceQueryIndex[namespace]
		if !ok {
			c.NamespaceQueryIndex[namespace] = []QueryConfig{}
		}
		c.NamespaceQueryIndex[namespace] = append(c.NamespaceQueryIndex[namespace], q)
	}
}
