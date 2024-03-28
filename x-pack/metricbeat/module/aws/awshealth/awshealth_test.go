// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awshealth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentDateTime(t *testing.T) {
	cdt := getCurrentDateTime()
	assert.NotEmpty(t, cdt)
}

func TestGenerateEventID(t *testing.T) {
	cdt := getCurrentDateTime()
	e_arn := "arn:aws:health:us-east-1::event/LAMBDA/AWS_LAMBDA_OPERATIONAL_NOTIFICATION/AWS_LAMBDA_OPERATIONAL_NOTIFICATION_e76969649ab96dd"
	sc := ""
	eventID := cdt + e_arn + sc
	eid := generateEventID(eventID)
	assert.NotEmpty(t, eid)
}

func TestGetValueOrDefault(t *testing.T) {
	// Test case 1: Test with non-nil string pointer
	inputString := "hello"
	resultString := getValueOrDefault(&inputString, "")
	assert.Equal(t, "hello", resultString, "Result should match input string")

	// Test case 2: Test with nil string pointer
	var nilString *string
	resultString = getValueOrDefault(nilString, "")
	assert.Equal(t, "", resultString, "Result should be an empty string")

	// Test case 3: Test with non-nil time pointer
	now := time.Now()
	resultTime := getValueOrDefault(&now, time.Time{})
	assert.Equal(t, now, resultTime, "Result should match current time")

	// Test case 4: Test with nil time pointer
	var nilTime *time.Time
	resultTime = getValueOrDefault(nilTime, time.Time{})
	assert.Equal(t, time.Time{}, resultTime, "Result should be zero time")
}
