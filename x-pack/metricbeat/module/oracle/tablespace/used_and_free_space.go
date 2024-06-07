// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"
	"fmt"
)

type usedAndFreeSpace struct {
	TablespaceName  string
	TotalSpaceBytes sql.NullInt64
	TotalFreeBytes  sql.NullInt64
	TotalUsedBytes  sql.NullInt64
}

func (e *tablespaceExtractor) usedAndFreeSpaceData(ctx context.Context) ([]usedAndFreeSpace, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT b.tablespace_name, (b.tbs_size - NVL(a.free_space, 0)) AS used, NVL(a.free_space, 0) AS free, (SELECT SUM(bytes) FROM DBA_DATA_FILES) + (SELECT SUM(bytes) FROM DBA_TEMP_FILES) AS total_sum FROM (SELECT tablespace_name, SUM(bytes) AS free_space FROM DBA_FREE_SPACE GROUP BY tablespace_name) a RIGHT JOIN (SELECT tablespace_name, SUM(bytes) AS tbs_size FROM DBA_DATA_FILES GROUP BY tablespace_name) b ON a.tablespace_name = b.tablespace_name`)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	results := make([]usedAndFreeSpace, 0)

	for rows.Next() {
		dest := usedAndFreeSpace{}
		if err = rows.Scan(&dest.TablespaceName, &dest.TotalUsedBytes, &dest.TotalFreeBytes, &dest.TotalSpaceBytes); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
