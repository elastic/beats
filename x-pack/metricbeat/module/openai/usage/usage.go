// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/paths"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("openai", "usage", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	httpClient   *RLHTTPClient
	logger       *logp.Logger
	config       Config
	report       mb.ReporterV2
	stateManager *stateManager
	headers      map[string]string
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The openai usage metricset is beta.")

	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sm, err := newStateManager(paths.Resolve(paths.Data, path.Join("state", base.Module().Name(), base.Name())))
	if err != nil {
		return nil, fmt.Errorf("create state manager: %w", err)
	}

	logger := logp.NewLogger("openai.usage")

	httpClient := newClient(
		logger,
		rate.NewLimiter(
			rate.Every(time.Duration(*config.RateLimit.Limit)*time.Second),
			*config.RateLimit.Burst,
		),
		config.Timeout,
	)

	return &MetricSet{
		BaseMetricSet: base,
		httpClient:    httpClient,
		logger:        logger,
		config:        config,
		stateManager:  sm,
		headers:       processHeaders(config.Headers),
	}, nil
}

// Fetch collects OpenAI API usage data for the configured time range.
//
// The collection process:
// 1. Determines the time range based on realtime/non-realtime configuration
// 2. Calculates start date using configured lookback days
// 3. Fetches usage data for each day in the range
// 4. Reports collected metrics through the mb.ReporterV2
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	endDate := time.Now().UTC().Truncate(time.Hour * 24) // truncate to day as we only collect daily data

	if !m.config.Collection.Realtime {
		// If we're not collecting realtime data, then just pull until
		// yesterday (in UTC).
		endDate = endDate.AddDate(0, 0, -1)
	}

	startDate := endDate.AddDate(0, 0, -m.config.Collection.LookbackDays)

	m.report = report
	return m.fetchDateRange(startDate, endDate, m.httpClient)
}

// fetchDateRange retrieves OpenAI API usage data for each configured API key within a date range.
//
// For each API key:
// 1. Retrieves last processed date from state store
// 2. Adjusts collection range to avoid duplicates
// 3. Collects daily usage data
// 4. Updates state store with latest processed date
// 5. Handles errors per day without failing entire range
func (m *MetricSet) fetchDateRange(startDate, endDate time.Time, httpClient *RLHTTPClient) error {
	g, ctx := errgroup.WithContext(context.TODO())

	for i := range m.config.APIKeys {
		apiKey := m.config.APIKeys[i]
		apiKeyIdx := i + 1
		g.Go(func() error {
			lastProcessedDate, err := m.stateManager.GetLastProcessedDate(apiKey.Key)
			if err == nil {
				currentStartDate := lastProcessedDate.AddDate(0, 0, 1)
				if currentStartDate.After(endDate) {
					m.logger.Infof("Skipping API key #%d as current start date (%s) is after end date (%s)", apiKeyIdx, currentStartDate, endDate)
					return nil
				}
				startDate = currentStartDate
			}

			m.logger.Debugf("Fetching data for API key #%d from %s to %s", apiKeyIdx, startDate, endDate)

			for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					dateStr := d.Format(dateFormatForStateStore)
					if err := m.fetchSingleDay(apiKeyIdx, dateStr, apiKey.Key, httpClient); err != nil {
						// If there's an error, log it and continue to the next day.
						// In this case, we are not saving the state.
						m.logger.Errorf("Error fetching data (api key #%d) for date %s: %v", apiKeyIdx, dateStr, err)
						continue
					}
					if err := m.stateManager.SaveState(apiKey.Key, dateStr); err != nil {
						m.logger.Errorf("Error storing state for API key: %v at index %d", err, apiKeyIdx)
					}
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		m.logger.Errorf("Error fetching data: %v", err)
	}

	return nil
}

// fetchSingleDay retrieves usage data for a specific date and API key.
func (m *MetricSet) fetchSingleDay(apiKeyIdx int, dateStr, apiKey string, httpClient *RLHTTPClient) error {
	req, err := m.createRequest(dateStr, apiKey)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return fmt.Errorf("request timed out with configured timeout: %v and error: %w", m.config.Timeout, err)
		}
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from API: status=%s", resp.Status)
	}

	return m.processResponse(apiKeyIdx, resp, dateStr)
}

// createRequest builds an HTTP request for the OpenAI usage API.
func (m *MetricSet) createRequest(dateStr, apiKey string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, m.config.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("date", dateStr)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	for key, value := range m.headers {
		req.Header.Add(key, value)
	}

	return req, nil
}

// processResponse handles the API response and processes the usage data.
func (m *MetricSet) processResponse(apiKeyIdx int, resp *http.Response, dateStr string) error {
	var usageResponse UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResponse); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	m.logger.Infof("Fetching usage metrics (api key #%d) for date: %s", apiKeyIdx, dateStr)

	m.processUsageData(usageResponse.Data)
	m.processDalleData(usageResponse.DalleApiData)
	m.processWhisperData(usageResponse.WhisperApiData)
	m.processTTSData(usageResponse.TtsApiData)

	// Process additional data.
	//
	// NOTE(shmsr): During testing, could not get the usage data for the following
	// and found no documentation, example responses, etc. That's why let's store them
	// as it is so that we can use processors later on to process them as needed.
	m.processFTData(usageResponse.FtData)
	m.processAssistantCodeInterpreterData(usageResponse.AssistantCodeInterpreterData)
	m.processRetrievalStorageData(usageResponse.RetrievalStorageData)

	return nil
}

func getBaseFields(data BaseData) mapstr.M {
	return mapstr.M{
		"organization_id":   data.OrganizationID,
		"organization_name": data.OrganizationName,
		"api_key_id":        data.ApiKeyID,
		"api_key_name":      data.ApiKeyName,
		"api_key_redacted":  data.ApiKeyRedacted,
		"api_key_type":      data.ApiKeyType,
		"project_id":        data.ProjectID,
		"project_name":      data.ProjectName,
	}
}

func (m *MetricSet) processUsageData(data []UsageData) {
	events := make([]mb.Event, 0, len(data))
	for _, usage := range data {
		event := mb.Event{
			Timestamp: time.Unix(usage.AggregationTimestamp, 0).UTC(), // epoch time to time.Time (UTC)
			MetricSetFields: mapstr.M{
				"data": mapstr.M{
					"requests_total":              usage.NRequests,
					"operation":                   usage.Operation,
					"snapshot_id":                 usage.SnapshotID,
					"context_tokens_total":        usage.NContextTokensTotal,
					"generated_tokens_total":      usage.NGeneratedTokensTotal,
					"email":                       usage.Email,
					"request_type":                usage.RequestType,
					"cached_context_tokens_total": usage.NCachedContextTokensTotal,
				},
			},
		}
		event.MetricSetFields.DeepUpdate(getBaseFields(usage.BaseData))
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processDalleData(data []DalleData) {
	events := make([]mb.Event, 0, len(data))
	for _, dalle := range data {
		event := mb.Event{
			Timestamp: time.Unix(dalle.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
			MetricSetFields: mapstr.M{
				"dalle": mapstr.M{
					"num_images":     dalle.NumImages,
					"requests_total": dalle.NumRequests,
					"image_size":     dalle.ImageSize,
					"operation":      dalle.Operation,
					"user_id":        dalle.UserID,
					"model_id":       dalle.ModelID,
				},
			},
		}
		event.MetricSetFields.DeepUpdate(getBaseFields(dalle.BaseData))
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processWhisperData(data []WhisperData) {
	events := make([]mb.Event, 0, len(data))
	for _, whisper := range data {
		event := mb.Event{
			Timestamp: time.Unix(whisper.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
			MetricSetFields: mapstr.M{
				"whisper": mapstr.M{
					"model_id":       whisper.ModelID,
					"num_seconds":    whisper.NumSeconds,
					"requests_total": whisper.NumRequests,
					"user_id":        whisper.UserID,
				},
			},
		}
		event.MetricSetFields.DeepUpdate(getBaseFields(whisper.BaseData))
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processTTSData(data []TtsData) {
	events := make([]mb.Event, 0, len(data))
	for _, tts := range data {
		event := mb.Event{
			Timestamp: time.Unix(tts.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
			MetricSetFields: mapstr.M{
				"tts": mapstr.M{
					"model_id":       tts.ModelID,
					"num_characters": tts.NumCharacters,
					"requests_total": tts.NumRequests,
					"user_id":        tts.UserID,
				},
			},
		}
		event.MetricSetFields.DeepUpdate(getBaseFields(tts.BaseData))
		events = append(events, event)
	}

	m.processEvents(events)
}

func (m *MetricSet) processFTData(data []interface{}) {
	events := make([]mb.Event, 0, len(data))
	for _, ft := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"ft_data": mapstr.M{
					"original": ft,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processAssistantCodeInterpreterData(data []interface{}) {
	events := make([]mb.Event, 0, len(data))
	for _, aci := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"assistant_code_interpreter": mapstr.M{
					"original": aci,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processRetrievalStorageData(data []interface{}) {
	events := make([]mb.Event, 0, len(data))
	for _, rs := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"retrieval_storage": mapstr.M{
					"original": rs,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processEvents(events []mb.Event) {
	if len(events) == 0 {
		return
	}
	for i := range events {
		m.report.Event(events[i])
	}
}
