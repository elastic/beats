// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	meraki "github.com/meraki/dashboard-api-go/v3/sdk"
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
	tests := []struct {
		name     string
		client   NetworkHealthService
		devices  map[Serial]*Device
		wantErr  bool
		validate func(t *testing.T, devices map[Serial]*Device)
	}{
		{
			name:   "successful data retrieval",
			client: &SuccessfulMockNetworkHealthService{},
			devices: map[Serial]*Device{
				"serial-1": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-1",
					},
				},
				"serial-2": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-2",
					},
				},
			},
			validate: func(t *testing.T, devices map[Serial]*Device) {
				require.NotNil(t, devices["serial-1"].wifi0)
				require.Equal(t, 1.0, *devices["serial-1"].wifi0.Utilization80211)
				require.Equal(t, 1.1, *devices["serial-1"].wifi0.UtilizationNon80211)
				require.Equal(t, 1.2, *devices["serial-1"].wifi0.UtilizationTotal)
				require.NotNil(t, devices["serial-2"].wifi1)
				require.Equal(t, 2.0, *devices["serial-2"].wifi1.Utilization80211)
				require.Equal(t, 2.1, *devices["serial-2"].wifi1.UtilizationNon80211)
				require.Equal(t, 2.2, *devices["serial-2"].wifi1.UtilizationTotal)
			},
		},
		{
			name:   "multiple buckets use first entry",
			client: &MultipleBucketsMockNetworkHealthService{},
			devices: map[Serial]*Device{
				"serial-3": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-3",
					},
				},
			},
			validate: func(t *testing.T, devices map[Serial]*Device) {
				require.NotNil(t, devices["serial-3"].wifi0)
				require.Equal(t, 3.0, *devices["serial-3"].wifi0.Utilization80211)
				require.Equal(t, 3.1, *devices["serial-3"].wifi0.UtilizationNon80211)
				require.Equal(t, 3.2, *devices["serial-3"].wifi0.UtilizationTotal)
				require.Nil(t, devices["serial-3"].wifi1)
			},
		},
		{
			name:   "MR 27.0 error skips network",
			client: &MR27ErrorMockNetworkHealthService{},
			devices: map[Serial]*Device{
				"serial-4": {
					details: &meraki.ResponseItemOrganizationsGetOrganizationDevices{
						ProductType: "wireless",
						NetworkID:   "network-4",
					},
				},
			},
			validate: func(t *testing.T, devices map[Serial]*Device) {
				require.Nil(t, devices["serial-4"].wifi0)
				require.Nil(t, devices["serial-4"].wifi1)
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
					details: v.details,
					wifi0:   v.wifi0,
					wifi1:   v.wifi1,
				}
			}

			err := getDeviceChannelUtilization(tt.client, devicesCopy, time.Second)
			if tt.wantErr {
				require.Error(t, err)
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

func (m *SuccessfulMockNetworkHealthService) GetNetworkNetworkHealthChannelUtilization(networkID string, params *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	wifi0utilization80211 := 1.0
	wifi0utilizationNon80211 := 1.1
	wifi0utilizationTotal := 1.2

	wifi1utilization80211 := 2.0
	wifi1utilizationNon80211 := 2.1
	wifi1utilizationTotal := 2.2

	return &meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization{
		meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilization{
			Serial: "serial-1",
			Wifi0: &[]meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi0{
				{
					Utilization80211:    &wifi0utilization80211,
					UtilizationNon80211: &wifi0utilizationNon80211,
					UtilizationTotal:    &wifi0utilizationTotal,
				},
			},
			Wifi1: &[]meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi1{
				{
					Utilization80211:    &wifi1utilization80211,
					UtilizationNon80211: &wifi1utilizationNon80211,
					UtilizationTotal:    &wifi1utilizationTotal,
				},
			},
		},
		meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilization{
			Serial: "serial-2",
			Wifi0: &[]meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi0{
				{
					Utilization80211:    &wifi0utilization80211,
					UtilizationNon80211: &wifi0utilizationNon80211,
					UtilizationTotal:    &wifi0utilizationTotal,
				},
			},
			Wifi1: &[]meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi1{
				{
					Utilization80211:    &wifi1utilization80211,
					UtilizationNon80211: &wifi1utilizationNon80211,
					UtilizationTotal:    &wifi1utilizationTotal,
				},
			},
		},
	}, &resty.Response{}, nil
}

// MultipleBucketsMockNetworkHealthService returns multiple utilization buckets
type MultipleBucketsMockNetworkHealthService struct{}

func (m *MultipleBucketsMockNetworkHealthService) GetNetworkNetworkHealthChannelUtilization(networkID string, params *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	wifi0util_80211 := 3.0
	wifi0util_non80211 := 3.1
	wifi0util_total := 3.2

	return &meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization{
		meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilization{
			Serial: "serial-3",
			Wifi0: &[]meraki.ResponseItemNetworksGetNetworkNetworkHealthChannelUtilizationWifi0{
				{ // First bucket will be used
					Utilization80211:    &wifi0util_80211,
					UtilizationNon80211: &wifi0util_non80211,
					UtilizationTotal:    &wifi0util_total,
				},
				{ // Second bucket will be ignored
					Utilization80211:    &wifi0util_80211,
					UtilizationNon80211: &wifi0util_non80211,
					UtilizationTotal:    &wifi0util_total,
				},
			},
		},
	}, &resty.Response{}, nil
}

// MR27ErrorMockNetworkHealthService simulates the MR 27.0 version error
type MR27ErrorMockNetworkHealthService struct{}

func (m *MR27ErrorMockNetworkHealthService) GetNetworkNetworkHealthChannelUtilization(networkID string, params *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	r := &resty.Response{}
	bodyContent := []byte("This endpoint is only available for networks on MR 27.0 or above.")
	r.SetBody(bodyContent)
	r.RawResponse = &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(bodyContent)),
	}
	return nil, r, fmt.Errorf("MR 27.0 error")
}

// GenericErrorMockNetworkHealthService simulates generic errors
type GenericErrorMockNetworkHealthService struct{}

func (m *GenericErrorMockNetworkHealthService) GetNetworkNetworkHealthChannelUtilization(networkID string, params *meraki.GetNetworkNetworkHealthChannelUtilizationQueryParams) (*meraki.ResponseNetworksGetNetworkNetworkHealthChannelUtilization, *resty.Response, error) {
	r := &resty.Response{}
	bodyContent := []byte("Internal Server Error")
	r.SetBody(bodyContent)
	r.RawResponse = &http.Response{
		Body: io.NopCloser(bytes.NewBuffer(bodyContent)),
	}
	return nil, r, fmt.Errorf("mock API error")
}
