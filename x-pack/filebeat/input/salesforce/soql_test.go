package salesforce

import "github.com/g8rswimmer/go-sfdc/soql"

// compile-time check if querier implements soql.QueryFormatter
var _ soql.QueryFormatter = (*querier)(nil)
