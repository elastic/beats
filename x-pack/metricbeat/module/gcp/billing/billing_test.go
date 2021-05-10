// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentMonth(t *testing.T) {
	currentMonth := getCurrentMonth()
	_, err := strconv.ParseInt(currentMonth, 0, 64)
	assert.NoError(t, err)
}
