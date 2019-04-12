// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/rds/rdsiface"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "rds"

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New,
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	*aws.MetricSet
}

// DBDetails holds detailed information from DescribeDBInstances for each rds.
type DBDetails struct {
	dbArn              string
	dbClass            string
	dbAvailabilityZone string
	dbIdentifier       string
	dbStatus           string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 60s
	remainder := metricSet.PeriodInSec % 60
	if remainder != 0 {
		err := errors.New("Period needs to be set to 60s (or a multiple of 60s). To avoid data missing or " +
			"extra costs, please make sure period is set correctly in config.yml")
		base.Logger().Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startTime and endTime
	startTime, endTime, err := aws.GetStartTimeEndTime(m.DurationString)
	if err != nil {
		return errors.Wrap(err, "Error ParseDuration")
	}

	for _, regionName := range m.MetricSet.RegionsList {
		m.MetricSet.AwsConfig.Region = regionName
		svc := rds.New(*m.MetricSet.AwsConfig)
		// Get DBInstance ARNs per region
		dbInstanceARNs, dbDetailsMap, err := getDBInstancesPerRegion(svc)
		if err != nil {
			err = errors.Wrap(err, "getDBInstancesPerRegion failed, skipping region "+regionName)
			m.Logger().Errorf(err.Error())
			report.Error(err)
			continue
		}

		if len(dbInstanceARNs) == 0 {
			continue
		}

		svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)
		namespace := "AWS/RDS"
		listMetricsOutput, err := aws.GetListMetricsOutput(namespace, regionName, svcCloudwatch)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutput == nil || len(listMetricsOutput) == 0 {
			continue
		}

		for _, dbInstanceARN := range dbInstanceARNs {
			metricDataQueries := constructMetricQueries(listMetricsOutput, dbInstanceARN, m.PeriodInSec)
			// If metricDataQueries, still needs to createCloudWatchEvents.
			metricDataOutput := []cloudwatch.MetricDataResult{}
			if len(metricDataQueries) != 0 {
				// Use metricDataQueries to make GetMetricData API calls
				metricDataOutput, err = aws.GetMetricDataResults(metricDataQueries, svcCloudwatch, startTime, endTime)
				if err != nil {
					err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
					m.Logger().Error(err.Error())
					report.Error(err)
					continue
				}
			}

			// Create Cloudwatch Events for RDS
			event, err := createCloudWatchEvents(metricDataOutput, regionName, dbDetailsMap[dbInstanceARN])
			if err != nil {
				m.Logger().Error(err.Error())
				report.Error(err)
				continue
			}
			if reported := report.Event(event); !reported {
				m.Logger().Debug("Fetch interrupted, failed to emit event")
				return nil
			}
		}
	}

	return nil
}

func getDBInstancesPerRegion(svc rdsiface.RDSAPI) ([]string, map[string]DBDetails, error) {
	describeInstanceInput := &rds.DescribeDBInstancesInput{}
	req := svc.DescribeDBInstancesRequest(describeInstanceInput)
	output, err := req.Send()
	if err != nil {
		err = errors.Wrap(err, "Error DescribeDBInstancesRequest")
		return nil, nil, err
	}

	dbInstanceARNs := []string{}
	dbDetailsMap := map[string]DBDetails{}

	for _, dbInstance := range output.DBInstances {
		dbInstanceARNs = append(dbInstanceARNs, *dbInstance.DBInstanceArn)
		dbDetails := DBDetails{
			dbArn:              *dbInstance.DBInstanceArn,
			dbAvailabilityZone: *dbInstance.AvailabilityZone,
			dbClass:            *dbInstance.DBInstanceClass,
			dbIdentifier:       *dbInstance.DBInstanceIdentifier,
			dbStatus:           *dbInstance.DBInstanceStatus,
		}
		dbDetailsMap[*dbInstance.DBInstanceArn] = dbDetails
	}
	return dbInstanceARNs, dbDetailsMap, nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, dbInstanceArn string, periodInSec int) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, dbInstanceArn, periodInSec)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, dbInstanceARN string, periodInSec int) cloudwatch.MetricDataQuery {
	statistic := "Average"
	period := int64(periodInSec)
	id := "rds" + strconv.Itoa(index)
	metricDims := metric.Dimensions

	metricDataQuery := cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &period,
			Stat:   &statistic,
			Metric: &metric,
		},
	}

	label := constructLabel(metricDims, dbInstanceARN, *metric.MetricName)
	metricDataQuery.Label = &label
	return metricDataQuery
}

func constructLabel(metricDimensions []cloudwatch.Dimension, dbInstanceARN string, metricName string) string {
	label := dbInstanceARN + " " + metricName
	if len(metricDimensions) != 0 {
		for _, dim := range metricDimensions {
			label += " "
			label += *dim.Name + " " + *dim.Value
		}
	}
	return label
}

func createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, regionName string, dbInstanceMap DBDetails) (mb.Event, error) {
	event := mb.Event{}
	event.Service = metricsetName
	event.RootFields = common.MapStr{}

	event.RootFields.Put("service.name", metricsetName)
	event.RootFields.Put("cloud.provider", "aws")
	event.RootFields.Put("cloud.region", regionName)

	// AWS RDS Metrics
	mapOfMetricSetFieldResults := make(map[string]interface{})

	// Find a timestamp for all metrics in output
	timestamp := aws.FindTimestamp(getMetricDataResults)
	if !timestamp.IsZero() {
		for _, output := range getMetricDataResults {
			if len(output.Values) == 0 {
				continue
			}
			exists, timestampIdx := aws.CheckTimestampInArray(timestamp, output.Timestamps)
			if exists {
				labels := strings.Split(*output.Label, " ")
				if len(output.Values) > timestampIdx && len(labels) > 1 {
					mapOfMetricSetFieldResults[labels[1]] = fmt.Sprint(output.Values[timestampIdx])
					for i := 1; i <= (len(labels)-2)/2; i++ {
						mapOfMetricSetFieldResults[labels[i*2]] = labels[(i*2 + 1)]
					}
				}
			}
		}
	}

	resultMetricSetFields, err := aws.EventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS RDS metricbeat module.")
		return event, err
	}

	event.MetricSetFields = resultMetricSetFields
	event.RootFields.Put("cloud.availability_zone", dbInstanceMap.dbAvailabilityZone)
	event.MetricSetFields.Put("db_instance.arn", dbInstanceMap.dbArn)
	event.MetricSetFields.Put("db_instance.class", dbInstanceMap.dbClass)
	event.MetricSetFields.Put("db_instance.identifier", dbInstanceMap.dbIdentifier)
	event.MetricSetFields.Put("db_instance.status", dbInstanceMap.dbStatus)
	return event, err
}
