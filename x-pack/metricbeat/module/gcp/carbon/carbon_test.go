// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package carbon

import (
	"errors"
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	for _, tt := range []struct {
		name string

		config        config
		expectedError error
	}{
		{
			name:          "with an empty config",
			config:        config{},
			expectedError: errors.New("no credentials_file_path or credentials_json specified"),
		},
		{
			name: "with a period lower than 24 hours",
			config: config{
				CredentialsJSON: "{}",
			},
			expectedError: errors.New("collection period for carbon footprint metricset 0s cannot be less than 24 hours"),
		},
		{
			name: "with all required fields filled",
			config: config{
				CredentialsJSON: "{}",
				Period:          25 * time.Hour,
			},
			expectedError: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedError, tt.config.Validate())
		})
	}
}

func TestGetReportMonth(t *testing.T) {
	for _, tt := range []struct {
		name string

		now           time.Time
		expectedValue string
	}{
		{
			name:          "with a date before the 15",
			now:           time.Date(2023, 03, 12, 0, 0, 0, 0, time.UTC),
			expectedValue: "2023-02-01",
		},
		{
			name:          "with a date after the 15",
			now:           time.Date(2023, 03, 16, 0, 0, 0, 0, time.UTC),
			expectedValue: "2023-03-01",
		},
		{
			name:          "with a month before matching the previous year",
			now:           time.Date(2023, 01, 03, 0, 0, 0, 0, time.UTC),
			expectedValue: "2022-12-01",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedValue, getReportMonth(tt.now))
		})
	}
}

func TestGenerateQuery(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	query := generateQuery("my-table", "jan")

	// verify that table name quoting is in effect
	assert.Contains(t, query, "`my-table`")
	// verify the order by is preserved
	assert.Contains(t, query, "ORDER BY usage_month ASC")
}
