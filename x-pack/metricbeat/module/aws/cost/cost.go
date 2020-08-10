// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cost

import (
	"fmt"
	"strconv"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"

	"context"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
)

var (
	metricsetName = "cost"
)

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

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "error creating aws metricset")
	}

	return &MetricSet{
		MetricSet: metricSet,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Get startDate and endDate
	startDate, endDate := getStartDateEndDate(m.Period)

	awsConfig := m.MetricSet.AwsConfig.Copy()
	svc := costexplorer.New(awsConfig)
	timePeriod := costexplorer.DateInterval{
		Start: awssdk.String(startDate),
		End:   awssdk.String(endDate),
	}

	// no permission for "NetAmortizedCost" and "NetUnblendedCost"
	input := costexplorer.GetCostAndUsageInput{
		Granularity: costexplorer.GranularityDaily,
		Metrics: []string{"AmortizedCost", "BlendedCost",
			"NormalizedUsageAmount", "UnblendedCost", "UsageQuantity"},
		TimePeriod: &timePeriod,
	}

	req := svc.GetCostAndUsageRequest(&input)
	output, err := req.Send(context.Background())
	if err != nil {
		err = fmt.Errorf("costexplorer GetCostAndUsageRequest failed: %w", err)
		m.Logger().Errorf(err.Error())
		report.Error(err)
	}

	if len(output.ResultsByTime) > 0 {
		costResults := output.ResultsByTime[0].Total
		event := aws.InitEvent("", m.AccountName, m.AccountID)

		for costName, costValue := range costResults {
			cost := costValue
			costFloat, err := strconv.ParseFloat(*cost.Amount, 64)
			if err != nil {
				err = fmt.Errorf("strconv ParseFloat failed: %w", err)
				m.Logger().Errorf(err.Error())
				continue
			}

			value := common.MapStr{
				"amount": costFloat,
				"unit":   &cost.Unit,
			}
			event.MetricSetFields.Put(costName, value)
		}

		event.MetricSetFields.Put("start_date", startDate)
		event.MetricSetFields.Put("end_date", endDate)

		if reported := report.Event(event); !reported {
			m.Logger().Debug("Fetch interrupted, failed to emit event")
			return nil
		}
	}
	return err
}

func getStartDateEndDate(period time.Duration) (startDate string, endDate string) {
	currentTime := time.Now()
	startTime := currentTime.Add(period * -1)
	startDate = startTime.Format("2006-01-02")
	endDate = currentTime.Format("2006-01-02")
	return
}
