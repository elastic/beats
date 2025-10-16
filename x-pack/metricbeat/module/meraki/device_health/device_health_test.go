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
)

func TestGetDeviceChannelUtilization(t *testing.T) {
	orgs := []string{"123"}
	tests := []struct {
		name     string
		client   DeviceService
		devices  map[Serial]*Device
		wantErr  bool
		validate func(t *testing.T, devices map[Serial]*Device)
	}{
		{
			name:   "successful data retrieval",
			client: &SuccessfulMockNetworkHealthService{},
			devices: map[Serial]*Device{
				"ABC123": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-1",
					},
				},
			},
			validate: func(t *testing.T, devices map[Serial]*Device) {
				assert.NotNil(t, devices["ABC123"].bandUtilization)

				band1, ok := devices["ABC123"].bandUtilization["2.4"]
				assert.NotNil(t, band1)
				assert.True(t, ok)
				assert.Equal(t, 45.0, *band1.Wifi.Percentage)
				assert.Equal(t, 10.0, *band1.NonWifi.Percentage)
				assert.Equal(t, 55.0, *band1.Total.Percentage)

				band2, ok := devices["ABC123"].bandUtilization["5"]
				assert.NotNil(t, band2)
				assert.True(t, ok)
				assert.Equal(t, 10.0, *band2.Wifi.Percentage)
				assert.Equal(t, 45.0, *band2.NonWifi.Percentage)
				assert.Equal(t, 55.0, *band2.Total.Percentage)
			},
		},
		{
			name:   "other errors propagate",
			client: &GenericErrorMockNetworkHealthService{},
			devices: map[Serial]*Device{
				"serial-5": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-5",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devicesCopy := make(map[Serial]*Device, len(tt.devices))
			for k, v := range tt.devices {
				devicesCopy[k] = &Device{
					details:         v.details,
					bandUtilization: v.bandUtilization,
				}
			}

			err := getDeviceChannelUtilization(tt.client, devicesCopy, time.Second, orgs)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, devicesCopy)
			}
		})
	}
}

// SuccessfulMockNetworkHealthService returns valid utilization data
type SuccessfulMockNetworkHealthService struct{}

func (m *SuccessfulMockNetworkHealthService) GetOrganizationWirelessDevicesChannelUtilizationByDevice(organizationID string, getOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams *meraki.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams) (*resty.Response, error) {
	percentage45 := 45.0
	percentage10 := 10.0
	percentage55 := 55.0

	dummyData := &meraki.ResponseOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDevice{
		meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDevice{
			ByBand: &[]meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBand{
				{
					Band: "2.4",
					Wifi: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandWifi{
						Percentage: &percentage45,
					},
					NonWifi: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandNonWifi{
						Percentage: &percentage10,
					},
					Total: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandTotal{
						Percentage: &percentage55,
					},
				},
				{
					Band: "5",
					Wifi: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandWifi{
						Percentage: &percentage10,
					},
					NonWifi: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandNonWifi{
						Percentage: &percentage45,
					},
					Total: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceByBandTotal{
						Percentage: &percentage55,
					},
				},
			},
			Mac: "00:11:22:33:44:55",
			Network: &meraki.ResponseItemOrganizationsGetOrganizationWirelessDevicesChannelUtilizationByDeviceNetwork{
				ID: "network-1",
			},
			Serial: "ABC123",
		},
	}

	r := &resty.Response{}

	bodyBytes, err := json.Marshal(dummyData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dummy data: %w", err)
	}

	r.SetBody(bodyBytes)
	r.RawResponse = &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(bodyBytes)),
	}
	r.Request = &resty.Request{
		Result: dummyData,
	}

	return r, nil
}

// GenericErrorMockNetworkHealthService simulates generic errors
type GenericErrorMockNetworkHealthService struct{}

func (m *GenericErrorMockNetworkHealthService) GetOrganizationWirelessDevicesChannelUtilizationByDevice(organizationID string, getOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams *meraki.GetOrganizationWirelessDevicesChannelUtilizationByDeviceQueryParams) (*resty.Response, error) {
	r := &resty.Response{}
	bodyContent := []byte("Internal Server Error")
	r.SetBody(bodyContent)
	r.RawResponse = &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(bodyContent)),
	}
	return r, fmt.Errorf("mock API error")
}

func TestGetDevices_Pagination(t *testing.T) {
	logger := logp.NewLogger("test")

	tests := []struct {
		name            string
		client          OrganizationsClient
		expectedDevices int
		expectedCalls   int
		wantErr         bool
	}{
		{
			name:            "single page",
			client:          newMockOrganizationsClient(1, 2), // 1 page, 2 devices
			expectedDevices: 2,
			expectedCalls:   1,
		},
		{
			name:            "multiple pages",
			client:          newMockOrganizationsClient(3, 2), // 3 pages, 2 devices per page
			expectedDevices: 6,
			expectedCalls:   3,
		},
		{
			name:            "max pages limit",
			client:          newMockOrganizationsClient(101, 1), // 101 pages (exceeds MAX_PAGES)
			expectedDevices: 100,                                // Should stop at MAX_PAGES
			expectedCalls:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices, err := getDevices(tt.client, "org123", logger)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedDevices, len(devices))

			// Verify the mock was called the expected number of times
			if mock, ok := tt.client.(*mockOrganizationsClient); ok {
				assert.Equal(t, tt.expectedCalls, mock.callCount)
			}
		})
	}
}

// Mock implementation for pagination testing
type mockOrganizationsClient struct {
	totalPages     int
	devicesPerPage int
	callCount      int
}

func newMockOrganizationsClient(totalPages, devicesPerPage int) *mockOrganizationsClient {
	return &mockOrganizationsClient{
		totalPages:     totalPages,
		devicesPerPage: devicesPerPage,
	}
}

func (m *mockOrganizationsClient) GetOrganizationDevices(organizationID string, params *meraki.GetOrganizationDevicesQueryParams) (*meraki.ResponseOrganizationsGetOrganizationDevices, *resty.Response, error) {
	m.callCount++

	devices := make(meraki.ResponseOrganizationsGetOrganizationDevices, 0, m.devicesPerPage)
	for i := 0; i < m.devicesPerPage; i++ {
		serial := fmt.Sprintf("SERIAL-%d-%d", m.callCount, i)
		devices = append(devices, meraki.ResponseItemOrganizationsGetOrganizationDevices{
			Serial: serial,
			Name:   fmt.Sprintf("Device %s", serial),
		})
	}

	resp := &resty.Response{}
	bodyBytes, _ := json.Marshal(devices)
	resp.SetBody(bodyBytes)

	headers := http.Header{}
	if m.callCount < m.totalPages {
		nextSerial := fmt.Sprintf("SERIAL-%d-%d", m.callCount, m.devicesPerPage-1)
		linkHeader := fmt.Sprintf(`<https://api.meraki.com/api/v1/organizations/%s/devices?startingAfter=%s>; rel="next"`, organizationID, nextSerial)
		headers.Set("Link", linkHeader)
	}

	resp.RawResponse = &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBuffer(bodyBytes)),
		Header:     headers,
	}

	return &devices, resp, nil
}
