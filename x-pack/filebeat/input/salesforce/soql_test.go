// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import "github.com/g8rswimmer/go-sfdc/soql"

// compile-time check if querier implements soql.QueryFormatter
var _ soql.QueryFormatter = (*querier)(nil)
