// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/x-pack/metricbeat/module/oracle"
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

/*
 * The following function executes a query that produces the following result
 *
 * TOTAL_CUR	AVG_CUR	MAX_CUR	USERNAME	MACHINE
 * 25			0.6410	17					2ed9ac3a4c3d
 * 2			2		2		SYS			mcastro
 * 0			0		0		SYS			2ed9ac3a4c3d
 *
 * Which are parsed into different cursorsByUsernameAndMachine instances
 */
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

func (m *MetricSet) addCursorByUsernameAndMachine(cs []cursorsByUsernameAndMachine) []common.MapStr {
	out := make([]common.MapStr, 0)

	for _, v := range cs {
		ms := common.MapStr{}

		oracle.SetSqlValue(m.Logger(), ms, "username", &oracle.StringValue{NullString: v.username})
		oracle.SetSqlValue(m.Logger(), ms, "machine", &oracle.StringValue{NullString: v.machine})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.total", &oracle.Int64Value{NullInt64: v.total})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.max", &oracle.Int64Value{NullInt64: v.max})
		oracle.SetSqlValue(m.Logger(), ms, "cursors.avg", &oracle.Float64Value{NullFloat64: v.avg})

		out = append(out, ms)
	}

	return out
}

/*
 * The following function executes a query that produces the following result
 *
 * TOTAL_CURSORS	CURRENT_CURSORS	SESS_CUR_CACHE_HITS	PARSE_COUNT_TOTAL	SESS_CUR_CACHE_HITS/TOTAL_CURSORS	SESS_CUR_CACHE_HITS-PARSE_COUNT_TOTAL
 * 2278			3				2814				992					1.23529411764705882352941176		1822
 *
 * Which is parsed into a totalCursors instance
 */
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

func (m *MetricSet) addCursorData(cs *totalCursors) common.MapStr {
	out := make(common.MapStr)

	oracle.SetSqlValue(m.Logger(), out, "cursors.opened.total", &oracle.Int64Value{NullInt64: cs.totalCursors})
	oracle.SetSqlValue(m.Logger(), out, "cursors.opened.current", &oracle.Int64Value{NullInt64: cs.currentCursors})
	oracle.SetSqlValue(m.Logger(), out, "cursors.session.cache_hits", &oracle.Int64Value{NullInt64: cs.sessCurCacheHits})
	oracle.SetSqlValue(m.Logger(), out, "cursors.parse.total", &oracle.Int64Value{NullInt64: cs.parseCountTotal})
	oracle.SetSqlValue(m.Logger(), out, "cursors.cache_hit.pct", &oracle.Float64Value{NullFloat64: cs.cacheHitsTotalCursorsRatio})
	oracle.SetSqlValue(m.Logger(), out, "cursors.parse.real", &oracle.Int64Value{NullInt64: cs.realParses})

	return out
}
