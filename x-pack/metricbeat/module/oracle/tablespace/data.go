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

type tablespaceNameGetter interface {
	eventKey() string
	hash() string
}

// extract is the E of a ETL processing. Gets the data files, used/free space and temp free space data that is fetch
// by doing queries to Oracle
func (m *MetricSet) extract(extractor tablespaceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	if out.dataFiles, err = extractor.dataFilesData(); err != nil {
		return nil, errors.Wrap(err, "error getting data_files")
	}

	if out.tempFreeSpace, err = extractor.tempFreeSpaceData(); err != nil {
		return nil, errors.Wrap(err, "error getting temp_free_space")
	}

	if out.freeSpace, err = extractor.usedAndFreeSpaceData(); err != nil {
		return nil, errors.Wrap(err, "error getting free space data")
	}

	return
}

// transform is the T of an ETL (refer to the 'extract' method above if you need to see the origin). Transforms the data
// to create a Kibana/Elasticsearch friendly JSON. Data from Oracle is pretty fragmented by design so a lot of data
// was necessary. Data is organized by Tablespace entity (Tablespaces might contain one or more data files)
func (m *MetricSet) transform(in *extractedData) (out map[string]common.MapStr) {
	out = make(map[string]common.MapStr, 0)

	for _, dataFile := range in.dataFiles {
		m.addDataFileData(&dataFile, out)
	}

	m.addUsedAndFreeSpaceData(in.freeSpace, out)
	m.addTempFreeSpaceData(in.tempFreeSpace, out)

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

// addTempFreeSpaceData is specific to the TEMP Tablespace.
func (m *MetricSet) addTempFreeSpaceData(tempFreeSpaces []tempFreeSpace, out map[string]common.MapStr) {
	for key, cm := range out {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}
		name := val.(string)
		if name == "TEMP" {
			for _, tempFreeSpaceTable := range tempFreeSpaces {
				m.checkNullInt64(out, key, "space.total.bytes", tempFreeSpaceTable.TablespaceSize)
				m.checkNullInt64(out, key, "space.used.bytes", tempFreeSpaceTable.UsedSpaceBytes)
				m.checkNullInt64(out, key, "space.free.bytes", tempFreeSpaceTable.FreeSpace)
			}
		}
	}
}

// addUsedAndFreeSpaceData is specific to all Tablespaces but TEMP
func (m *MetricSet) addUsedAndFreeSpaceData(freeSpaces []usedAndFreeSpace, out map[string]common.MapStr) {
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
					m.checkNullInt64(out, key, "space.free.bytes", freeSpaceTable.TotalFreeBytes)
					m.checkNullInt64(out, key, "space.used.bytes", freeSpaceTable.TotalUsedBytes)
				}
			}
		}
	}
}

// addDataFileData is a specific data file which generates a JSON output.
func (m *MetricSet) addDataFileData(res tablespaceNameGetter, output map[string]common.MapStr) {
	if _, found := output[res.hash()]; !found {
		output[res.hash()] = common.MapStr{}
	}

	val, ok := res.(*dataFile)
	if !ok {
		m.Logger().Debug("error trying to type assert a dataFile type")
		return
	}

	_, _ = output[res.hash()].Put("name", res.eventKey())

	m.checkNullString(output, res.hash(), "data_file.name", val.FileName)
	m.checkNullInt64(output, res.hash(), "data_file.id", val.FileID)
	m.checkNullInt64(output, res.hash(), "data_file.size.bytes", val.FileSizeBytes)
	m.checkNullInt64(output, res.hash(), "data_file.size.max.bytes", val.MaxFileSizeBytes)
	m.checkNullInt64(output, res.hash(), "data_file.size.free.bytes", val.AvailableForUserBytes)
	m.checkNullString(output, res.hash(), "data_file.status", val.Status)
	m.checkNullString(output, res.hash(), "data_file.online_status", val.OnlineStatus)

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
