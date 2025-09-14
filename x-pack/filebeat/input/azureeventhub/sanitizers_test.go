// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizersSanitize(t *testing.T) {

	// Set up some sanitizers
	nlSanitizer, err := newSanitizer(SanitizerSpec{
		Type: SanitizerNewLines,
	})
	require.NoError(t, err)

	sqSanitizer, err := newSanitizer(SanitizerSpec{
		Type: SanitizerSingleQuotes,
	})
	require.NoError(t, err)

	raSanitizer, err := newSanitizer(SanitizerSpec{
		Type: SanitizerReplaceAll,
		Spec: map[string]interface{}{
			"pattern":     `\[\s*([^\[\]{},\s]+(?:\s+[^\[\]{},\s]+)*)\s*\]`,
			"replacement": "{}",
		},
	})
	require.NoError(t, err)

	testCases := []struct {
		name       string
		sanitizers []Sanitizer
		original   []byte
		expected   []byte
	}{
		{
			name:       "NEW_LINES option",
			sanitizers: []Sanitizer{nlSanitizer},
			original:   []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
			expected:   []byte("{'test':\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:       "SINGLE_QUOTES option",
			sanitizers: []Sanitizer{sqSanitizer},
			original:   []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
			expected:   []byte("{\"test\":\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:       "NEW_LINES + SINGLE_QUOTES option",
			sanitizers: []Sanitizer{nlSanitizer, sqSanitizer},
			original:   []byte("{'test':\"this is 'some' message\n\",\n\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
			expected:   []byte("{\"test\":\"this is 'some' message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"),
		},
		{
			name:       "Replace all option",
			sanitizers: []Sanitizer{raSanitizer},
			original: []byte(`
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
			expected: []byte(`
	{
		"AppImage": "orcas/postgres_standalone_16_u18:38.1.240825",
		"AppType": "PostgreSQL",
		"AppVersion": "breadthpg16_2024-08-06-07-22-43",
		"Region": "westeurope",
		"category": "PostgreSQLLogs",
		"location": "westeurope",
		"operationName": "LogEvent",
		"properties": {},
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
		t.Run(tc.name, func(t *testing.T) {
			res := tc.original

			for _, sanitizer := range tc.sanitizers {
				res = sanitizer.Sanitize(res)
			}

			// using string(..) for readability
			assert.Equal(t, string(tc.expected), string(res))
		})
	}
}

func TestSanitizersInit(t *testing.T) {

	t.Run("Support legacy sanitizers", func(t *testing.T) {
		legacySanitizers := []string{"NEW_LINES", "SINGLE_QUOTES"}

		sanitizers, err := newSanitizers([]SanitizerSpec{}, legacySanitizers)
		require.NoError(t, err)

		require.Len(t, sanitizers, 2)

		// Check the struct types
		assert.IsType(t, &newLinesSanitizer{}, sanitizers[0])
		assert.IsType(t, &singleQuotesSanitizer{}, sanitizers[1])
	})

	t.Run("Support new sanitizers", func(t *testing.T) {
		raSanitizerSpec := SanitizerSpec{
			Type: SanitizerReplaceAll,
			Spec: map[string]any{
				"pattern":     `\[\s*([^\[\]{},\s]+(?:\s+[^\[\]{},\s]+)*)\s*\]`,
				"replacement": "{}",
			},
		}

		sanitizers, err := newSanitizers([]SanitizerSpec{raSanitizerSpec}, nil)
		require.NoError(t, err)

		require.Len(t, sanitizers, 1)

		// Check the struct types
		assert.IsType(t, &replaceAllSanitizer{}, sanitizers[0])
	})

	t.Run("Support legacy and new sanitizer together", func(t *testing.T) {
		raSanitizerSpec := SanitizerSpec{
			Type: SanitizerReplaceAll,
			Spec: map[string]any{
				"pattern":     `\[\s*([^\[\]{},\s]+(?:\s+[^\[\]{},\s]+)*)\s*\]`,
				"replacement": "{}",
			},
		}

		legacySanitizers := []string{"NEW_LINES", "SINGLE_QUOTES"}

		sanitizers, err := newSanitizers([]SanitizerSpec{raSanitizerSpec}, legacySanitizers)
		require.NoError(t, err)

		require.Len(t, sanitizers, 3)

		// Check the struct types
		assert.IsType(t, &replaceAllSanitizer{}, sanitizers[0])
		assert.IsType(t, &newLinesSanitizer{}, sanitizers[1])
		assert.IsType(t, &singleQuotesSanitizer{}, sanitizers[2])
	})
}
