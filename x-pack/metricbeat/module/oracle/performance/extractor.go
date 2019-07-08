// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import "database/sql"

// performanceExtractMethods contains the methods needed to extract the necessary information about a the performance of the database
type performanceExtractMethods interface {
	bufferCacheHitRatio() (bufferCacheHitRatio, error)
}

// performanceExtractor is the implementor of performanceExtractMethods. It's implementation are on different Go files
// which refers to the origin of the data for organization purposes.
type performanceExtractor struct {
	db *sql.DB
}
