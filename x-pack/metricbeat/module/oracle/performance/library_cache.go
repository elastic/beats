// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package performance

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type libraryCache struct {
	name  sql.NullString
	value sql.NullFloat64
}

/*
 * The following function executes a query that produces the following result
 *
 * Ratio			AVG(GETHITRATIO)
 * io_reloads		0.004130404962582493789151209400862660224657
 * lock_requests	0.4902639097748147904635170747058646829122
 * pin_requests	0.755804477116177544453478080666426614297
 *
 * Which is parsed into libraryCache instances
 */
func (e *performanceExtractor) libraryCache(ctx context.Context) ([]libraryCache, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT 'lock_requests' "Ratio" , AVG(gethitratio) FROM V$LIBRARYCACHE
		UNION
		SELECT 'pin_requests' "Ratio", AVG(pinhitratio) FROM V$LIBRARYCACHE
		UNION
		SELECT 'io_reloads' "Ratio", (SUM(reloads) / SUM(pins)) FROM V$LIBRARYCACHE`)
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]libraryCache, 0)

	for rows.Next() {
		dest := libraryCache{}
		if err = rows.Scan(&dest.name, &dest.value); err != nil {
			return nil, err
		}

		results = append(results, dest)
	}

	return results, nil
}

func (m *MetricSet) addLibraryCacheData(ls []libraryCache) mapstr.M {
	out := mapstr.M{}

	for _, v := range ls {
		if v.name.Valid {
			oracle.SetSqlValue(m.Logger(), out, v.name.String, &oracle.Float64Value{NullFloat64: v.value})
		}
	}

	return out
}
