// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panw

import (
	"encoding/xml"
	"fmt"
)

const (
	// SystemInfoQuery is the API query to retrieve system information including hostname
	SystemInfoQuery = "<show><system><info></info></system></show>"
)

// SystemInfoResponse represents the XML response from the system info API call
type SystemInfoResponse struct {
	XMLName xml.Name         `xml:"response"`
	Status  string           `xml:"status,attr"`
	Result  SystemInfoResult `xml:"result"`
}

// SystemInfoResult contains the system information
type SystemInfoResult struct {
	System SystemInfoData `xml:"system"`
}

// SystemInfoData contains the actual system data fields
type SystemInfoData struct {
	Hostname    string `xml:"hostname"`
	IPAddress   string `xml:"ip-address"`
	Netmask     string `xml:"netmask"`
	MACAddress  string `xml:"mac-address"`
	DeviceName  string `xml:"devicename"`
	Family      string `xml:"family"`
	Model       string `xml:"model"`
	Serial      string `xml:"serial"`
	SWVersion   string `xml:"sw-version"`
	AppVersion  string `xml:"app-version"`
	AVVersion   string `xml:"av-version"`
	Uptime      string `xml:"uptime"`
	MultiVsys   string `xml:"multi-vsys"`
	CloudMode   string `xml:"cloud-mode"`
	VPNDisabled string `xml:"vpn-disable-mode"`
}

// GetSystemInfo fetches the system information from the firewall
func GetSystemInfo(client PanwClient) (*SystemInfoData, error) {
	output, err := client.Op(SystemInfoQuery, Vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute system info query: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("empty response from PanOS for system info query")
	}

	var response SystemInfoResponse
	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal system info XML response: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("system info query returned non-success status: %s", response.Status)
	}

	return &response.Result.System, nil
}

// GetHostname fetches just the hostname from the firewall
func GetHostname(client PanwClient) (string, error) {
	sysInfo, err := GetSystemInfo(client)
	if err != nil {
		return "", err
	}

	if sysInfo.Hostname == "" {
		return "", fmt.Errorf("hostname is empty in system info response")
	}

	return sysInfo.Hostname, nil
}
