// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package cost

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetStartDateEndDate(t *testing.T) {
	startDate, endDate := getStartDateEndDate(time.Duration(24) * time.Hour)
	assert.NotEmpty(t, startDate)
	assert.NotEmpty(t, endDate)
}
