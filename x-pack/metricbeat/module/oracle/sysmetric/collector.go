// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"database/sql"
)

// sysmetricCollectMethod contains the methods needed to collect the necessary information about the performance of the database.
type sysmetricCollectMethod interface {
	sysmetricMetric(context.Context) ([]sysmetricMetric, error)
}

// collectedData contains the necessary system metric information.
type collectedData struct {
	sysmetricMetrics []sysmetricMetric
}

// sysmetricCollector is the implementor of sysmetricCollectMethod. It's implementation are on different Go files
// which refers to the origin of the data for organization purposes.
type sysmetricCollector struct {
	db       *sql.DB
	patterns []interface{}
}
