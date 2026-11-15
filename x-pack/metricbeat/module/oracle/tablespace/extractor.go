// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"
)

// tablespaceExtractMethods contains the methods needed to extract the necessary information about a Tablespace
type tablespaceExtractMethods interface {
	dataFilesData(context.Context) ([]dataFile, error)
	tempFreeSpaceData(context.Context) ([]tempFreeSpace, error)
	usedAndFreeSpaceData(context.Context) ([]usedAndFreeSpace, error)
}

// extractedData contains the necessary tablespace information. Can be updated with more data without affecting methods
// signatures.
type extractedData struct {
	dataFiles     []dataFile
	freeSpace     []usedAndFreeSpace
	tempFreeSpace []tempFreeSpace
}

// tablespaceExtractor is the implementor of tablespaceExtractMethods. It's implementation are on different Go files
// which refers to the origin of the data for organization purposes.
type tablespaceExtractor struct {
	db *sql.DB
}
