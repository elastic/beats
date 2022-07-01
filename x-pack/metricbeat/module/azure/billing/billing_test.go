// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPreviousDayFrom(t *testing.T) {
	t.Run("returns the previous day as time interval to collect metrics", func(t *testing.T) {
		referenceTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-09 09:41:00")
		assert.NoError(t, err)
		expectedStartTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-08 00:00:00")
		assert.NoError(t, err)
		expectedEndTime, err := time.Parse("2006-01-02 15:04:05", "2007-01-08 23:59:59")
		assert.NoError(t, err)

		actualStartTime, actualEndTime := previousDayFrom(referenceTime)

		assert.Equal(t, expectedStartTime, actualStartTime)
		assert.Equal(t, expectedEndTime, actualEndTime)
	})
}
