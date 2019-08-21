// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
)

type cursorsByUsernameAndMachine struct {
	total    sql.NullInt64
	avg      sql.NullFloat64
	max      sql.NullInt64
	username sql.NullString
	machine  sql.NullString
}

type totalCursors struct {
	totalCursors               sql.NullInt64
	currentCursors             sql.NullInt64
	sessCurCacheHits           sql.NullInt64
	parseCountTotal            sql.NullInt64
	cacheHitsTotalCursorsRatio sql.NullFloat64
	realParses                 sql.NullInt64
}

func (e *performanceExtractor) cursorsByUsernameAndMachine(ctx context.Context) ([]cursorsByUsernameAndMachine, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT sum(a.value) total_cur, 
		       avg(a.value) avg_cur, 
		       max(a.value) max_cur,
       		   s.username,
		       s.machine
		FROM v$sesstat a, v$statname b, v$session s
		WHERE a.statistic# = b.statistic#  
		  AND s.sid = a.sid
		  AND b.name = 'opened cursors current'
		GROUP BY s.username, 
		         s.machine
		ORDER BY 1 DESC`)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]cursorsByUsernameAndMachine, 0)

	for rows.Next() {
		dest := cursorsByUsernameAndMachine{}
		if err = rows.Scan(&dest.total, &dest.avg, &dest.max, &dest.username, &dest.machine); err != nil {
			return nil, err
		}

		// Username could be nil if it's unknown. If any value of username is nil, set it as unknown instead. Same for machine
		if !dest.username.Valid || dest.username.String == "" {
			dest.username.String = "Unknown"
			dest.username.Valid = true
		}

		if !dest.machine.Valid || dest.machine.String == "" {
			dest.machine.String = "Unknown"
			dest.machine.Valid = true
		}

		results = append(results, dest)
	}

	return results, nil
}

func (e *performanceExtractor) totalCursors(ctx context.Context) (*totalCursors, error) {
	rows := e.db.QueryRowContext(ctx, `
		SELECT total_cursors, 
		       current_cursors, 
		       sess_cur_cache_hits, 
		       parse_count_total, 
		       sess_cur_cache_hits / total_cursors, 
		       sess_cur_cache_hits - parse_count_total
		FROM (
		    SELECT sum ( decode ( name, 'opened cursors cumulative', value, 0)) total_cursors,
			   sum ( decode ( name, 'opened cursors current',value,0)) current_cursors,
			   sum ( decode ( name, 'session cursor cache hits',value,0)) sess_cur_cache_hits,
			   sum ( decode ( name, 'parse count (total)',value,0)) parse_count_total
			FROM v$sysstat
			WHERE name IN ( 'opened cursors cumulative','opened cursors current','session cursor cache hits', 'parse count (total)' ))`)

	dest := totalCursors{}

	err := rows.Scan(&dest.totalCursors, &dest.currentCursors, &dest.sessCurCacheHits, &dest.parseCountTotal, &dest.cacheHitsTotalCursorsRatio, &dest.realParses)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}
