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
	endDate := time.Now().UTC()

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
	for _, apiKey := range m.config.APIKeys {
		// stateKey using stateManager's key prefix and hashing apiKey
		stateKey := m.stateManager.GetStateKey(apiKey.Key)

		lastProcessedDate, err := m.stateManager.GetLastProcessedDate(apiKey.Key)
		if err == nil {
			// We have previous state, adjust start date
			startDate = lastProcessedDate.AddDate(0, 0, 1)
			if startDate.After(endDate) {
				continue
			}
		}

		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			if err := m.fetchSingleDay(dateStr, apiKey.Key, httpClient); err != nil {
				m.logger.Errorf("Error fetching data for date %s: %v", dateStr, err)
				continue
			}
			if err := m.stateManager.store.Put(stateKey, dateStr); err != nil {
				m.logger.Errorf("Error storing state for API key: %v", err)
			}
		}
	}
	return nil
}

// fetchSingleDay retrieves usage data for a specific date and API key.
func (m *MetricSet) fetchSingleDay(dateStr, apiKey string, httpClient *RLHTTPClient) error {
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

	return m.processResponse(resp, dateStr)
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
func (m *MetricSet) processResponse(resp *http.Response, dateStr string) error {
	var usageResponse UsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&usageResponse); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	m.logger.Info("Fetched usage metrics for date:", dateStr)

	events := make([]mb.Event, 0, len(usageResponse.Data))

	m.processUsageData(events, usageResponse.Data)
	m.processDalleData(events, usageResponse.DalleApiData)
	m.processWhisperData(events, usageResponse.WhisperApiData)
	m.processTTSData(events, usageResponse.TtsApiData)

	// Process additional data.
	//
	// NOTE(shmsr): During testing, could not get the usage data for the following
	// and found no documentation, example responses, etc. That's why let's store them
	// as it is so that we can use processors later on to process them as needed.
	m.processFTData(events, usageResponse.FtData)
	m.processAssistantCodeInterpreterData(events, usageResponse.AssistantCodeInterpreterData)
	m.processRetrievalStorageData(events, usageResponse.RetrievalStorageData)

	return nil
}

func (m *MetricSet) processUsageData(events []mb.Event, data []UsageData) {

	for _, usage := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"data": mapstr.M{
					"organization_id":               usage.OrganizationID,
					"organization_name":             usage.OrganizationName,
					"aggregation_timestamp":         time.Unix(usage.AggregationTimestamp, 0).UTC(), // epoch time to time.Time (UTC)
					"n_requests":                    usage.NRequests,
					"operation":                     usage.Operation,
					"snapshot_id":                   usage.SnapshotID,
					"n_context_tokens_total":        usage.NContextTokensTotal,
					"n_generated_tokens_total":      usage.NGeneratedTokensTotal,
					"email":                         usage.Email,
					"api_key_id":                    usage.ApiKeyID,
					"api_key_name":                  usage.ApiKeyName,
					"api_key_redacted":              usage.ApiKeyRedacted,
					"api_key_type":                  usage.ApiKeyType,
					"project_id":                    usage.ProjectID,
					"project_name":                  usage.ProjectName,
					"request_type":                  usage.RequestType,
					"n_cached_context_tokens_total": usage.NCachedContextTokensTotal,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processDalleData(events []mb.Event, data []DalleData) {
	for _, dalle := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"dalle": mapstr.M{
					"timestamp":         time.Unix(dalle.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
					"num_images":        dalle.NumImages,
					"num_requests":      dalle.NumRequests,
					"image_size":        dalle.ImageSize,
					"operation":         dalle.Operation,
					"user_id":           dalle.UserID,
					"organization_id":   dalle.OrganizationID,
					"api_key_id":        dalle.ApiKeyID,
					"api_key_name":      dalle.ApiKeyName,
					"api_key_redacted":  dalle.ApiKeyRedacted,
					"api_key_type":      dalle.ApiKeyType,
					"organization_name": dalle.OrganizationName,
					"model_id":          dalle.ModelID,
					"project_id":        dalle.ProjectID,
					"project_name":      dalle.ProjectName,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processWhisperData(events []mb.Event, data []WhisperData) {
	for _, whisper := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"whisper": mapstr.M{
					"timestamp":         time.Unix(whisper.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
					"model_id":          whisper.ModelID,
					"num_seconds":       whisper.NumSeconds,
					"num_requests":      whisper.NumRequests,
					"user_id":           whisper.UserID,
					"organization_id":   whisper.OrganizationID,
					"api_key_id":        whisper.ApiKeyID,
					"api_key_name":      whisper.ApiKeyName,
					"api_key_redacted":  whisper.ApiKeyRedacted,
					"api_key_type":      whisper.ApiKeyType,
					"organization_name": whisper.OrganizationName,
					"project_id":        whisper.ProjectID,
					"project_name":      whisper.ProjectName,
				},
			},
		}
		events = append(events, event)
	}
	m.processEvents(events)
}

func (m *MetricSet) processTTSData(events []mb.Event, data []TtsData) {
	for _, tts := range data {
		event := mb.Event{
			MetricSetFields: mapstr.M{
				"tts": mapstr.M{
					"timestamp":         time.Unix(tts.Timestamp, 0).UTC(), // epoch time to time.Time (UTC)
					"model_id":          tts.ModelID,
					"num_characters":    tts.NumCharacters,
					"num_requests":      tts.NumRequests,
					"user_id":           tts.UserID,
					"organization_id":   tts.OrganizationID,
					"api_key_id":        tts.ApiKeyID,
					"api_key_name":      tts.ApiKeyName,
					"api_key_redacted":  tts.ApiKeyRedacted,
					"api_key_type":      tts.ApiKeyType,
					"organization_name": tts.OrganizationName,
					"project_id":        tts.ProjectID,
					"project_name":      tts.ProjectName,
				},
			},
		}
		events = append(events, event)
	}

	m.processEvents(events)
}

func (m *MetricSet) processFTData(events []mb.Event, data []interface{}) {
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

func (m *MetricSet) processAssistantCodeInterpreterData(events []mb.Event, data []interface{}) {
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

func (m *MetricSet) processRetrievalStorageData(events []mb.Event, data []interface{}) {
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
	if len(events) > 0 {
		for i := range events {
			m.report.Event(events[i])
		}
	}
	clear(events)
}
