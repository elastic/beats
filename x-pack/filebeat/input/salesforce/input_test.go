// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"testing"
	"time"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
)

func TestFormQueryWithCursor(t *testing.T) {
	mockTimeNow(time.Date(2023, time.May, 18, 12, 0, 0, 0, time.UTC))
	t.Cleanup(resetTimeNow)

	tests := map[string]struct {
		initialInterval     time.Duration
		defaultSOQLTemplate string
		valueSOQLTemplate   string
		wantQuery           string
		cursor              *state
		wantErr             error
	}{
		"valid soql templates with nil cursor": { // expect default query with LogDate > initialInterval
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > 2023-03-19T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              &state{},
		},
		"valid soql templates with non-empty .cursor.logdate": { // expect value SOQL query with .cursor.logdate set
			initialInterval:     60 * 24 * time.Hour, // 60 * 24h = 60 days (2 months)
			defaultSOQLTemplate: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND Logdate > [[ .var.initial_interval ]] ORDER BY CreatedDate ASC NULLS FIRST",
			valueSOQLTemplate:   "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST",
			wantQuery:           "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > 2023-05-18T12:00:00Z ORDER BY CreatedDate ASC NULLS FIRST",
			cursor:              &state{LogDateTime: timeNow().Format(formatRFC3339Like)},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			v1, v2 := &valueTpl{}, &valueTpl{}

			err := v1.Unpack(tc.defaultSOQLTemplate)
			assert.NoError(t, err)

			err = v2.Unpack(tc.valueSOQLTemplate)
			assert.NoError(t, err)

			queryConfig := &QueryConfig{
				Default: v1,
				Value:   v2,
			}

			sfInput := &salesforceInput{
				config: config{InitialInterval: tc.initialInterval},
				log:    logp.L().With("hello", "world"),
				cursor: tc.cursor,
			}

			querier, err := sfInput.FormQueryWithCursor(queryConfig)
			assert.NoError(t, err)

			assert.EqualValues(t, tc.wantQuery, querier.Query)
		})
	}
}
