// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

type tempFreeSpace struct {
	TablespaceName string
	TablespaceSize sql.NullInt64
	UsedSpaceBytes sql.NullInt64
	FreeSpace      sql.NullInt64
}

func (d *tempFreeSpace) hash() string {
	return d.TablespaceName
}

func (d *tempFreeSpace) eventKey() string {
	return d.TablespaceName
}

func (e *tablespaceExtractor) tempFreeSpaceData(ctx context.Context) ([]tempFreeSpace, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT TABLESPACE_NAME, TABLESPACE_SIZE, ALLOCATED_SPACE, FREE_SPACE FROM DBA_TEMP_FREE_SPACE")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]tempFreeSpace, 0)

	for rows.Next() {
		dest := tempFreeSpace{}
		if err = rows.Scan(&dest.TablespaceName, &dest.TablespaceSize, &dest.UsedSpaceBytes, &dest.FreeSpace); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
