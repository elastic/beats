// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

func TestFilterSwitchportsByStatus(t *testing.T) {
	tests := []struct {
		name             string
		switchports      []*switchport
		statusesToReport []string
		expectedCount    int
		expectedPortIDs  []string
	}{
		{
			name: "filter only connected ports",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "Connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "2", Status: "Disconnected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "3"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "3", Status: "connected"},
				},
			},
			statusesToReport: []string{"connected"},
			expectedCount:    2,
			expectedPortIDs:  []string{"1", "3"},
		},
		{
			name: "filter only disconnected ports",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "Connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "2", Status: "Disconnected"},
				},
			},
			statusesToReport: []string{"disconnected"},
			expectedCount:    1,
			expectedPortIDs:  []string{"2"},
		},
		{
			name: "filter multiple statuses",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "Connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "2", Status: "Disconnected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "3"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "3", Status: "Disabled"},
				},
			},
			statusesToReport: []string{"connected", "disconnected"},
			expectedCount:    2,
			expectedPortIDs:  []string{"1", "2"},
		},
		{
			name: "skip ports with nil portStatus",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "Connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: nil,
				},
			},
			statusesToReport: []string{"connected"},
			expectedCount:    1,
			expectedPortIDs:  []string{"1"},
		},
		{
			name: "case insensitive matching - port status varies",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "CONNECTED"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "2", Status: "Connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "3"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "3", Status: "connected"},
				},
			},
			statusesToReport: []string{"connected"},
			expectedCount:    3,
			expectedPortIDs:  []string{"1", "2", "3"},
		},
		{
			name: "case insensitive matching - config status varies",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "connected"},
				},
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "2"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "2", Status: "disconnected"},
				},
			},
			statusesToReport: []string{"CONNECTED", "Disconnected"},
			expectedCount:    2,
			expectedPortIDs:  []string{"1", "2"},
		},
		{
			name:             "empty switchports",
			switchports:      []*switchport{},
			statusesToReport: []string{"connected"},
			expectedCount:    0,
			expectedPortIDs:  []string{},
		},
		{
			name:             "nil switchports",
			switchports:      nil,
			statusesToReport: []string{"connected"},
			expectedCount:    0,
			expectedPortIDs:  []string{},
		},
		{
			name: "no matching statuses",
			switchports: []*switchport{
				{
					port:       &sdk.ResponseItemSwitchGetOrganizationSwitchPortsBySwitchPorts{PortID: "1"},
					portStatus: &sdk.ResponseItemSwitchGetDeviceSwitchPortsStatuses{PortID: "1", Status: "Connected"},
				},
			},
			statusesToReport: []string{"disabled"},
			expectedCount:    0,
			expectedPortIDs:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterSwitchportsByStatus(tt.switchports, tt.statusesToReport)

			assert.Equal(t, tt.expectedCount, len(result), "unexpected number of filtered switchports")

			actualPortIDs := make([]string, len(result))
			for i, sp := range result {
				actualPortIDs[i] = sp.port.PortID
			}
			assert.ElementsMatch(t, tt.expectedPortIDs, actualPortIDs, "unexpected port IDs in filtered result")
		})
	}
}
