// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type sysmetricMetric struct {
	groupId sql.NullInt64
	name    sql.NullString
	value   sql.NullFloat64
}

/*
 * The following function executes a query that produces the following result
 *
 *	GROUP_ID	METRIC_NAME								VALUE	METRIC_UNIT
 *	2 			PX operations not downgraded Per Sec	0		PX Operations Per Second                                                    0
 *
 * Which is parsed into sysmetricMetric instances
 */
func (e *sysmetricExtractor) calculateQuery() string {
	if len(e.patterns) == 0 {
		e.patterns = make([]interface{}, 1)
		e.patterns[0] = "%"
	}
	query := "SELECT GROUP_ID, METRIC_NAME, VALUE FROM V$SYSMETRIC WHERE METRIC_NAME LIKE :pattern0"
	for i := 1; i < len(e.patterns); i++ {
		query = query + " OR METRIC_NAME LIKE :pattern" + strconv.Itoa(i)
	}
	return query
}

func (e *sysmetricExtractor) sysmetricMetric(ctx context.Context) ([]sysmetricMetric, error) {
	query := e.calculateQuery()
	rows, err := e.db.QueryContext(ctx, query, e.patterns...)
	if err != nil {
		return nil, fmt.Errorf("error executing query %w", err)
	}

	results := make([]sysmetricMetric, 0)

	for rows.Next() {
		dest := sysmetricMetric{}
		if err = rows.Scan(&dest.groupId, &dest.name, &dest.value); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	return results, nil
}

func (m *MetricSet) addSysmetricData(bs []sysmetricMetric) []mapstr.M {
	out := make([]mapstr.M, 0)

	ms := mapstr.M{}

	for _, sysmetricMetric := range bs {
		metricName := ConvertToSnakeCase(sysmetricMetric.name).String
		oracle.SetSqlValue(m.Logger(), ms, "metrics."+metricName+"_"+strconv.Itoa(int(sysmetricMetric.groupId.Int64)), &oracle.Float64Value{NullFloat64: sysmetricMetric.value})
	}
	out = append(out, ms)

	return out
}

// ConvertToSnakeCase function converts a string to snake case to follow
// the Elastic naming conventions in the dynamically mapped fields
func ConvertToSnakeCase(name sql.NullString) sql.NullString {
	reg, _ := regexp.Compile("[()/]") // Regex to remove '(', ')' and '/' characters from the string
	// Convert to lowercase, replace spaces and hyphens with '_' and replace '%' with 'pct'
	str := name.String
	str = strings.ToLower(str)
	str = strings.ReplaceAll(str, " ", "_")
	str = reg.ReplaceAllString(str, "")
	str = strings.ReplaceAll(str, "%", "pct")
	str = strings.ReplaceAll(str, "-", "_")
	name.String = str
	return name
}
