// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package rds

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/metricbeat/mb"
	awscommon "github.com/elastic/beats/x-pack/libbeat/common/aws"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "rds"
	metricNameIdx = 0
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(aws.ModuleName, metricsetName, New)
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
	tags               []aws.Tag
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 60s
	remainder := int(metricSet.Period.Seconds()) % 60
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
	startTime, endTime := aws.GetStartTimeEndTime(m.Period)

	for _, regionName := range m.MetricSet.RegionsList {
		awsConfig := m.MetricSet.AwsConfig.Copy()
		awsConfig.Region = regionName

		svc := rds.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "rds", regionName, awsConfig))

		// Get DBInstance IDs per region
		dbInstanceIDs, dbDetailsMap, err := m.getDBInstancesPerRegion(svc)
		if err != nil {
			err = errors.Wrap(err, "getDBInstancesPerRegion failed, skipping region "+regionName)
			m.Logger().Errorf(err.Error())
			report.Error(err)
			continue
		}

		if len(dbInstanceIDs) == 0 {
			continue
		}

		svcCloudwatch := cloudwatch.New(awscommon.EnrichAWSConfigWithEndpoint(
			m.Endpoint, "monitoring", regionName, awsConfig))

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

		// Get MetricDataQuery for all dbInstances per region
		metricDataQueriesTotal := constructMetricQueries(listMetricsOutput, m.Period)
		var metricDataOutput []cloudwatch.MetricDataResult
		if len(metricDataQueriesTotal) != 0 {
			// Use metricDataQueries to make GetMetricData API calls
			metricDataOutput, err = aws.GetMetricDataResults(metricDataQueriesTotal, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "GetMetricDataResults failed, skipping region "+regionName)
				m.Logger().Error(err.Error())
				report.Error(err)
				continue
			}
		}

		// Create Cloudwatch Events for RDS
		events, err := m.createCloudWatchEvents(metricDataOutput, regionName, dbDetailsMap)
		if err != nil {
			m.Logger().Error(err.Error())
			report.Error(err)
			continue
		}

		for _, event := range events {
			if len(event.MetricSetFields) != 0 {
				if reported := report.Event(event); !reported {
					m.Logger().Debug("Fetch interrupted, failed to emit event")
					return nil
				}
			}
		}
	}

	return nil
}

func (m *MetricSet) getDBInstancesPerRegion(svc rdsiface.ClientAPI) ([]string, map[string]DBDetails, error) {
	describeInstanceInput := &rds.DescribeDBInstancesInput{}
	req := svc.DescribeDBInstancesRequest(describeInstanceInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error DescribeDBInstancesRequest")
	}

	var dbInstanceIDs []string
	dbDetailsMap := map[string]DBDetails{}
	for _, dbInstance := range output.DBInstances {
		dbDetails := DBDetails{}
		if dbInstance.DBInstanceIdentifier != nil {
			dbDetails.dbIdentifier = *dbInstance.DBInstanceIdentifier
			dbInstanceIDs = append(dbInstanceIDs, *dbInstance.DBInstanceIdentifier)
		}

		if dbInstance.DBInstanceArn != nil {
			dbDetails.dbArn = *dbInstance.DBInstanceArn
		}

		if dbInstance.DBInstanceClass != nil {
			dbDetails.dbClass = *dbInstance.DBInstanceClass
		}

		if dbInstance.DBInstanceClass != nil {
			dbDetails.dbStatus = *dbInstance.DBInstanceStatus
		}

		if dbInstance.AvailabilityZone != nil {
			dbDetails.dbAvailabilityZone = *dbInstance.AvailabilityZone
		}

		// Get tags for each RDS instance
		listTagsInput := rds.ListTagsForResourceInput{
			ResourceName: dbInstance.DBInstanceArn,
		}
		reqListTags := svc.ListTagsForResourceRequest(&listTagsInput)
		outputListTags, err := reqListTags.Send(context.TODO())
		if err != nil {
			m.Logger().Warn("ListTagsForResourceRequest failed, rds:ListTagsForResource permission is required for getting tags.")
			dbDetailsMap[*dbInstance.DBInstanceIdentifier] = dbDetails
			return dbInstanceIDs, dbDetailsMap, nil
		}

		if m.TagsFilter != nil {
			// Check with each tag filter
			// If tag filter doesn't exist in tagKeys/tagValues,
			// then remove this dbInstance entry from dbDetailsMap.
			if exists := aws.CheckTagFiltersExist(m.TagsFilter, outputListTags.TagList); !exists {
				delete(dbDetailsMap, *dbInstance.DBInstanceIdentifier)
				continue
			}
		}

		for _, tag := range outputListTags.TagList {
			// By default, replace dot "." using under bar "_" for tag keys and values
			dbDetails.tags = append(dbDetails.tags,
				aws.Tag{
					Key:   common.DeDot(*tag.Key),
					Value: common.DeDot(*tag.Value),
				})
		}
		dbDetailsMap[*dbInstance.DBInstanceIdentifier] = dbDetails
	}
	return dbInstanceIDs, dbDetailsMap, nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric, period time.Duration) []cloudwatch.MetricDataQuery {
	var metricDataQueries []cloudwatch.MetricDataQuery
	metricDataQueryEmpty := cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i, period)
		if metricDataQuery == metricDataQueryEmpty {
			continue
		}
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func createMetricDataQuery(metric cloudwatch.Metric, index int, period time.Duration) cloudwatch.MetricDataQuery {
	statistic := "Average"
	periodInSeconds := int64(period.Seconds())
	id := metricsetName + strconv.Itoa(index)
	metricDims := metric.Dimensions

	metricDataQuery := cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &periodInSeconds,
			Stat:   &statistic,
			Metric: &metric,
		},
	}

	label := constructLabel(metricDims, *metric.MetricName)
	metricDataQuery.Label = &label
	return metricDataQuery
}

func constructLabel(metricDimensions []cloudwatch.Dimension, metricName string) string {
	// label = metricName + dimensionKey1 + dimensionValue1
	// + dimensionKey2 + dimensionValue2 + ...
	label := metricName
	if len(metricDimensions) != 0 {
		for _, dim := range metricDimensions {
			label += " "
			label += *dim.Name + " " + *dim.Value
		}
	}
	return label
}

func (m *MetricSet) createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, regionName string, dbInstanceMap map[string]DBDetails) (map[string]mb.Event, error) {
	// Initialize events and metricSetFieldResults per dbInstance
	events := map[string]mb.Event{}
	metricSetFieldResults := map[string]map[string]interface{}{}

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
				// Collect dimension values from the labels and initialize events and metricSetFieldResults with dimValues
				var dimValues string
				for i := 1; i < len(labels); i += 2 {
					dimValues = dimValues + labels[i+1]
				}

				if _, ok := events[dimValues]; !ok {
					events[dimValues] = aws.InitEvent(regionName, m.AccountName, m.AccountID)
				}

				if _, ok := metricSetFieldResults[dimValues]; !ok {
					metricSetFieldResults[dimValues] = map[string]interface{}{}
				}

				if len(output.Values) > timestampIdx && len(labels) > 0 {
					if labels[metricNameIdx] == "CPUUtilization" {
						metricSetFieldResults[dimValues][labels[metricNameIdx]] = fmt.Sprint(output.Values[timestampIdx] / 100)
					} else {
						metricSetFieldResults[dimValues][labels[metricNameIdx]] = fmt.Sprint(output.Values[timestampIdx])
					}

					for i := 1; i < len(labels); i += 2 {
						if labels[i] == "DBInstanceIdentifier" {
							dbIdentifier := labels[i+1]
							if _, found := events[dbIdentifier]; found {
								if _, found := dbInstanceMap[dbIdentifier]; !found {
									delete(metricSetFieldResults, dimValues)
									continue
								}
								events[dbIdentifier].RootFields.Put("cloud.availability_zone", dbInstanceMap[dbIdentifier].dbAvailabilityZone)
								events[dbIdentifier].MetricSetFields.Put("db_instance.arn", dbInstanceMap[dbIdentifier].dbArn)
								events[dbIdentifier].MetricSetFields.Put("db_instance.class", dbInstanceMap[dbIdentifier].dbClass)
								events[dbIdentifier].MetricSetFields.Put("db_instance.identifier", dbInstanceMap[dbIdentifier].dbIdentifier)
								events[dbIdentifier].MetricSetFields.Put("db_instance.status", dbInstanceMap[dbIdentifier].dbStatus)

								for _, tag := range dbInstanceMap[dbIdentifier].tags {
									events[dbIdentifier].ModuleFields.Put("tags."+tag.Key, tag.Value)
								}
							}
						}
						metricSetFieldResults[dimValues][labels[i]] = fmt.Sprint(labels[(i + 1)])
					}

					// if tags_filter is given, then only return metrics with DBInstanceIdentifier as dimension
					if m.TagsFilter != nil {
						if len(labels) == 1 {
							delete(events, dimValues)
							delete(metricSetFieldResults, dimValues)
						}

						for i := 1; i < len(labels); i += 2 {
							if labels[i] != "DBInstanceIdentifier" && i == len(labels)-2 {
								delete(events, dimValues)
								delete(metricSetFieldResults, dimValues)
							}
						}
					}
				}
			}
		}
	}

	for dimValues, metricSetFieldsPerInstance := range metricSetFieldResults {
		resultMetricsetFields, err := aws.EventMapping(metricSetFieldsPerInstance, schemaMetricSetFields)
		if err != nil {
			return events, errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS RDS metricbeat module")
		}

		events[dimValues].MetricSetFields.Update(resultMetricsetFields)
	}

	return events, nil
}
