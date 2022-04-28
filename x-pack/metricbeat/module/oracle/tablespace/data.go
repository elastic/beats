// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/oracle"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

// extract is the E of a ETL processing. Gets the data files, used/free space and temp free space data that is fetch
// by doing queries to Oracle
func (m *MetricSet) extract(ctx context.Context, extractor tablespaceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	if out.dataFiles, err = extractor.dataFilesData(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting data_files")
	}

	if out.tempFreeSpace, err = extractor.tempFreeSpaceData(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting temp_free_space")
	}

	if out.freeSpace, err = extractor.usedAndFreeSpaceData(ctx); err != nil {
		return nil, errors.Wrap(err, "error getting free space data")
	}

	return
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. Data is organized by Tablespace entity (Tablespaces might contain one or more data files)
func (m *MetricSet) transform(in *extractedData) (out map[string]mapstr.M) {
	out = make(map[string]mapstr.M, 0)

	for _, dataFile := range in.dataFiles {
		m.addDataFileData(&dataFile, out)
	}

	m.addUsedAndFreeSpaceData(in.freeSpace, out)
	m.addTempFreeSpaceData(in.tempFreeSpace, out)

	return
}

func (m *MetricSet) extractAndTransform(ctx context.Context) ([]mb.Event, error) {
	extractedMetricsData, err := m.extract(ctx, m.extractor)
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

// addTempFreeSpaceData is specific to the TEMP Tablespace.
func (m *MetricSet) addTempFreeSpaceData(tempFreeSpaces []tempFreeSpace, out map[string]mapstr.M) {
	for key, cm := range out {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}

		name := val.(string)
		if name == "TEMP" {
			for _, tempFreeSpaceTable := range tempFreeSpaces {
				oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "space.total.bytes", &oracle.Int64Value{NullInt64: tempFreeSpaceTable.TablespaceSize})
				oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "space.used.bytes", &oracle.Int64Value{NullInt64: tempFreeSpaceTable.UsedSpaceBytes})
				oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "space.free.bytes", &oracle.Int64Value{NullInt64: tempFreeSpaceTable.FreeSpace})
			}
		}
	}
}

// addUsedAndFreeSpaceData is specific to all Tablespaces but TEMP
func (m *MetricSet) addUsedAndFreeSpaceData(freeSpaces []usedAndFreeSpace, out map[string]mapstr.M) {
	for key, cm := range out {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}

		name := val.(string)
		if name != "" {
			for _, freeSpaceTable := range freeSpaces {
				if name == freeSpaceTable.TablespaceName {
					oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "space.free.bytes", &oracle.Int64Value{NullInt64: freeSpaceTable.TotalFreeBytes})
					oracle.SetSqlValueWithParentKey(m.Logger(), out, key, "space.used.bytes", &oracle.Int64Value{NullInt64: freeSpaceTable.TotalUsedBytes})
				}
			}
		}
	}
}

// addDataFileData is a specific data file which generates a JSON output.
func (m *MetricSet) addDataFileData(d *dataFile, output map[string]mapstr.M) {
	if _, found := output[d.hash()]; !found {
		output[d.hash()] = mapstr.M{}
	}

	_, _ = output[d.hash()].Put("name", d.eventKey())

	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.name", &oracle.StringValue{NullString: d.FileName})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.name", &oracle.StringValue{NullString: d.FileName})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.status", &oracle.StringValue{NullString: d.Status})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.online_status", &oracle.StringValue{NullString: d.OnlineStatus})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.id", &oracle.Int64Value{NullInt64: d.FileID})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.size.bytes", &oracle.Int64Value{NullInt64: d.FileSizeBytes})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.size.max.bytes", &oracle.Int64Value{NullInt64: d.MaxFileSizeBytes})
	oracle.SetSqlValueWithParentKey(m.Logger(), output, d.hash(), "data_file.size.free.bytes", &oracle.Int64Value{NullInt64: d.AvailableForUserBytes})

}
