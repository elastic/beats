// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package rds

import (
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsiface"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/stretchr/testify/assert"
)

// MockRDSClient struct is used for unit tests.
type MockRDSClient struct {
	rdsiface.RDSAPI
}

var (
	metricName           = "Queries"
	namespace            = "AWS/RDS"
	period               = 60 * time.Second
	index                = 0
	dimName1             = "DatabaseClass"
	dbInstanceClass      = "db.r5.large"
	dimName2             = "Role"
	dimValue2            = "READER"
	dbInstanceArn        = "arn:aws:rds:us-east-2:627959692251:db:test1"
	availabilityZone     = "us-east-1a"
	dbInstanceIdentifier = "test1"
	dbInstanceStatus     = "available"
)

func TestCreateMetricDataQuery(t *testing.T) {
	metric := cloudwatch.Metric{
		MetricName: &metricName,
		Namespace:  &namespace,
	}

	metricDataQuery := createMetricDataQuery(metric, index, dbInstanceArn, period)
	assert.Equal(t, "arn:aws:rds:us-east-2:627959692251:db:test1 Queries", *metricDataQuery.Label)
	assert.Equal(t, "Average", *metricDataQuery.MetricStat.Stat)
	assert.Equal(t, metricName, *metricDataQuery.MetricStat.Metric.MetricName)
	assert.Equal(t, namespace, *metricDataQuery.MetricStat.Metric.Namespace)
	assert.Equal(t, int64(60), *metricDataQuery.MetricStat.Period)
}

func (m *MockRDSClient) DescribeDBInstancesRequest(input *rds.DescribeDBInstancesInput) rds.DescribeDBInstancesRequest {
	return rds.DescribeDBInstancesRequest{
		Request: &awssdk.Request{
			Data: &rds.DescribeDBInstancesOutput{
				DBInstances: []rds.DBInstance{
					{
						AvailabilityZone:     &availabilityZone,
						DBInstanceArn:        &dbInstanceArn,
						DBInstanceClass:      &dbInstanceClass,
						DBInstanceIdentifier: &dbInstanceIdentifier,
						DBInstanceStatus:     &dbInstanceStatus,
					},
				},
			},
		},
	}
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
					Value: &dbInstanceClass,
				},
			},
			"arn:aws:rds:us-east-2:627959692251:db:test1 Queries DatabaseClass db.r5.large",
		},
		{
			[]cloudwatch.Dimension{
				{
					Name:  &dimName1,
					Value: &dbInstanceClass,
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

func TestGetDBInstancesPerRegion(t *testing.T) {
	mockSvc := &MockRDSClient{}
	dbInstanceARNs, dbDetailsMap, err := getDBInstancesPerRegion(mockSvc)
	if err != nil {
		t.FailNow()
	}

	assert.Equal(t, 1, len(dbInstanceARNs))
	assert.Equal(t, 1, len(dbDetailsMap))
	assert.Equal(t, dbInstanceArn, dbInstanceARNs[0])

	dbInstanceMap := DBDetails{
		dbArn:              dbInstanceArn,
		dbClass:            dbInstanceClass,
		dbAvailabilityZone: availabilityZone,
		dbIdentifier:       dbInstanceIdentifier,
		dbStatus:           dbInstanceStatus,
	}
	assert.Equal(t, dbInstanceMap, dbDetailsMap[dbInstanceARNs[0]])
}

func TestConstructMetricQueries(t *testing.T) {
	dim1 := cloudwatch.Dimension{
		Name:  &dimName1,
		Value: &dbInstanceClass,
	}

	dim2 := cloudwatch.Dimension{
		Name:  &dimName2,
		Value: &dimValue2,
	}

	listMetric1 := cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{dim1},
		MetricName: &metricName,
		Namespace:  &namespace,
	}

	listMetric2 := cloudwatch.Metric{
		Dimensions: []cloudwatch.Dimension{dim2},
		MetricName: &metricName,
		Namespace:  &namespace,
	}

	listMetricsOutput := []cloudwatch.Metric{listMetric1, listMetric2}
	metricDataQueries := constructMetricQueries(listMetricsOutput, dbInstanceArn, period)
	assert.Equal(t, 2, len(metricDataQueries))

	assert.Equal(t, "rds0", *metricDataQueries[0].Id)
	assert.Equal(t, "arn:aws:rds:us-east-2:627959692251:db:test1 Queries DatabaseClass db.r5.large", *metricDataQueries[0].Label)
	assert.Equal(t, "Queries", *metricDataQueries[0].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/RDS", *metricDataQueries[0].MetricStat.Metric.Namespace)
	assert.Equal(t, []cloudwatch.Dimension{dim1}, metricDataQueries[0].MetricStat.Metric.Dimensions)
	assert.Equal(t, int64(60), *metricDataQueries[0].MetricStat.Period)
	assert.Equal(t, "Average", *metricDataQueries[0].MetricStat.Stat)

	assert.Equal(t, "rds1", *metricDataQueries[1].Id)
	assert.Equal(t, "arn:aws:rds:us-east-2:627959692251:db:test1 Queries Role READER", *metricDataQueries[1].Label)
	assert.Equal(t, "Queries", *metricDataQueries[1].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/RDS", *metricDataQueries[1].MetricStat.Metric.Namespace)
	assert.Equal(t, []cloudwatch.Dimension{dim2}, metricDataQueries[1].MetricStat.Metric.Dimensions)
	assert.Equal(t, int64(60), *metricDataQueries[1].MetricStat.Period)
	assert.Equal(t, "Average", *metricDataQueries[1].MetricStat.Stat)

}
