// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

type library struct {
	name  sql.NullString
	value sql.NullFloat64
}

func (e *performanceExtractor) library(ctx context.Context) ([]library, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT 'lock_requests' "Ratio" , AVG(gethitratio) FROM V$LIBRARYCACHE
		UNION
		SELECT 'pin_requests' "Ratio", AVG(pinhitratio) FROM V$LIBRARYCACHE
		UNION
		SELECT 'io_reloads' "Ratio", (SUM(reloads) / SUM(pins)) FROM V$LIBRARYCACHE`)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]library, 0)

	for rows.Next() {
		dest := library{}
		if err = rows.Scan(&dest.name, &dest.value); err != nil {
			return nil, err
		}

		results = append(results, dest)
	}

	return results, nil
}
