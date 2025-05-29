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
)

func TestIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
	}{
		{
			name:     "Nil pointer",
			input:    (*int)(nil),
			expected: true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: true,
		},
		{
			name:     "Non-empty string",
			input:    "test",
			expected: false,
		},
		{
			name:     "Empty slice",
			input:    []string{},
			expected: true,
		},
		{
			name:     "Regular value",
			input:    float64(1.2),
			expected: false,
		},
		{
			name:     "Pointer to int",
			input:    func() *int { i := 42; return &i }(),
			expected: false,
		},
		{
			name:     "Pointer to bool",
			input:    func() *bool { b := false; return &b }(),
			expected: false,
		},
		{
			name:     "Boolean false",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEmpty(tt.input)
			if result != tt.expected {
				t.Errorf("isEmpty(%v) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

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
