// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tablespace

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var expectedResults = []string{`{"data_file":{"id":18,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux01.dbf","online_status":"ONLINE","size":{"bytes":9999990,"free":{"bytes":99999994},"max":{"bytes":9999994}},"status":"AVAILABLE"},"name":"SYSAUX","space":{"free":{"bytes":9999},"used":{"bytes":9991}}}`,
	`{"data_file":{"id":181,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux02.dbf","online_status":"ONLINE","size":{"bytes":9999991,"free":{"bytes":99999995},"max":{"bytes":9999995}},"status":"AVAILABLE"},"name":"SYSAUX","space":{"free":{"bytes":9999},"used":{"bytes":9991}}}`,
	`{"data_file":{"id":182,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/sysaux03.dbf","online_status":"ONLINE","size":{"bytes":9999992,"free":{"bytes":99999996},"max":{"bytes":9999996}},"status":"AVAILABLE"},"name":"SYSAUX","space":{"free":{"bytes":9999},"used":{"bytes":9991}}}`,
	`{"data_file":{"id":18,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/system01.dbf","online_status":"ONLINE","size":{"bytes":999990,"free":{"bytes":9999994},"max":{"bytes":9999994}},"status":"AVAILABLE"},"name":"SYSTEM","space":{"free":{"bytes":9990},"used":{"bytes":9991}}}`,
	`{"data_file":{"id":18,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/temp012017-03-02_07-54-38-075-AM.dbf","online_status":"ONLINE","size":{"bytes":999991,"free":{"bytes":9999994},"max":{"bytes":9999994}},"status":"AVAILABLE"},"name":"TEMP","space":{"free":{"bytes":99999},"total":{"bytes":99999},"used":{"bytes":99999}}}`,
	`{"data_file":{"id":18,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/undotbs01.dbf","online_status":"ONLINE","size":{"bytes":999992,"free":{"bytes":9999994},"max":{"bytes":9999994}},"status":"AVAILABLE"},"name":"UNDOTBS1","space":{"free":{"bytes":9999},"used":{"bytes":9991}}}`,
	`{"data_file":{"id":18,"name":"/u02/app/oracle/oradata/ORCLCDB/orclpdb1/users01.dbf","online_status":"ONLINE","size":{"bytes":999993,"free":{"bytes":9999994},"max":{"bytes":9999994}},"status":"AVAILABLE"},"name":"USERS","space":{"free":{"bytes":9999},"used":{"bytes":9991}}}`}

var notExpectedEvents = []string{`{}`, `{"foo":"bar"}`}

func TestEventMapping(t *testing.T) {
	m := MetricSet{extractor: &happyMockExtractor{}}

	events, err := m.extractAndTransform(context.Background())
	assert.NoError(t, err)

	t.Logf("Total %d events\n", len(events))

	t.Run("Happy Path", func(t *testing.T) {
		for _, event := range events {
			var found bool

			for _, expected := range expectedResults {
				if expected == event.MetricSetFields.String() {
					found = true
				}
			}

			assert.Truef(t, found, "event not found into the expected events:\nEvent:%s  \nExpected events: %v", event, expectedResults)
		}
	})

	t.Run("Check that the events checker doesn't become flaky by mistake", func(t *testing.T) {
		for _, event := range events {
			var found = false

			for _, notExpected := range notExpectedEvents {
				if notExpected == event.MetricSetFields.String() {
					found = true
				}
			}

			assert.Falsef(t, found, "event should not be found into the not expected events\nEvent: %s\nNot expected events: %v", event, notExpectedEvents)
		}
	})

	t.Run("Error Paths", func(t *testing.T) {
		t.Run("data files", func(t *testing.T) {
			m := MetricSet{extractor: &errorDataFilesMockExtractor{}}

			_, err := m.extractAndTransform(context.Background())
			assert.Error(t, err)
		})

		t.Run("free space data", func(t *testing.T) {
			m := MetricSet{extractor: &errorFreeSpaceDataMockExtractor{}}

			_, err := m.extractAndTransform(context.Background())
			assert.Error(t, err)
		})

		t.Run("temp free space data", func(t *testing.T) {
			m := MetricSet{extractor: &errorTempFreeSpaceDataMockExtractor{}}

			_, err := m.extractAndTransform(context.Background())
			assert.Error(t, err)
		})
	})
}

func TestPeriod(t *testing.T) {
	t.Run("Check lower period", func(t *testing.T) {
		var printWarning = CheckCollectionPeriod(time.Second * 59)
		assert.True(t, printWarning, "Warning expected.")
	})

	t.Run("Check period", func(t *testing.T) {
		var printWarning = CheckCollectionPeriod(time.Minute * 10)
		assert.False(t, printWarning, "Warning not expected.")
	})

	t.Run("Check higher period", func(t *testing.T) {
		var printWarning = CheckCollectionPeriod(time.Minute * 11)
		assert.False(t, printWarning, "Warning not expected.")
	})
}
