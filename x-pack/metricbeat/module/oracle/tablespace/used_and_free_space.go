// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

type usedAndFreeSpace struct {
	TablespaceName string
	TotalFreeBytes sql.NullInt64
	TotalUsedBytes sql.NullInt64
}

func (d *usedAndFreeSpace) hash() string {
	return d.TablespaceName
}

func (d *usedAndFreeSpace) eventKey() string {
	return d.TablespaceName
}

func (e *tablespaceExtractor) usedAndFreeSpaceData(ctx context.Context) ([]usedAndFreeSpace, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT b.tablespace_name, tbs_size used, a.free_space free FROM (SELECT tablespace_name, sum(bytes) AS free_space FROM dba_free_space GROUP BY tablespace_name) a, (SELECT tablespace_name, sum(bytes) AS tbs_size FROM dba_data_files GROUP BY tablespace_name) b WHERE a.tablespace_name(+)=b.tablespace_name")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]usedAndFreeSpace, 0)

	for rows.Next() {
		dest := usedAndFreeSpace{}
		if err = rows.Scan(&dest.TablespaceName, &dest.TotalUsedBytes, &dest.TotalFreeBytes); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
