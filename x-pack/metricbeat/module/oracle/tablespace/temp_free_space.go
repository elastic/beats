// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"
	"fmt"
)

type tempFreeSpace struct {
	TablespaceName  string
	TotalSpaceBytes sql.NullInt64
	UsedSpaceBytes  sql.NullInt64
	FreeSpace       sql.NullInt64
}

func (e *tablespaceExtractor) tempFreeSpaceData(ctx context.Context) ([]tempFreeSpace, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT t.TABLESPACE_NAME, (SELECT SUM(BYTES) FROM DBA_DATA_FILES) + (SELECT SUM(BYTES) FROM DBA_TEMP_FILES) AS TOTAL_SUM, t.ALLOCATED_SPACE, t.FREE_SPACE FROM DBA_TEMP_FREE_SPACE t `)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	results := make([]tempFreeSpace, 0)

	for rows.Next() {
		dest := tempFreeSpace{}
		if err = rows.Scan(&dest.TablespaceName, &dest.TotalSpaceBytes, &dest.UsedSpaceBytes, &dest.FreeSpace); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
