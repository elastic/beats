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

//go:build windows

package wmi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewDefaultConfig verifies the default values for the Config struct.
func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.False(t, cfg.IncludeQueries, "IncludeQueries should default to false")
	assert.False(t, cfg.IncludeNullProperties, "IncludeNullProperties should default to false")
	assert.False(t, cfg.IncludeEmptyStringProperties, "IncludeEmptyStringProperties should default to false")
	assert.Equal(t, WMIDefaultNamespace, cfg.Namespace, "Namespace should default to WMIDefaultNamespace")
	assert.Empty(t, cfg.Queries, "Queries should default to an empty slice")
}

// TestValidateConnectionParameters checks the validation logic for user and password.
func TestValidateConnectionParameters(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name:          "Valid user and password",
			config:        Config{User: "admin", Password: "password"},
			expectedError: "",
		},
		{
			name:          "User without password",
			config:        Config{User: "admin"},
			expectedError: "if user is set, password should be set",
		},
		{
			name:          "Password without user",
			config:        Config{Password: "password"},
			expectedError: "if password is set, user should be set",
		},
		{
			name:          "No user and no password",
			config:        Config{},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConnectionParameters()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

// TestCompileQueries ensures queries are properly compiled.
func TestCompileQueries(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedError string
	}{
		{
			name: "Valid queries",
			config: Config{
				Queries: []QueryConfig{
					{
						Class:      "Win32_Process",
						Properties: []string{"Name", "ID"},
						Where:      "Name LIKE 'chrome%'",
					},
				},
			},
			expectedError: "",
		},
		{
			name:          "No queries defined",
			config:        Config{},
			expectedError: "at least one query is needed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.CompileQueries()
			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.NotEmpty(t, tt.config.Queries[0].QueryStr, "QueryStr should be compiled and not empty")
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

// TestQueryCompile ensures individual query compilation works correctly.
func TestQueryCompile(t *testing.T) {
	tests := []struct {
		name                     string
		queryConfig              QueryConfig
		expectedQuery            string
		expectedPropertiesLength int
	}{
		{
			name: "Simple query with WHERE clause",
			queryConfig: QueryConfig{
				Class:      "Win32_Process",
				Properties: []string{"Name", "ProcessId"},
				Where:      "Name = 'notepad.exe'",
			},
			expectedQuery:            "SELECT Name,ProcessId FROM Win32_Process WHERE Name = 'notepad.exe'",
			expectedPropertiesLength: 2,
		},
		{
			name: "Query with multiple properties and no WHERE clause",
			queryConfig: QueryConfig{
				Class:      "Win32_Service",
				Properties: []string{"Name", "State", "StartMode"},
				Where:      "",
			},
			expectedQuery:            "SELECT Name,State,StartMode FROM Win32_Service",
			expectedPropertiesLength: 3,
		},
		{
			name: "Query with  empty list for properties and Where",
			queryConfig: QueryConfig{
				Class:      "Win32_ComputerSystem",
				Properties: []string{},
				Where:      "Manufacturer = 'Dell'",
			},
			expectedQuery:            "SELECT * FROM Win32_ComputerSystem WHERE Manufacturer = 'Dell'",
			expectedPropertiesLength: 0,
		},
		{
			name: "Query with wildcard (*) and no WHERE clause",
			queryConfig: QueryConfig{
				Class:      "Win32_BIOS",
				Properties: []string{"*"},
				Where:      "",
			},
			expectedQuery:            "SELECT * FROM Win32_BIOS",
			expectedPropertiesLength: 0, // The normalization process make sure that ['*'] and [] are the same case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.queryConfig.compileQuery()
			assert.Equal(t, tt.expectedQuery, tt.queryConfig.QueryStr, "QueryStr should match the expected query string")
			assert.Equal(t, tt.expectedPropertiesLength, len(tt.queryConfig.Properties))
		})
	}
}

func TestBuildNamespaceQueryIndex(t *testing.T) {

	defaultNamespace := "root\\cimv2"
	upperCaseDefaultNamespace := "ROOT\\CIMV2"

	tests := []struct {
		name          string
		queries       []QueryConfig
		expectedIndex map[string][]QueryConfig
		description   string
	}{
		{
			name: "Single query, single namespace",
			queries: []QueryConfig{
				{Namespace: defaultNamespace},
			},
			expectedIndex: map[string][]QueryConfig{
				WMIDefaultNamespace: {
					{Namespace: defaultNamespace},
				},
			},
			description: "Should build an index with a single query in the 'default' namespace",
		},
		{
			name: "Multiple queries, same namespace, different spells",
			queries: []QueryConfig{
				{Namespace: defaultNamespace},
				{Namespace: upperCaseDefaultNamespace},
			},
			expectedIndex: map[string][]QueryConfig{
				defaultNamespace: {
					{Namespace: defaultNamespace},
					{Namespace: upperCaseDefaultNamespace},
				},
			},
			description: "Should correctly handle multiple queries in the same namespace",
		},
		{
			name: "Multiple queries, different namespaces",
			queries: []QueryConfig{
				{Namespace: "default"},
				{Namespace: "custom"},
			},
			expectedIndex: map[string][]QueryConfig{
				"default": {
					{Namespace: "default"},
				},
				"custom": {
					{Namespace: "custom"},
				},
			},
			description: "Should correctly build separate indices for different namespaces",
		},
		{
			name:          "Empty queries",
			queries:       []QueryConfig{},
			expectedIndex: map[string][]QueryConfig{},
			description:   "Should return an empty index when no queries are provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the config and assign queries
			config := &Config{Queries: tt.queries}

			// Build the namespace index
			config.BuildNamespaceQueryIndex()

			// Assert that the namespace index matches the expected result
			assert.Equal(t, tt.expectedIndex, config.NamespaceQueryIndex, tt.description)
		})
	}
}
