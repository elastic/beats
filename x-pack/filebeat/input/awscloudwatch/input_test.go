// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awscloudwatch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetStartPosition(t *testing.T) {
	currentTime := time.Date(2020, time.June, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		title             string
		startPosition     string
		prevEndTime       int64
		expectedStartTime int64
		expectedEndTime   int64
	}{
		{
			"startPosition=beginning",
			"beginning",
			int64(0),
			int64(0),
			int64(1590969600000),
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			startTime, endTime := getStartPosition(c.startPosition, currentTime, c.prevEndTime)
			assert.Equal(t, c.expectedStartTime, startTime)
			assert.Equal(t, c.expectedEndTime, endTime)
		})
	}
}
