// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package billing

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

func TestParseGroupKey(t *testing.T) {
	cases := []struct {
		title            string
		groupKey         string
		expectedTagKey   string
		expectedTagValue string
	}{
		{
			"empty tag value",
			"aws:createdBy$",
			"aws:createdBy",
			"",
		},
		{
			"with a tag value of assumed role",
			"aws:createdBy$AssumedRole:AROAWHL7AXDB:158385",
			"aws:createdBy",
			"AssumedRole:AROAWHL7AXDB:158385",
		},
		{
			"with a tag value of IAM user",
			"aws:createdBy$IAMUser:AIDAWHL7AXDB:foo@test.com",
			"aws:createdBy",
			"IAMUser:AIDAWHL7AXDB:foo@test.com",
		},
		{
			"tag value with $",
			"aws:createdBy$IAMUser:AIDAWH$L7AXDB:foo@test.com",
			"aws:createdBy",
			"IAMUser:AIDAWH$L7AXDB:foo@test.com",
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			tagKey, tagValue := parseGroupKey(c.groupKey)
			assert.Equal(t, c.expectedTagKey, tagKey)
			assert.Equal(t, c.expectedTagValue, tagValue)
		})
	}
}
