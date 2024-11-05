// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package usage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
	logger     *logp.Logger
	config     Config
	report     mb.ReporterV2
	stateStore *stateStore
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

	st, err := newStateStore(paths.Resolve(paths.Data, path.Join(base.Module().Name(), base.Name())))
	if err != nil {
		return nil, fmt.Errorf("creating state store: %w", err)
	}

	return &MetricSet{
		BaseMetricSet: base,
		logger:        logp.NewLogger("openai.usage"),
		config:        config,
		stateStore:    st,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	httpClient := newClient(
		context.TODO(),
		m.logger,
		rate.NewLimiter(
			rate.Every(time.Duration(*m.config.RateLimit.Limit)*time.Second),
			*m.config.RateLimit.Burst,
		),
		m.config.Timeout,
	)

	m.report = report

	endDate := time.Now().UTC()
	startDate := endDate.AddDate(0, 0, -m.config.Collection.LookbackDays)

	return m.fetchDateRange(startDate, endDate, httpClient)
}

func (m *MetricSet) fetchDateRange(startDate, endDate time.Time, httpClient *RLHTTPClient) error {
	for _, apiKey := range m.config.APIKeys {
		// SHA-256 produces a fixed-length (64 characters) hexadecimal string
		// that is safe for filenames across all major platforms. Hex encoding
		// ensures that the hash is safe for use in file paths as it uses only
		// alphanumeric characters.
		//
		// Also, SHA-256 is a strong cryptographic hash function that is
		// deterministic, meaning that the same input will always produce
		// the same output and it is an one-way function, meaning that it is
		// computationally infeasible to reverse the hash to obtain the
		// original.
		hasher := sha256.New()
		hasher.Write([]byte(apiKey.Key))
		hashedKey := hex.EncodeToString(hasher.Sum(nil))
		stateKey := "state_" + hashedKey

		// If state exists, only fetch current day
		if m.stateStore.Has(stateKey) {
			currentDay := endDate.Format("2006-01-02")
			if err := m.fetchSingleDay(currentDay, apiKey.Key, httpClient); err != nil {
				m.logger.Errorf("Error fetching data for date %s: %v", currentDay, err)
			}
			continue
		}

		// First run for this API key - fetch historical data
		for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			if err := m.fetchSingleDay(dateStr, apiKey.Key, httpClient); err != nil {
				m.logger.Errorf("Error fetching data for date %s: %v", dateStr, err)
				continue
			}
		}

		// Mark this API key as processed
		if err := m.stateStore.Put(stateKey); err != nil {
			m.logger.Errorf("Error storing state for API key: %v", err)
		}
	}
	return nil
}

func (m *MetricSet) fetchSingleDay(dateStr, apiKey string, httpClient *RLHTTPClient) error {
	req, err := m.createRequest(dateStr, apiKey)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error response from API: %s", resp.Status)
	}

	return m.processResponse(resp, dateStr)
}

func (m *MetricSet) createRequest(dateStr, apiKey string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, m.config.APIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("date", dateStr)
	req.URL.RawQuery = q.Encode()

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	for key, value := range processHeaders(m.config.Headers) {
		req.Header.Add(key, value)
	}

	return req, nil
}

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
					"aggregation_timestamp":         time.Unix(usage.AggregationTimestamp, 0),
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
					"timestamp":         time.Unix(dalle.Timestamp, 0),
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
					"timestamp":         time.Unix(whisper.Timestamp, 0),
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
					"timestamp":         time.Unix(tts.Timestamp, 0),
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
