// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"strings"

	"github.com/g8rswimmer/go-sfdc/soql"
)

// compile-time check if querier implements soql.QueryFormatter
var _ soql.QueryFormatter = (*querier)(nil)

type querier struct {
	Query string
}

// Format returns the query string.
func (q querier) Format() (string, error) {
	if strings.TrimSpace(q.Query) == "" {
		return "", errors.New("query is empty")
	}
	return q.Query, nil
}
