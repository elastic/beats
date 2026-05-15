// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	meraki "github.com/meraki/dashboard-api-go/v3/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestGetSensorReadingsHistory(t *testing.T) {
	logger := logp.NewLogger("test")

	tests := []struct {
		name             string
		client           SensorClient
		expectedReadings int
		expectedCalls    int
		wantErr          bool
	}{
		{
			name:             "single page",
			client:           newMockSensorClient(1, 3), // 1 page, 3 readings
			expectedReadings: 3,
			expectedCalls:    1,
		},
		{
			name:             "multiple pages",
			client:           newMockSensorClient(3, 2), // 3 pages, 2 readings per page
			expectedReadings: 6,
			expectedCalls:    3,
		},
		{
			name:             "max pages limit",
			client:           newMockSensorClient(101, 1), // 101 pages (exceeds MAX_PAGES)
			expectedReadings: 100,                         // Should stop at MAX_PAGES
			expectedCalls:    100,
		},
		{
			name:    "API error",
			client:  &errorMockSensorClient{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readings, err := getSensorReadingsHistory(tt.client, "org123", 5*time.Minute, logger)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedReadings, len(readings))

			// Verify the mock was called the expected number of times
			if mock, ok := tt.client.(*mockSensorClient); ok {
				assert.Equal(t, tt.expectedCalls, mock.callCount)
			}
		})
	}
}

func TestAddSensorReadingFields(t *testing.T) {
	celsius := 25.45
	fahrenheit := 77.81
	humidity := 34
	co2 := 100
	score := 89
	level := 45
	percentage := 91
	draw := 15.9
	voltageLevel := 122.4
	open := true
	present := true

	reading := &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistory{
		Serial: "Q234-ABCD-5678",
		Metric: "temperature",
		Ts:     "2021-10-18T23:54:48.000000Z",
		Network: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryNetwork{
			ID:   "N_24329156",
			Name: "Main Office",
		},
		Temperature: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryTemperature{
			Celsius:    &celsius,
			Fahrenheit: &fahrenheit,
		},
		Humidity: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryHumidity{
			RelativePercentage: &humidity,
		},
		Co2: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryCo2{
			Concentration: &co2,
		},
		IndoorAirQuality: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryIndoorAirQuality{
			Score: &score,
		},
		Noise: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryNoise{
			Ambient: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryNoiseAmbient{
				Level: &level,
			},
		},
		Battery: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryBattery{
			Percentage: &percentage,
		},
		ApparentPower: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryApparentPower{
			Draw: &draw,
		},
		Voltage: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryVoltage{
			Level: &voltageLevel,
		},
		Door: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryDoor{
			Open: &open,
		},
		Water: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryWater{
			Present: &present,
		},
	}

	result := make(mapstr.M)
	addSensorReadingFields(result, reading)

	// Verify basic fields
	assert.Equal(t, "Q234-ABCD-5678", result["sensor.serial"])
	assert.Equal(t, "temperature", result["sensor.metric"])
	assert.Equal(t, "2021-10-18T23:54:48.000000Z", result["@timestamp"])

	// Verify network fields
	assert.Equal(t, "N_24329156", result["sensor.network.id"])
	assert.Equal(t, "Main Office", result["sensor.network.name"])

	// Verify temperature fields
	assert.Equal(t, &celsius, result["sensor.temperature.celsius"])
	assert.Equal(t, &fahrenheit, result["sensor.temperature.fahrenheit"])

	// Verify other sensor readings
	assert.Equal(t, &humidity, result["sensor.humidity.relative_percentage"])
	assert.Equal(t, &co2, result["sensor.co2.concentration"])
	assert.Equal(t, &score, result["sensor.indoor_air_quality.score"])
	assert.Equal(t, &level, result["sensor.noise.ambient.level"])
	assert.Equal(t, &percentage, result["sensor.battery.percentage"])
	assert.Equal(t, &draw, result["sensor.apparent_power.draw"])
	assert.Equal(t, &voltageLevel, result["sensor.voltage.level"])
	assert.Equal(t, &open, result["sensor.door.open"])
	assert.Equal(t, &present, result["sensor.water.present"])
}

func TestAddSensorReadingFields_NilFields(t *testing.T) {
	// Test with minimal data (only required fields)
	reading := &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistory{
		Serial: "Q234-ABCD-5678",
		Metric: "temperature",
		Ts:     "2021-10-18T23:54:48.000000Z",
	}

	result := make(mapstr.M)
	addSensorReadingFields(result, reading)

	// Verify basic fields are present
	assert.Equal(t, "Q234-ABCD-5678", result["sensor.serial"])
	assert.Equal(t, "temperature", result["sensor.metric"])
	assert.Equal(t, "2021-10-18T23:54:48.000000Z", result["@timestamp"])

	// Verify nil fields are not in the map
	_, hasNetwork := result["sensor.network.id"]
	assert.False(t, hasNetwork)

	_, hasTemp := result["sensor.temperature.celsius"]
	assert.False(t, hasTemp)
}

// Mock implementation for sensor readings pagination testing
type mockSensorClient struct {
	totalPages      int
	readingsPerPage int
	callCount       int
}

func newMockSensorClient(totalPages, readingsPerPage int) *mockSensorClient {
	return &mockSensorClient{
		totalPages:      totalPages,
		readingsPerPage: readingsPerPage,
	}
}

func (m *mockSensorClient) GetOrganizationSensorReadingsHistory(organizationID string, params *meraki.GetOrganizationSensorReadingsHistoryQueryParams) (*meraki.ResponseSensorGetOrganizationSensorReadingsHistory, *resty.Response, error) {
	m.callCount++

	readings := make(meraki.ResponseSensorGetOrganizationSensorReadingsHistory, 0, m.readingsPerPage)
	celsius := 25.0
	fahrenheit := 77.0

	for i := 0; i < m.readingsPerPage; i++ {
		serial := fmt.Sprintf("SENSOR-%d-%d", m.callCount, i)
		readings = append(readings, meraki.ResponseItemSensorGetOrganizationSensorReadingsHistory{
			Serial: serial,
			Metric: "temperature",
			Ts:     fmt.Sprintf("2021-10-18T23:%02d:00.000000Z", m.callCount),
			Network: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryNetwork{
				ID:   "N_12345",
				Name: "Test Network",
			},
			Temperature: &meraki.ResponseItemSensorGetOrganizationSensorReadingsHistoryTemperature{
				Celsius:    &celsius,
				Fahrenheit: &fahrenheit,
			},
		})
	}

	resp := &resty.Response{}
	bodyBytes, _ := json.Marshal(readings)
	resp.SetBody(bodyBytes)

	headers := http.Header{}
	if m.callCount < m.totalPages {
		nextSerial := fmt.Sprintf("SENSOR-%d-%d", m.callCount, m.readingsPerPage-1)
		linkHeader := fmt.Sprintf(`<https://api.meraki.com/api/v1/organizations/%s/sensor/readings/history?startingAfter=%s>; rel="next"`, organizationID, nextSerial)
		headers.Set("Link", linkHeader)
	}

	resp.RawResponse = &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
		Header:     headers,
	}

	return &readings, resp, nil
}

// Error mock for testing error handling
type errorMockSensorClient struct{}

func (m *errorMockSensorClient) GetOrganizationSensorReadingsHistory(organizationID string, params *meraki.GetOrganizationSensorReadingsHistoryQueryParams) (*meraki.ResponseSensorGetOrganizationSensorReadingsHistory, *resty.Response, error) {
	resp := &resty.Response{}
	bodyContent := []byte("Internal Server Error")
	resp.SetBody(bodyContent)
	resp.RawResponse = &http.Response{
		StatusCode: 500,
		Body:       io.NopCloser(bytes.NewBuffer(bodyContent)),
	}
	return nil, resp, fmt.Errorf("mock API error")
}
