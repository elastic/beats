// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize(t *testing.T) {
	jsonByte := []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}")

	testCases := []struct {
		name     string
		opts     []string
		actual   []byte
		expected []byte
	}{
		{
			name:     "no options",
			opts:     []string{},
			actual:   jsonByte,
			expected: jsonByte,
		},
		{
			name:     "NEW_LINES option",
			opts:     []string{"NEW_LINES"},
			actual:   jsonByte,
			expected: []byte("{'test':\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "SINGLE_QUOTES option",
			opts:     []string{"SINGLE_QUOTES"},
			actual:   jsonByte,
			expected: []byte("{\"test\":\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:     "both options",
			opts:     []string{"NEW_LINES", "SINGLE_QUOTES"},
			actual:   jsonByte,
			expected: []byte("{\"test\":\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name: "REGEXP option",
			opts: []string{"REGEXP"},
			actual: []byte(`
	{
		"AppImage": "orcas/postgres_standalone_16_u18:38.1.240825",
		"AppType": "PostgreSQL",
		"AppVersion": "breadthpg16_2024-08-06-07-22-43",
		"Region": "westeurope",
		"category": "PostgreSQLLogs",
		"location": "westeurope",
		"operationName": "LogEvent",
		"properties": [
			218 B blob data
		],
		"resourceId": "/SUBSCRIPTIONS/88D1708E-CBC3-4799-9C16-C5BB5F57A0C3/RESOURCEGROUPS/CKAUF-AZURE-OBS/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/CHRIS-PG-TEST",
		"time": "2024-08-27T14:26:08.629Z",
		"ServerType": "PostgreSQL",
		"LogicalServerName": "chris-pg-test",
		"ServerVersion": "breadthpg16_2024-08-06-07-22-43",
		"ServerLocation": "prod:westeurope",
		"ReplicaRole": "Primary",
		"OriginalPrimaryServerName": "chris-pg-test"
	}`),
			expected: []byte(`{
		"AppImage": "orcas/postgres_standalone_16_u18:38.1.240825",
		"AppType": "PostgreSQL",
		"AppVersion": "breadthpg16_2024-08-06-07-22-43",
		"Region": "westeurope",
		"category": "PostgreSQLLogs",
		"location": "westeurope",
		"operationName": "LogEvent",
		"properties": ["218 B blob data"],
		"resourceId": "/SUBSCRIPTIONS/88D1708E-CBC3-4799-9C16-C5BB5F57A0C3/RESOURCEGROUPS/CKAUF-AZURE-OBS/PROVIDERS/MICROSOFT.DBFORPOSTGRESQL/FLEXIBLESERVERS/CHRIS-PG-TEST",
		"time": "2024-08-27T14:26:08.629Z",
		"ServerType": "PostgreSQL",
		"LogicalServerName": "chris-pg-test",
		"ServerVersion": "breadthpg16_2024-08-06-07-22-43",
		"ServerLocation": "prod:westeurope",
		"ReplicaRole": "Primary",
		"OriginalPrimaryServerName": "chris-pg-test"
	}`),
		},
	}

	// Run test cases
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			res := sanitize(tc.actual, tc.opts...)
			assert.Equal(t, string(tc.expected), string(res))
		})
	}
}
