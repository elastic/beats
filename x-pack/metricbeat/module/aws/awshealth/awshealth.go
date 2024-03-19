// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awshealth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/health"
	"github.com/aws/aws-sdk-go-v2/service/health/types"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/aws"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const metricsetName = "awshealth"

var Locale string = "en"

var maxResults int32 = 10

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
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
	Config Config `config:"aws_health_config"`
}

// Config holds the configuration specific for aws health metricset
type Config struct {
	EventARNPattern []string `config:"event_arns_pattern"`
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	logger := logp.NewLogger(metricsetName)
	metricSet, err := aws.NewMetricSet(base)
	if err != nil {
		return nil, fmt.Errorf("error creating aws metricset: %w", err)
	}

	cfgwarn.Beta("The aws awshealth metricset is beta.")

	config := struct {
		Config Config `config:"aws_health_config"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		MetricSet: metricSet,
		logger:    logger,
		Config:    config.Config,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, report mb.ReporterV2) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var config aws.Config
	err := m.Module().UnpackConfig(&config)
	if err != nil {
		return err
	}

	// Get startDate and endDate
	// startDate, endDate := getStartDateEndDate(m.Period)

	awsConfig := m.MetricSet.AwsConfig.Copy()

	// Get startTime and endTime
	startTime, endTime := aws.GetStartTimeEndTime(time.Now(), m.Period, m.Latency)
	m.Logger().Debugf("[AWS Health] startTime = %s, endTime = %s", startTime, endTime)
	health_client := health.NewFromConfig(awsConfig, func(o *health.Options) {
		if config.AWSConfig.FIPSEnabled {
			o.EndpointOptions.UseFIPSEndpoint = awssdk.FIPSEndpointStateEnabled
		}
	})

	events := m.getEventsSummary(ctx, health_client, startTime, endTime)
	for _, event := range events {
		report.Event(event)
	}

	return nil
}

// Make call to DescribeEvents()
// Returns information about events that meet the specified filter criteria. Events are returned in a summary form and do not include the detailed description, any additional metadata that depends on the event type, or any affected resources.
func (m *MetricSet) getEventsSummary(
	ctx context.Context,
	awsHealth *health.Client,
	startTime time.Time,
	endTime time.Time,
) []mb.Event {
	var events []mb.Event
	eventFilter := types.EventFilter{
		// LastUpdatedTimes: []types.DateTimeRange{
		// 	{
		// 		From: &startTime,
		// 		To:   &endTime,
		// 	},
		// },
		//Regions: []string{"ap-south"},
		EventStatusCodes: []types.EventStatusCode{
			types.EventStatusCodeUpcoming,
			types.EventStatusCodeOpen,
			// types.EventStatusCodeClosed,
		},
	}

	var nextTokenString = ""
	var eventOutput *health.DescribeEventsOutput
	var err error

	for {
		if nextTokenString == "" {
			eventOutput, err = awsHealth.DescribeEvents(ctx,
				&health.DescribeEventsInput{
					Filter:     &eventFilter,
					MaxResults: &maxResults,
				},
			)
		} else {
			eventOutput, err = awsHealth.DescribeEvents(ctx,
				&health.DescribeEventsInput{
					Filter:     &eventFilter,
					MaxResults: &maxResults,
					NextToken:  &nextTokenString,
				},
			)
		}
		if err != nil {
			err = fmt.Errorf("AWS Health DescribeEvents failed with %w", err)
			m.Logger().Error(err.Error())
			return nil
		}
		ets := eventOutput.Events
		c := make(chan HealthDetails)
		select {
		case <-ctx.Done():
			// Context cancelled, handle graceful termination
			m.Logger().Info("Context cancelled. Exiting gracefully.")
			close(c)
			return nil
		default:
			// Context not cancelled, proceed with the function
		}

		for _, et := range ets {
			m.Logger().Debugf("[AWS Health] [Fetch DescribeEventDetails] Event ARN : %s", getStringValueOrDefault(et.Arn))
			go m.getDescribeEventDetails(ctx, awsHealth, et, c)
		}

		for i := 0; i < len(ets); i++ {
			select {
			case <-ctx.Done():
				// Context cancelled, handle graceful termination
				m.Logger().Debug("Context cancelled. Exiting gracefully.")
				close(c)
				return nil
			case healthDetails, ok := <-c:
				if !ok {
					return nil
				}
				m.Logger().Debugf("[AWS Health] [DescribeEventDetails] Event ARN : %s, Affected Entities (Pending) : %d, Affected Entities (Resolved): %d, Affected Entities (Others) : %d", *healthDetails.event.Arn, healthDetails.affectedEntityPending, healthDetails.affectedEntityResolved, healthDetails.affectedEntityOthers)
				events = append(events, createEvents(healthDetails))
			}
		}
		if eventOutput.NextToken == nil {
			break
		} else {
			nextTokenString = *eventOutput.NextToken
		}
		time.Sleep(10 * time.Millisecond)
		close(c)
	}
	return events
}

func createEvents(hd HealthDetails) mb.Event {
	event := mb.Event{}
	event.MetricSetFields = mapstr.M{
		"event_arn":                  getStringValueOrDefault(hd.event.Arn),
		"end_time":                   getTimeValueOrDefault(hd.event.EndTime),
		"event_scope_code":           getStringValueOrDefault((*string)(&hd.event.EventScopeCode)),
		"event_type_category":        getStringValueOrDefault((*string)(&hd.event.EventTypeCategory)),
		"event_type_code":            getStringValueOrDefault(hd.event.EventTypeCode),
		"last_updated_time":          getTimeValueOrDefault(hd.event.LastUpdatedTime),
		"region":                     getStringValueOrDefault(hd.event.Region),
		"service":                    getStringValueOrDefault(hd.event.Service),
		"start_time":                 getTimeValueOrDefault(hd.event.StartTime),
		"status_code":                getStringValueOrDefault((*string)(&hd.event.StatusCode)),
		"affected_entities_pending":  hd.affectedEntityPending,
		"affected_entities_resolved": hd.affectedEntityResolved,
		"affected_entities_others":   hd.affectedEntityOthers,
		"affected_entities":          createAffectedEntityDetails(hd.affectedEntities),
	}
	event.RootFields = mapstr.M{
		"cloud.provider": "aws",
	}
	currentDate := getCurrentDateTime()
	eventID := currentDate + getStringValueOrDefault(hd.event.Arn) + getStringValueOrDefault((*string)(&hd.event.StatusCode))
	event.ID = generateEventID(eventID)
	return event
}

type HealthDetails struct {
	event                  types.Event
	eventDescription       string
	affectedEntities       []types.AffectedEntity
	affectedEntityPending  int32
	affectedEntityResolved int32
	affectedEntityOthers   int32
}

type AffectedEntityDetails struct {
	AwsAccountId    string    `json:"aws_account_id"`
	EntityUrl       string    `json:"entity_url"`
	EntityValue     string    `json:"entity_value"`
	LastUpdatedTime time.Time `json:"last_updated_time"`
	StatusCode      string    `json:"status_code"`
	EntityArn       string    `json:"entity_arn"`
}

// getStringValueOrDefault returns the string value or an empty string if the pointer is nil.
func getStringValueOrDefault(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

func getTimeValueOrDefault(t *time.Time) time.Time {
	if t != nil {
		return *t
	}
	return time.Time{}
}

// createAffectedEntityDetails populates and returns a slice of AffectedEntityDetails
// based on the given list of AffectedEntity instances.
// Each AffectedEntity is converted into an AffectedEntityDetails struct,
func createAffectedEntityDetails(affectedEntities []types.AffectedEntity) []AffectedEntityDetails {
	aed := []AffectedEntityDetails{}
	// Populate a slice of AffectedEntityDetails
	for _, entity := range affectedEntities {
		aed = append(aed, AffectedEntityDetails{
			AwsAccountId:    getStringValueOrDefault(entity.AwsAccountId),
			EntityUrl:       getStringValueOrDefault(entity.EntityUrl),
			EntityValue:     getStringValueOrDefault(entity.EntityValue),
			LastUpdatedTime: getTimeValueOrDefault(entity.LastUpdatedTime),
			StatusCode:      string(entity.StatusCode),
			EntityArn:       getStringValueOrDefault(entity.EntityArn),
		})
	}
	return aed

}

func generateEventID(eventID string) string {
	h := sha256.New()
	h.Write([]byte(eventID))
	prefix := hex.EncodeToString(h.Sum(nil))
	return prefix[:20]
}

func (m *MetricSet) getDescribeEventDetails(ctx context.Context, awsHealth *health.Client, event types.Event, ch chan<- HealthDetails) {
	var hd HealthDetails
	hd.event = event
	eventDetails, err := awsHealth.DescribeEventDetails(ctx, &health.DescribeEventDetailsInput{
		EventArns: []string{*event.Arn},
		Locale:    &Locale,
	})
	if err != nil {
		if ctx.Err() == context.Canceled {
			m.Logger().Debug("Context cancelled. Exiting gracefully.")
			return
		}
		err = fmt.Errorf("AWS Health DescribeEventDetails failed with %w", err)
		m.Logger().Error(err.Error())
		return
	} else {
		hd.eventDescription = *(eventDetails.SuccessfulSet[0].EventDescription.LatestDescription)
	}

	var affEntityTokString = ""
	var nextToken *string
	var pending int32 = 0
	var resolved int32 = 0
	var others int32 = 0
	for {
		if affEntityTokString == "" {
			affectedEntities, err := awsHealth.DescribeAffectedEntities(ctx, &health.DescribeAffectedEntitiesInput{
				Filter: &types.EntityFilter{
					EventArns: []string{*event.Arn},
				},
				Locale:     &Locale,
				MaxResults: &maxResults,
			})
			if err != nil {
				err = fmt.Errorf("AWS Health DescribeAffectedEntities failed with %w", err)

				// Check if the error is due to context cancellation
				if ctx.Err() == context.Canceled {
					m.Logger().Info("Context cancelled. Exiting gracefully.")
					return
				}
				// Handle other errors
				m.Logger().Error(err.Error())
				return
			}
			if affectedEntities != nil {
				nextToken = affectedEntities.NextToken

				hd.affectedEntities = append(hd.affectedEntities, affectedEntities.Entities...)
				for _, affEntity := range affectedEntities.Entities {
					if affEntity.StatusCode != "" {
						if affEntity.StatusCode == "PENDING" {
							pending++
						} else if affEntity.StatusCode == "RESOLVED" {
							resolved++
						} else {
							others++
						}
					}
				}
			}

		} else {
			affectedEntities, err := awsHealth.DescribeAffectedEntities(ctx, &health.DescribeAffectedEntitiesInput{
				Filter: &types.EntityFilter{
					EventArns: []string{*event.Arn},
				},
				Locale:     &Locale,
				MaxResults: &maxResults,
				NextToken:  &affEntityTokString,
			})
			if err != nil {
				err = fmt.Errorf("AWS Health DescribeAffectedEntities failed with %w", err)

				// Check if the error is due to context cancellation
				if ctx.Err() == context.Canceled {
					m.Logger().Info("Context cancelled. Exiting gracefully.")
					return
				}
				// Handle other errors
				m.Logger().Error(err.Error())
				return
			}
			if affectedEntities != nil {
				nextToken = affectedEntities.NextToken

				// nextToken = affectedEntities.NextToken
				hd.affectedEntities = append(hd.affectedEntities, affectedEntities.Entities...)

				for _, affEntity := range affectedEntities.Entities {
					if affEntity.StatusCode != "" {
						if affEntity.StatusCode == "PENDING" {
							pending++
						} else if affEntity.StatusCode == "RESOLVED" {
							resolved++
						} else {
							others++
						}
					}
				}
			}
		}
		if nextToken == nil {
			break
		} else {
			affEntityTokString = *nextToken
		}
	}
	hd.affectedEntityResolved = resolved
	hd.affectedEntityPending = pending
	hd.affectedEntityOthers = others
	select {
	case ch <- hd:
		// Writing to the channel
	default:
		// Channel is closed,
		return
	}
	time.Sleep(10 * time.Millisecond)
}

func getCurrentDateTime() string {
	currentTime := time.Now()
	return fmt.Sprintf("%04d%02d%02d%02d%02d%02d", currentTime.Year(), int(currentTime.Month()), currentTime.Day(), currentTime.Hour(), currentTime.Minute(), currentTime.Second())
}
