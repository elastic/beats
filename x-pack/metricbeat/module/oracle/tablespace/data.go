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

func (m *MetricSet) extract(extractor tablespaceExtractMethods) (out *extractedData, err error) {
	out = &extractedData{}

	if out.dataFiles, err = extractor.dataFilesData(); err != nil {
		return nil, errors.Wrap(err, "error getting data_files")
	}

	if out.tempFreeSpace, err = extractor.tempFreeSpaceData(); err != nil {
		return nil, errors.Wrap(err, "error getting temp_free_space")
	}

	if out.freeSpace, err = extractor.freeSpaceData(); err != nil {
		return nil, errors.Wrap(err, "error getting free space data")
	}

	return
}

func (m *MetricSet) transform(in *extractedData) (out map[string]common.MapStr) {
	out = make(map[string]common.MapStr, 0)

	for _, dataFile := range in.dataFiles {
		m.addRowResultToCommonMapstr(&dataFile, out)
	}

	m.addFreeSpaceData(in.freeSpace, out)
	m.addTempFreeSpaceData(in.tempFreeSpace, out)

	return
}

func (m *MetricSet) eventMapping() ([]mb.Event, error) {
	extractedData_, err := m.extract(m.extractor)
	if err != nil {
		return nil, errors.Wrap(err, "error extracting data")
	}

	out := m.transform(extractedData_)

	events := make([]mb.Event, 0)
	for _, v := range out {
		events = append(events, mb.Event{MetricSetFields: v})
	}

	return events, nil
}

func (m *MetricSet) addTempFreeSpaceData(tempFreeSpaces []tempFreeSpace, final map[string]common.MapStr) {
	for key, cm := range final {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}
		name := val.(string)
		if name == "TEMP" {
			for _, freeSpaceTable := range tempFreeSpaces {
				m.checkNullInt64(final, key, "free_space.table_size.bytes", freeSpaceTable.TablespaceSize)
				m.checkNullInt64(final, key, "free_space.allocated.bytes", freeSpaceTable.AllocatedSpace)
				m.checkNullInt64(final, key, "free_space.free.bytes", freeSpaceTable.FreeSpace)
			}
		}
	}
}

func (m *MetricSet) addFreeSpaceData(freeSpaces []freeSpace, final map[string]common.MapStr) {
	for i, cm := range final {
		val, err := cm.GetValue("name")
		if err != nil {
			m.Logger().Debug("error getting tablespace name")
			continue
		}
		name := val.(string)
		if name != "" {
			for _, freeSpaceTable := range freeSpaces {
				if name == freeSpaceTable.TablespaceName {
					_, _ = final[i].Put("free_space.bytes", freeSpaceTable.TotalBytes.Int64)
				}
			}
		}
	}
}

func (m *MetricSet) addRowResultToCommonMapstr(res tablespaceNameGetter, output map[string]common.MapStr) {
	if _, found := output[res.hash()]; !found {
		output[res.hash()] = common.MapStr{}
	}

	val, ok := res.(*dataFile)
	if !ok {
		m.Logger().Debug("error trying to type assert a dataFile type")
		return
	}

	output[res.hash()].Put("name", res.eventKey())

	m.checkNullString(output, res.hash(), "data_file.name", val.FileName)
	m.checkNullInt64(output, res.hash(), "data_file.id", val.FileID)
	m.checkNullInt64(output, res.hash(), "data_file.size.bytes", val.TotalSizeBytes)
	m.checkNullInt64(output, res.hash(), "data_file.size.max.bytes", val.MaxFileSizeBytes)
	m.checkNullString(output, res.hash(), "data_file.status", val.Status)
	m.checkNullString(output, res.hash(), "data_file.online_status", val.OnlineStatus)
	m.checkNullInt64(output, res.hash(), "data_file.user.bytes", val.AvailableForUserBytes)

}

func (m *MetricSet) checkNullInt64(output map[string]common.MapStr, parentKey, field string, nullInt64 sql.NullInt64) {
	if nullInt64.Valid {
		if _, ok := output[parentKey]; ok {
			if _, err := output[parentKey].Put(field, nullInt64.Int64); err != nil {
				m.Logger().Debug(err)
			}
		}
	}
}

func (m *MetricSet) checkNullString(output map[string]common.MapStr, parentKey, field string, nullString sql.NullString) {
	if nullString.Valid {
		if _, ok := output[parentKey]; ok {
			if _, err := output[parentKey].Put(field, nullString.String); err != nil {
				m.Logger().Debug(err)
			}
		}
	}
}
