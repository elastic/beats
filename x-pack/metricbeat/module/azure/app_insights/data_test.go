// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package app_insights

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/Azure/azure-sdk-for-go/services/preview/appinsights/v1/insights"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func newMetricsTest(ts ...*date.Time) []MetricValue {
	type values struct {
		SegmentName map[string]string
		Value       map[string]interface{}
		T           *date.Time
	}

	const numOfMetricValue = 3

	if numOfMetricValue != len(ts) {
		panic("number of arguments to newMetricsTest is not correct")
	}

	vals := [numOfMetricValue]values{
		{
			SegmentName: map[string]string{"request_url_host": ""},
			Value:       map[string]interface{}{"users_count.unique": 44},
			T:           ts[0],
		},
		{
			SegmentName: map[string]string{"request_url_host": ""},
			Value:       map[string]interface{}{"sessions_count.unique": 44},
			T:           ts[1],
		},
		{
			SegmentName: map[string]string{"request_url_host": "localhost"},
			Value:       map[string]interface{}{"sessions_count.unique": 44},
			T:           ts[2],
		},
	}

	mv := make([]MetricValue, 0, numOfMetricValue)
	for i := range vals {
		mv = append(mv,
			MetricValue{
				SegmentName: map[string]string{},
				Value:       map[string]interface{}{},
				Segments: []MetricValue{
					{
						SegmentName: map[string]string{},
						Value:       map[string]interface{}{},
						Segments: []MetricValue{
							{
								SegmentName: vals[i].SegmentName,
								Value:       vals[i].Value,
							},
						},
					},
				},
				Interval: "P5M",
				Start:    vals[i].T, End: vals[i].T,
			},
		)
	}

	return mv
}

func TestGroupMetrics(t *testing.T) {
	t.Run("two dimensions groups with same timestamps", func(t *testing.T) {
		timestamp1 := &date.Time{Time: time.Now()}
		timestamp2 := &date.Time{Time: time.Now()}
		timestamp3 := &date.Time{Time: time.Now()}

		metrics := newMetricsTest(timestamp1, timestamp2, timestamp3)

		expectedGroup1 := []MetricValue{
			{
				SegmentName: map[string]string{
					"request_url_host": "",
				},
				Value: map[string]interface{}{
					"users_count.unique": 44,
				},
				Start: timestamp1,
				End:   timestamp1,
			},
			{
				SegmentName: map[string]string{
					"request_url_host": "",
				},
				Value: map[string]interface{}{
					"sessions_count.unique": 44,
				},
				Start: timestamp2,
				End:   timestamp2,
			},
		}

		expectedGroup2 := []MetricValue{
			{
				SegmentName: map[string]string{
					"request_url_host": "localhost",
				},
				Value: map[string]interface{}{
					"sessions_count.unique": 44,
				},
				Start: timestamp3,
				End:   timestamp3,
			},
		}

		groupedByDimensions := groupMetricsByDimension(metrics)
		assert.Len(t, groupedByDimensions, 2)

		k1 := fmt.Sprintf("%d_%d_%s", timestamp1.Unix(), timestamp1.Unix(), "request_url_host")

		dimensionsGroup1, ok := groupedByDimensions[k1]
		assert.True(t, ok)
		assert.Len(t, dimensionsGroup1, 2)
		assert.ElementsMatch(t, dimensionsGroup1, expectedGroup1)

		k2 := fmt.Sprintf("%d_%d_%s", timestamp3.Unix(), timestamp3.Unix(), "request_url_hostlocalhost")

		dimensionsGroup2, ok := groupedByDimensions[k2]
		assert.True(t, ok)
		assert.Len(t, dimensionsGroup2, 1)
		assert.ElementsMatch(t, dimensionsGroup2, expectedGroup2)
	})

	t.Run("two dimensions groups with different timestamps", func(t *testing.T) {
		timestamp1 := &date.Time{Time: time.Now()}
		timestamp2 := &date.Time{Time: time.Now().Add(time.Minute)}
		timestamp3 := &date.Time{Time: time.Now().Add(2 * time.Minute)}

		metrics := newMetricsTest(timestamp1, timestamp2, timestamp3)

		expectedDimensionsGroup1 := []MetricValue{
			{
				SegmentName: map[string]string{
					"request_url_host": "",
				},
				Value: map[string]interface{}{
					"users_count.unique": 44,
				},
				Start: timestamp1,
				End:   timestamp1,
			},
		}

		expectedDimensionsGroup2 := []MetricValue{
			{
				SegmentName: map[string]string{
					"request_url_host": "",
				},
				Value: map[string]interface{}{
					"sessions_count.unique": 44,
				},
				Start: timestamp2,
				End:   timestamp2,
			},
		}

		expectedDimensionsGroup3 := []MetricValue{
			{
				SegmentName: map[string]string{
					"request_url_host": "localhost",
				},
				Value: map[string]interface{}{
					"sessions_count.unique": 44,
				},
				Start: timestamp3,
				End:   timestamp3,
			},
		}

		groupedByDimensions := groupMetricsByDimension(metrics)
		assert.Len(t, groupedByDimensions, 3)

		k1 := fmt.Sprintf("%d_%d_%s", timestamp1.Unix(), timestamp1.Unix(), "request_url_host")

		dimensionsGroup1, ok := groupedByDimensions[k1]
		assert.True(t, ok)
		assert.Len(t, dimensionsGroup1, 1)
		assert.ElementsMatch(t, dimensionsGroup1, expectedDimensionsGroup1)

		k2 := fmt.Sprintf("%d_%d_%s", timestamp2.Unix(), timestamp2.Unix(), "request_url_host")

		dimensionsGroup2, ok := groupedByDimensions[k2]
		assert.True(t, ok)
		assert.Len(t, dimensionsGroup2, 1)
		assert.ElementsMatch(t, dimensionsGroup2, expectedDimensionsGroup2)

		k3 := fmt.Sprintf("%d_%d_%s", timestamp3.Unix(), timestamp3.Unix(), "request_url_hostlocalhost")

		dimensionsGroup3, ok := groupedByDimensions[k3]
		assert.True(t, ok)
		assert.Len(t, dimensionsGroup3, 1)
		assert.ElementsMatch(t, dimensionsGroup3, expectedDimensionsGroup3)
	})
}

func TestEventMapping(t *testing.T) {
	startDate := date.Time{Time: time.Now()}
	id := "123"
	var info = insights.MetricsResultInfo{
		AdditionalProperties: map[string]interface{}{
			"requests/count":  map[string]interface{}{"sum": 12},
			"requests/failed": map[string]interface{}{"sum": 10},
		},
		Start: &startDate,
		End:   &startDate,
	}
	var metricResult = insights.MetricsResult{
		Value: &info,
	}
	metrics := []insights.MetricsResultsItem{
		{
			ID:     &id,
			Status: nil,
			Body:   &metricResult,
		},
	}
	var result = insights.ListMetricsResultsItem{
		Value: &metrics,
	}
	applicationId := "abc"
	events := EventsMapping(result, applicationId, "")
	assert.Equal(t, len(events), 1)
	for _, event := range events {
		val1, _ := event.MetricSetFields.GetValue("start_date")
		assert.Equal(t, val1, startDate.Time)
		val2, _ := event.MetricSetFields.GetValue("end_date")
		assert.Equal(t, val2, startDate.Time)
		val3, _ := event.ModuleFields.GetValue("metrics.requests_count")
		assert.Equal(t, val3, mapstr.M{"sum": 12})
		val5, _ := event.ModuleFields.GetValue("metrics.requests_failed")
		assert.Equal(t, val5, mapstr.M{"sum": 10})
		val4, _ := event.ModuleFields.GetValue("application_id")
		assert.Equal(t, val4, applicationId)

	}

}

func TestEventMappingGrouping(t *testing.T) {
	start, err := time.Parse("2006-01-02T15:04:05Z", "2023-09-20T18:08:31Z")
	assert.NoError(t, err)

	end, err := time.Parse("2006-01-02T15:04:05Z", "2023-09-20T18:09:31Z")
	assert.NoError(t, err)

	interval := "P152D"
	results := []insights.MetricsResultsItem{
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"request/urlHost": "",
										"users/count":     map[string]interface{}{"unique": 1.0},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"sessions/count":  map[string]interface{}{"unique": 1.0},
										"request/urlHost": "",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"browserTiming/urlHost": "localhost",
									},
									Segments: &[]insights.MetricsSegmentInfo{
										{
											AdditionalProperties: map[string]interface{}{
												"browserTiming/urlPath":          "/test",
												"browserTimings/networkDuration": map[string]interface{}{"avg": 1.5},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"browserTiming/urlHost": "localhost",
									},
									Segments: &[]insights.MetricsSegmentInfo{
										{
											AdditionalProperties: map[string]interface{}{
												"browserTimings/sendDuration": map[string]interface{}{"avg": 1.25},
												"browserTiming/urlPath":       "/test",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"browserTiming/urlHost": "localhost",
									},
									Segments: &[]insights.MetricsSegmentInfo{
										{
											AdditionalProperties: map[string]interface{}{
												"browserTimings/receiveDuration": map[string]interface{}{"avg": 0.0},
												"browserTiming/urlPath":          "/test",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"browserTiming/urlHost": "localhost",
									},
									Segments: &[]insights.MetricsSegmentInfo{
										{
											AdditionalProperties: map[string]interface{}{
												"browserTimings/processingDuration": map[string]interface{}{"avg": 18.25},
												"browserTiming/urlPath":             "/test",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Body: &insights.MetricsResult{
				Value: &insights.MetricsResultInfo{
					Start:    &date.Time{Time: start},
					End:      &date.Time{Time: end},
					Interval: &interval,
					Segments: &[]insights.MetricsSegmentInfo{
						{
							Start: &date.Time{Time: start},
							End:   &date.Time{Time: end},
							Segments: &[]insights.MetricsSegmentInfo{
								{
									AdditionalProperties: map[string]interface{}{
										"browserTiming/urlHost": "localhost",
									},
									Segments: &[]insights.MetricsSegmentInfo{
										{
											AdditionalProperties: map[string]interface{}{
												"browserTimings/totalDuration": map[string]interface{}{"avg": 22},
												"browserTiming/urlPath":        "/test",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := insights.ListMetricsResultsItem{
		Value: &results,
	}

	expectedEvents := []mb.Event{
		{
			RootFields: mapstr.M{
				"cloud": mapstr.M{
					"provider": "azure",
				},
			},
			ModuleFields: mapstr.M{
				"application_id": "2c944c0d-5231-43bb-a59a-dba54894c8d9",
				"dimensions": map[string]string{
					"browser_timing_url_path": "/test",
					"browser_timing_url_host": "localhost",
				},
				"metrics": mapstr.M{
					"browser_timings_network_duration":    mapstr.M{"avg": 1.5},
					"browser_timings_send_duration":       mapstr.M{"avg": 1.25},
					"browser_timings_receive_duration":    mapstr.M{"avg": 0.0},
					"browser_timings_processing_duration": mapstr.M{"avg": 18.25},
					"browser_timings_total_duration":      mapstr.M{"avg": 22},
				},
			},
			MetricSetFields: mapstr.M{
				"start_date": start,
				"end_date":   end,
			},
			Timestamp: end,
		},
		{
			RootFields: mapstr.M{
				"cloud": mapstr.M{
					"provider": "azure",
				},
			},
			ModuleFields: mapstr.M{
				"application_id": "2c944c0d-5231-43bb-a59a-dba54894c8d9",
				"dimensions": map[string]string{
					"request_url_host": "",
				},
				"metrics": mapstr.M{
					"users_count":    mapstr.M{"unique": 1.0},
					"sessions_count": mapstr.M{"unique": 1.0},
				},
			},
			MetricSetFields: mapstr.M{
				"start_date": start,
				"end_date":   end,
			},
			Timestamp: end,
		},
	}

	events := EventsMapping(result, "2c944c0d-5231-43bb-a59a-dba54894c8d9", "")
	assert.Equal(t, len(events), 2)
	assert.ElementsMatch(t, expectedEvents, events)
}

func TestCleanMetricNames(t *testing.T) {
	ex := "customDimensions/ExecutingAssemblyFileVersion"
	result := cleanMetricNames(ex)
	assert.Equal(t, result, "custom_dimensions_executing_assembly_file_version")
	ex = "customDimensions/_MS.AggregationIntervalMs"
	result = cleanMetricNames(ex)
	assert.Equal(t, result, "custom_dimensions__ms_aggregation_interval_ms")
	ex = "customDimensions/_MS.IsAutocollected"
	result = cleanMetricNames(ex)
	assert.Equal(t, result, "custom_dimensions__ms_is_autocollected")
}
