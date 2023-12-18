// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"testing"

	"github.com/g8rswimmer/go-sfdc/soql"
	"github.com/stretchr/testify/assert"
)

// compile-time check if querier implements soql.QueryFormatter
var _ soql.QueryFormatter = (*querier)(nil)

func TestFormat(t *testing.T) {
	tests := map[string]struct {
		input   string
		wantStr string
		wantErr error
	}{
		"empty query":   {input: "", wantStr: "", wantErr: errors.New("query is empty")},
		"valid query":   {input: "SELECT FIELDS(STANDARD) FROM LoginEvent", wantStr: "SELECT FIELDS(STANDARD) FROM LoginEvent", wantErr: nil},
		"invalid query": {input: "SELECT <bad query>", wantStr: "SELECT <bad query>", wantErr: nil},
	}

	var q querier

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			q.Query = tc.input
			got, gotErr := q.Format()
			if !assert.Equal(t, tc.wantErr, gotErr) {
				t.FailNow()
			}
			if !assert.EqualValues(t, tc.wantStr, got) {
				t.FailNow()
			}
		})
	}
}
