// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"
)

// performanceExtractMethods contains the methods needed to extract the necessary information about a the performance of the database
type performanceExtractMethods interface {
	bufferCacheHitRatio(context.Context) ([]bufferCacheHitRatio, error)
	library(context.Context) ([]library, error)
}

// extractedData contains the necessary performance information. Can be updated with more data without affecting methods
// signatures.
type extractedData struct {
	bufferCacheHitRatios []bufferCacheHitRatio
	libraryData          []library
}

// performanceExtractor is the implementor of performanceExtractMethods. It's implementation are on different Go files
// which refers to the origin of the data for organization purposes.
type performanceExtractor struct {
	db *sql.DB
}
