// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package rds

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

var (
	metricName    = "Queries"
	namespace     = "AWS/RDS"
	periodInSec   = 60
	index         = 0
	dimName1      = "DatabaseClass"
	dimValue1     = "db.r5.large"
	dimName2      = "Role"
	dimValue2     = "READER"
	dbInstanceArn = "arn:aws:rds:us-east-2:627959692251:db:test1"
)

func TestCreateMetricDataQuery(t *testing.T) {
	metric := cloudwatch.Metric{
		MetricName: &metricName,
		Namespace:  &namespace,
	}

	metricDataQuery := createMetricDataQuery(metric, index, dbInstanceArn, periodInSec)
	assert.Equal(t, "arn:aws:rds:us-east-2:627959692251:db:test1 Queries", *metricDataQuery.Label)
	assert.Equal(t, "Average", *metricDataQuery.MetricStat.Stat)
	assert.Equal(t, metricName, *metricDataQuery.MetricStat.Metric.MetricName)
	assert.Equal(t, namespace, *metricDataQuery.MetricStat.Metric.Namespace)
	assert.Equal(t, int64(periodInSec), *metricDataQuery.MetricStat.Period)
}

func TestConstructLabel(t *testing.T) {
	cases := []struct {
		dimensions    []cloudwatch.Dimension
		expectedLabel string
	}{
		{
			[]cloudwatch.Dimension{},
			"arn:aws:rds:us-east-2:627959692251:db:test1 Queries",
		},
		{
			[]cloudwatch.Dimension{
				{
					Name:  &dimName1,
					Value: &dimValue1,
				},
			},
			"arn:aws:rds:us-east-2:627959692251:db:test1 Queries DatabaseClass db.r5.large",
		},
		{
			[]cloudwatch.Dimension{
				{
					Name:  &dimName1,
					Value: &dimValue1,
				},
				{
					Name:  &dimName2,
					Value: &dimValue2,
				},
			},
			"arn:aws:rds:us-east-2:627959692251:db:test1 Queries DatabaseClass db.r5.large Role READER",
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.expectedLabel, constructLabel(c.dimensions, dbInstanceArn, metricName))
	}
}
