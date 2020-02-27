// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !integration

package rds

import (
	"net/http"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsiface"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

// MockRDSClient struct is used for unit tests.
type MockRDSClient struct {
	rdsiface.ClientAPI
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

	metricDataQuery := createMetricDataQuery(metric, index, period)
	assert.Equal(t, "Queries", *metricDataQuery.Label)
	assert.Equal(t, "Average", *metricDataQuery.MetricStat.Stat)
	assert.Equal(t, metricName, *metricDataQuery.MetricStat.Metric.MetricName)
	assert.Equal(t, namespace, *metricDataQuery.MetricStat.Metric.Namespace)
	assert.Equal(t, int64(60), *metricDataQuery.MetricStat.Period)
}

func (m *MockRDSClient) DescribeDBInstancesRequest(input *rds.DescribeDBInstancesInput) rds.DescribeDBInstancesRequest {
	httpReq, _ := http.NewRequest("", "", nil)
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
			HTTPRequest: httpReq,
		},
	}
}

func (m *MockRDSClient) ListTagsForResourceRequest(input *rds.ListTagsForResourceInput) rds.ListTagsForResourceRequest {
	httpReq, _ := http.NewRequest("", "", nil)
	return rds.ListTagsForResourceRequest{
		Request: &awssdk.Request{
			Data: &rds.ListTagsForResourceOutput{
				TagList: []rds.Tag{
					{Key: awssdk.String("dept.name"), Value: awssdk.String("eng.software")},
					{Key: awssdk.String("created-by"), Value: awssdk.String("foo")},
				},
			},
			HTTPRequest: httpReq,
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
			"Queries",
		},
		{
			[]cloudwatch.Dimension{
				{
					Name:  &dimName1,
					Value: &dbInstanceClass,
				},
			},
			"Queries DatabaseClass db.r5.large",
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
			"Queries DatabaseClass db.r5.large Role READER",
		},
	}
	for _, c := range cases {
		assert.Equal(t, c.expectedLabel, constructLabel(c.dimensions, metricName))
	}
}

func TestGetDBInstancesPerRegion(t *testing.T) {
	mockSvc := &MockRDSClient{}
	m := MetricSet{}
	dbInstanceIDs, dbDetailsMap, err := m.getDBInstancesPerRegion(mockSvc)
	if err != nil {
		t.FailNow()
	}

	assert.Equal(t, 1, len(dbInstanceIDs))
	assert.Equal(t, 1, len(dbDetailsMap))

	dbInstanceMap := DBDetails{
		dbArn:              dbInstanceArn,
		dbClass:            dbInstanceClass,
		dbAvailabilityZone: availabilityZone,
		dbIdentifier:       dbInstanceIdentifier,
		dbStatus:           dbInstanceStatus,
		tags: []aws.Tag{
			{Key: "dept_name", Value: "eng_software"},
			{Key: "created-by", Value: "foo"},
		},
	}
	assert.Equal(t, dbInstanceMap, dbDetailsMap[dbInstanceIDs[0]])
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
	metricDataQueries := constructMetricQueries(listMetricsOutput, period)
	assert.Equal(t, 2, len(metricDataQueries))

	assert.Equal(t, "rds0", *metricDataQueries[0].Id)
	assert.Equal(t, "Queries DatabaseClass db.r5.large", *metricDataQueries[0].Label)
	assert.Equal(t, "Queries", *metricDataQueries[0].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/RDS", *metricDataQueries[0].MetricStat.Metric.Namespace)
	assert.Equal(t, []cloudwatch.Dimension{dim1}, metricDataQueries[0].MetricStat.Metric.Dimensions)
	assert.Equal(t, int64(60), *metricDataQueries[0].MetricStat.Period)
	assert.Equal(t, "Average", *metricDataQueries[0].MetricStat.Stat)

	assert.Equal(t, "rds1", *metricDataQueries[1].Id)
	assert.Equal(t, "Queries Role READER", *metricDataQueries[1].Label)
	assert.Equal(t, "Queries", *metricDataQueries[1].MetricStat.Metric.MetricName)
	assert.Equal(t, "AWS/RDS", *metricDataQueries[1].MetricStat.Metric.Namespace)
	assert.Equal(t, []cloudwatch.Dimension{dim2}, metricDataQueries[1].MetricStat.Metric.Dimensions)
	assert.Equal(t, int64(60), *metricDataQueries[1].MetricStat.Period)
	assert.Equal(t, "Average", *metricDataQueries[1].MetricStat.Stat)

}
