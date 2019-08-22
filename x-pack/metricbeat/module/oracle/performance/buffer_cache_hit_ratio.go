// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

type bufferCacheHitRatio struct {
	name           sql.NullString
	physicalReads  sql.NullInt64
	dbBlockGets    sql.NullInt64
	consistentGets sql.NullInt64
	hitRatio       sql.NullFloat64
}

/*
	The following function executes a query that produces the following result

	NAME	PHYSICAL_READS	DB_BLOCK_GETS	CONSISTENT_GETS	Hit Ratio
	DEFAULT	19024			17195			379538			0.9520483549389639883750532473981241792339

	Each instance of bufferCacheHitRatio represents a row from the previous results
*/
func (e *performanceExtractor) bufferCacheHitRatio(ctx context.Context) ([]bufferCacheHitRatio, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT name, physical_reads, db_block_gets, consistent_gets,
       1 - (physical_reads / (db_block_gets + consistent_gets)) "Hit Ratio"
FROM V$BUFFER_POOL_STATISTICS`)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]bufferCacheHitRatio, 0)

	for rows.Next() {
		dest := bufferCacheHitRatio{}
		if err = rows.Scan(&dest.name, &dest.physicalReads, &dest.dbBlockGets, &dest.consistentGets, &dest.hitRatio); err != nil {
			return nil, err
		}

		results = append(results, dest)
	}

	return results, nil
}
