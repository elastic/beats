// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"database/sql"
	"fmt"
)

type dataFile struct {
	FileName              sql.NullString
	FileID                sql.NullInt64
	TablespaceName        sql.NullString
	FileSizeBytes         sql.NullInt64
	Status                sql.NullString
	MaxFileSizeBytes      sql.NullInt64
	AvailableForUserBytes sql.NullInt64
	OnlineStatus          sql.NullString
}

func (d *dataFile) hash() string {
	return fmt.Sprintf("%s%d", d.TablespaceName.String, d.FileID.Int64)
}

func (d *dataFile) eventKey() string {
	return d.TablespaceName.String
}

func (e *tablespaceExtractor) dataFilesData(ctx context.Context) ([]dataFile, error) {
	rows, err := e.db.QueryContext(ctx, "SELECT FILE_NAME, FILE_ID, TABLESPACE_NAME, BYTES, STATUS, MAXBYTES, USER_BYTES, ONLINE_STATUS FROM SYS.DBA_DATA_FILES UNION SELECT FILE_NAME, FILE_ID, TABLESPACE_NAME, BYTES, STATUS, MAXBYTES, USER_BYTES, STATUS AS ONLINE_STATUS FROM SYS.DBA_TEMP_FILES")
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}

	results := make([]dataFile, 0)

	for rows.Next() {
		dest := dataFile{}
		if err = rows.Scan(&dest.FileName, &dest.FileID, &dest.TablespaceName, &dest.FileSizeBytes, &dest.Status, &dest.MaxFileSizeBytes, &dest.AvailableForUserBytes, &dest.OnlineStatus); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}
