// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
package interfaces

import (
	"github.com/osquery/osquery-go/gen/osquery"
)

// QueryExecutor is an interface for executing osquery queries
type QueryExecutor interface {
	Query(sql string) (*osquery.ExtensionResponse, error)
}
