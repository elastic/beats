// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package sysmetric

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
)

type sysmetricMetric struct {
	beginTime   sql.NullString
	endTime     sql.NullString
	intsizeCsec sql.NullFloat64
	groupId     sql.NullInt64
	metricId    sql.NullInt64
	name        sql.NullString
	value       sql.NullFloat64
	metricUnit  sql.NullString
	conId       sql.NullFloat64
}

/*
 * The following function executes a query that produces the following result
 *
 * BEGIN_TIM END_TIME  INTSIZE_CSEC   GROUP_ID  METRIC_ID
 * METRIC_NAME                                                           VALUE
 * METRIC_UNIT                                                          CON_ID
 * 19-APR-22 19-APR-22         6042          2       2146
 * I/O Requests per Second                                          2.99569679
 * Requests per Second                                                       0
 *
 * Which is parsed into sysmetricMetric instances
 */

func (e *sysmetricExtractor) calQuery() string {
	if len(e.patterns) == 0 {
		e.patterns = []string{"%"}
	}
	query := "SELECT * FROM V$SYSMETRIC WHERE (" + "METRIC_NAME LIKE '" + e.patterns[0] + "'"
	for i := 1; i < len(e.patterns); i++ {
		query = query + " OR " + "METRIC_NAME LIKE '" + e.patterns[i] + "'"
	}
	query = query + ")"
	return query
}

func (e *sysmetricExtractor) sysmetricMetric(ctx context.Context) ([]sysmetricMetric, error) {
	query := e.calQuery()

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error executing query %w", err)
	}

	results := make([]sysmetricMetric, 0)

	for rows.Next() {
		dest := sysmetricMetric{}
		if err = rows.Scan(&dest.beginTime, &dest.endTime, &dest.intsizeCsec, &dest.groupId, &dest.metricId, &dest.name, &dest.value, &dest.metricUnit, &dest.conId); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	return results, nil
}

func (m *MetricSet) addSysmetricData(bs []sysmetricMetric) map[string]common.MapStr {
	out := make(map[string]common.MapStr)

	for _, sysmetricMetric := range bs {
		key := strconv.Itoa(int(sysmetricMetric.metricId.Int64)) + strconv.Itoa(int(sysmetricMetric.groupId.Int64))

		out[key] = common.MapStr{}

		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.begin_time", &oracle.StringValue{NullString: sysmetricMetric.beginTime})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.end_time", &oracle.StringValue{NullString: sysmetricMetric.endTime})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.interval_size_csec", &oracle.Float64Value{NullFloat64: sysmetricMetric.intsizeCsec})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.group_id", &oracle.Int64Value{NullInt64: sysmetricMetric.groupId})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.metric_id", &oracle.Int64Value{NullInt64: sysmetricMetric.metricId})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.name", &oracle.StringValue{NullString: sysmetricMetric.name})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.value", &oracle.Float64Value{NullFloat64: sysmetricMetric.value})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.metric_unit", &oracle.StringValue{NullString: sysmetricMetric.metricUnit})
		oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "metrics.container_id", &oracle.Float64Value{NullFloat64: sysmetricMetric.conId})
	}
	return out
}
