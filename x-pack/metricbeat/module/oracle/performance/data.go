// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
)

type extractedData struct {
}

// extract is the E of a ETL processing. TODO
func (m *MetricSet) extract(extractor performanceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	//TODO

	return
}

func getBufferCacheHitRatio() {
	rows, err := e.db.Query("SELECT FILE_NAME, FILE_ID, TABLESPACE_NAME, BYTES, STATUS, MAXBYTES, USER_BYTES, ONLINE_STATUS FROM SYS.DBA_DATA_FILES UNION SELECT FILE_NAME, FILE_ID, TABLESPACE_NAME, BYTES, STATUS, MAXBYTES, USER_BYTES, STATUS AS ONLINE_STATUS FROM SYS.DBA_TEMP_FILES")
	if err != nil {
		return nil, errors.Wrap(err, "error executing query")
	}

	results := make([]bufferCacheHitRatio, 0)

	for rows.Next() {
		dest := bufferCacheHitRatio{}
		if err = rows.Scan(&dest.FileName, &dest.FileID, &dest.TablespaceName, &dest.FileSizeBytes, &dest.Status, &dest.MaxFileSizeBytes, &dest.AvailableForUserBytes, &dest.OnlineStatus); err != nil {
			return nil, err
		}
		results = append(results, dest)
	}

	return results, nil
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin of the data). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. TODO
func (m *MetricSet) transform(in *extractedData) (out map[string]common.MapStr) {
	out = make(map[string]common.MapStr, 0)

	return
}

func (m *MetricSet) eventMapping() ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(m.extractor)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting data")
	}

	out := m.transform(extractedMetricsData)

	events := make([]mb.Event, 0)
	for _, v := range out {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	return events, nil
}

// checkNullInt64 avoid setting an invalid 0 long value on Metricbeat event
func (m *MetricSet) checkNullInt64(output map[string]common.MapStr, parentKey, field string, nullInt64 sql.NullInt64) {
	if nullInt64.Valid {
		if _, ok := output[parentKey]; ok {
			if _, err := output[parentKey].Put(field, nullInt64.Int64); err != nil {
				m.Logger().Debug(err)
			}
		}
	}
}

// checkNullString avoid setting an invalid empty string value on Metricbeat event
func (m *MetricSet) checkNullString(output map[string]common.MapStr, parentKey, field string, nullString sql.NullString) {
	if nullString.Valid {
		if _, ok := output[parentKey]; ok {
			if _, err := output[parentKey].Put(field, nullString.String); err != nil {
				m.Logger().Debug(err)
			}
		}
	}
}
