// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"database/sql"
	"github.com/pkg/errors"
)

type freeSpace struct {
	TablespaceName string
	TotalBytes     sql.NullInt64
}

func (d *freeSpace) hash() string {
	return d.TablespaceName
}

func (d *freeSpace) eventKey() string {
	return d.TablespaceName
}

func (e *tablespaceExtractor) freeSpaceData() ([]freeSpace, error) {
	rows, err := e.db.Query("SELECT TABLESPACE_NAME, TOTAL_BYTES FROM sys.DBA_FREE_SPACE_COALESCED")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]freeSpace, 0)

	for rows.Next() {
		dest := freeSpace{}
		if err = rows.Scan(&dest.TablespaceName, &dest.TotalBytes); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
