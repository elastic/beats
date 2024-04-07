// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration

package billing

import (
	"errors"
	"fmt"
	costexplorertypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetStartDateEndDate(t *testing.T) {
	startDate, endDate := getStartDateEndDate(time.Duration(24) * time.Hour)
	assert.NotEmpty(t, startDate)
	assert.NotEmpty(t, endDate)
}

func TestValidateGroupByType(t *testing.T) {
	cases := []struct {
		groupByType     costexplorertypes.GroupDefinitionType
		expectedSupport bool
		expectedErr     error
	}{
		{
			costexplorertypes.GroupDefinitionTypeDimension,
			true,
			nil,
		},
		{
			costexplorertypes.GroupDefinitionTypeTag,
			true,
			nil,
		},
		{
			costexplorertypes.GroupDefinitionTypeCostCategory,
			false,
			errors.New(fmt.Sprintf("costexplorer GetCostAndUsageRequest or metricbeat module does not support group by type: %s", costexplorertypes.GroupDefinitionTypeCostCategory)),
		},
		{
			"INVALID_TYPE",
			false,
			errors.New("costexplorer GetCostAndUsageRequest or metricbeat module does not support group by type: INVALID_TYPE"),
		},
	}

	for _, c := range cases {
		t.Run(string(c.groupByType), func(t *testing.T) {
			supported, err := validateGroupByType(c.groupByType)
			assert.Equal(t, c.expectedSupport, supported)
			assert.Equal(t, c.expectedErr, err)
		})
	}
}

func TestValidateDimensionKey(t *testing.T) {
	cases := []struct {
		dimensionKey    string
		expectedSupport bool
		expectedErr     error
	}{
		{
			"INSTANCE_TYPE",
			true,
			nil,
		},
		{
			"INVALID_DIMENSION",
			false,
			errors.New("costexplorer GetCostAndUsageRequest does not support dimension key: INVALID_DIMENSION"),
		},
	}

	for _, c := range cases {
		t.Run(c.dimensionKey, func(t *testing.T) {
			supported, err := validateDimensionKey(c.dimensionKey)
			assert.Equal(t, c.expectedSupport, supported)
			assert.Equal(t, c.expectedErr, err)
		})
	}
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

func TestGetGroupBys(t *testing.T) {
	cases := []struct {
		title            string
		groupByTags      []string
		groupByDimKeys   []string
		expectedGroupBys []groupBy
	}{
		{
			"test with both tags and dimKeys",
			[]string{"createdBy"},
			[]string{"AZ", "INSTANCE_TYPE"},
			[]groupBy{
				{"createdBy", "AZ"},
				{"createdBy", "INSTANCE_TYPE"},
			},
		},
		{
			"test with only dimKeys",
			[]string{},
			[]string{"AZ", "INSTANCE_TYPE"},
			[]groupBy{
				{"", "AZ"},
				{"", "INSTANCE_TYPE"},
			},
		},
		{
			"test with only tags",
			[]string{"createdBy"},
			[]string{},
			[]groupBy{
				{"createdBy", ""},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			groupBys := getGroupBys(c.groupByTags, c.groupByDimKeys)
			assert.Equal(t, c.expectedGroupBys, groupBys)
		})
	}
}
