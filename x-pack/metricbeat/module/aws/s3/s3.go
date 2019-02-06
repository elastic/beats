// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package s3

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/cloudwatchiface"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/metricbeat/module/aws"
)

var metricsetName = "s3"

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
	logger *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The aws s3 metricset is beta.")
	s3Logger := logp.NewLogger(aws.ModuleName)

	moduleConfig := aws.Config{}
	if err := base.Module().UnpackConfig(&moduleConfig); err != nil {
		return nil, err
	}

	if moduleConfig.Period == "" {
		err := errors.New("period is not set in AWS module config")
		s3Logger.Error(err)
	}

	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	// Check if period is set to be multiple of 86400s
	remainder := metricSet.PeriodInSec % 86400
	if remainder != 0 {
		err := errors.New("period needs to be set to 86400s (or a multiple of 86400s). " +
			"To avoid data missing or extra costs, please make sure period is set correctly " +
			"in config.yml")
		s3Logger.Info(err)
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    s3Logger,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) {
	namespace := "AWS/S3"
	// Get startTime and endTime
	startTime, endTime, err := getStartTimeEndTime()
	if err != nil {
		logp.Error(errors.Wrap(err, "Error ParseDuration"))
		m.logger.Error(err.Error())
		report.Error(err)
		return
	}

	// GetMetricData for AWS S3 from Cloudwatch
	for _, regionName := range m.MetricSet.RegionsList {
		m.MetricSet.AwsConfig.Region = regionName
		svcCloudwatch := cloudwatch.New(*m.MetricSet.AwsConfig)
		listMetricsInput := &cloudwatch.ListMetricsInput{Namespace: &namespace}
		reqListMetrics := svcCloudwatch.ListMetricsRequest(listMetricsInput)

		// List metrics of S3 for each region
		listMetricsOutput, err := reqListMetrics.Send()
		if err != nil {
			err = errors.Wrap(err, "ListMetricsRequest failed, skipping region "+regionName)
			m.logger.Error(err.Error())
			report.Error(err)
			continue
		}

		if listMetricsOutput.Metrics == nil || len(listMetricsOutput.Metrics) == 0 {
			// No S3 buckets in this region
			continue
		}

		metricDataQueries := constructMetricQueries(listMetricsOutput.Metrics)

		init := true
		getMetricDataOutput := &cloudwatch.GetMetricDataOutput{NextToken: nil}
		for init || getMetricDataOutput.NextToken != nil {
			init = false
			output, err := getMetricDataPerRegion(metricDataQueries, getMetricDataOutput.NextToken, svcCloudwatch, startTime, endTime)
			if err != nil {
				err = errors.Wrap(err, "getMetricDataPerRegion failed, skipping region "+regionName)
				m.logger.Error(err.Error())
				report.Error(err)
				continue
			}
			getMetricDataOutput.MetricDataResults = append(getMetricDataOutput.MetricDataResults, output.MetricDataResults...)
		}

		// Create Cloudwatch Events for S3
		event, info, err := createCloudWatchEvents(getMetricDataOutput.MetricDataResults, regionName)
		if info != "" {
			m.logger.Info(info)
		}

		if err != nil {
			m.logger.Error(err.Error())
			event.Error = err
			report.Event(event)
			continue
		}
		report.Event(event)
	}
}

func getStartTimeEndTime() (startTime time.Time, endTime time.Time, err error) {
	endTime = time.Now()
	duration, err := time.ParseDuration("-24h")
	if err != nil {
		return
	}
	startTime = endTime.Add(duration)
	return startTime, endTime, nil
}

func constructMetricQueries(listMetricsOutput []cloudwatch.Metric) []cloudwatch.MetricDataQuery {
	metricDataQueries := []cloudwatch.MetricDataQuery{}
	for i, listMetric := range listMetricsOutput {
		metricDataQuery := createMetricDataQuery(listMetric, i)
		metricDataQueries = append(metricDataQueries, metricDataQuery)
	}
	return metricDataQueries
}

func getMetricDataPerRegion(metricDataQueries []cloudwatch.MetricDataQuery, nextToken *string, svc cloudwatchiface.CloudWatchAPI, startTime time.Time, endTime time.Time) (*cloudwatch.GetMetricDataOutput, error) {
	getMetricDataInput := &cloudwatch.GetMetricDataInput{
		NextToken:         nextToken,
		StartTime:         &startTime,
		EndTime:           &endTime,
		MetricDataQueries: metricDataQueries,
	}

	reqGetMetricData := svc.GetMetricDataRequest(getMetricDataInput)
	getMetricDataOutput, err := reqGetMetricData.Send()
	if err != nil {
		logp.Error(errors.Wrap(err, "Error GetMetricDataInput"))
		return nil, err
	}
	return getMetricDataOutput, nil
}

func createMetricDataQuery(metric cloudwatch.Metric, index int) (metricDataQuery cloudwatch.MetricDataQuery) {
	statistic := "Average"
	// period has to be 1day
	period := int64(86400)
	id := "d" + strconv.Itoa(index)
	metricDims := metric.Dimensions
	bucketName := ""
	storageType := ""
	for _, dim := range metricDims {
		if *dim.Name == "BucketName" {
			bucketName = *dim.Value
		} else if *dim.Name == "StorageType" {
			storageType = *dim.Value
		}
	}
	metricName := *metric.MetricName
	label := bucketName + " " + storageType + " " + metricName

	metricDataQuery = cloudwatch.MetricDataQuery{
		Id: &id,
		MetricStat: &cloudwatch.MetricStat{
			Period: &period,
			Stat:   &statistic,
			Metric: &metric,
		},
		Label: &label,
	}
	return
}

func createCloudWatchEvents(getMetricDataResults []cloudwatch.MetricDataResult, regionName string) (event mb.Event, info string, err error) {
	event.Service = metricsetName
	event.RootFields = common.MapStr{}
	mapOfRootFieldsResults := make(map[string]interface{})
	mapOfRootFieldsResults["service.name"] = metricsetName
	mapOfRootFieldsResults["cloud.region"] = regionName

	resultRootFields, err := eventMapping(mapOfRootFieldsResults, schemaRootFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaRootFields in AWS S3 metricbeat module.")
		return
	}
	event.RootFields = resultRootFields

	mapOfMetricSetFieldResults := make(map[string]interface{})
	for _, output := range getMetricDataResults {
		if len(output.Values) == 0 {
			continue
		}
		labels := strings.Split(*output.Label, " ")
		mapOfMetricSetFieldResults["bucket.name"] = labels[0]
		mapOfMetricSetFieldResults["bucket.storage.type"] = labels[1]
		mapOfMetricSetFieldResults[labels[2]] = fmt.Sprint(output.Values[0])
	}

	resultMetricSetFields, err := eventMapping(mapOfMetricSetFieldResults, schemaMetricSetFields)
	if err != nil {
		err = errors.Wrap(err, "Error trying to apply schema schemaMetricSetFields in AWS S3 metricbeat module.")
		return
	}
	event.MetricSetFields = resultMetricSetFields
	return
}
