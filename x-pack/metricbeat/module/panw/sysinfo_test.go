// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panw

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockClient implements PanwClient for testing
type MockClient struct {
	response []byte
	err      error
}

func (m *MockClient) Op(req interface{}, vsys string, extras, ans interface{}) ([]byte, error) {
	return m.response, m.err
}

func TestGetSystemInfo(t *testing.T) {
	// Read test data from file
	testDataPath := filepath.Join("_meta", "testdata", "system_info.xml")
	xmlData, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	client := &MockClient{response: xmlData}

	sysInfo, err := GetSystemInfo(client)
	require.NoError(t, err)
	require.NotNil(t, sysInfo)

	assert.Equal(t, "PA-FIREWALL-01", sysInfo.Hostname)
	assert.Equal(t, "192.168.1.1", sysInfo.IPAddress)
	assert.Equal(t, "00:11:22:33:44:55", sysInfo.MACAddress)
	assert.Equal(t, "PA-FIREWALL-01", sysInfo.DeviceName)
	assert.Equal(t, "400", sysInfo.Family)
	assert.Equal(t, "PA-440", sysInfo.Model)
	assert.Equal(t, "012345678901", sysInfo.Serial)
	assert.Equal(t, "10.1.12", sysInfo.SWVersion)
	assert.Equal(t, "9061-9857", sysInfo.AppVersion)
	assert.Equal(t, "5447-5974", sysInfo.AVVersion)
	assert.Equal(t, "off", sysInfo.MultiVsys)
	assert.Equal(t, "non-cloud", sysInfo.CloudMode)
	assert.Equal(t, "off", sysInfo.VPNDisabled)
}

func TestGetHostname(t *testing.T) {
	// Read test data from file
	testDataPath := filepath.Join("_meta", "testdata", "system_info.xml")
	xmlData, err := os.ReadFile(testDataPath)
	require.NoError(t, err, "Failed to read test data file")

	client := &MockClient{response: xmlData}

	hostname, err := GetHostname(client)
	require.NoError(t, err)
	assert.Equal(t, "PA-FIREWALL-01", hostname)
}

func TestGetSystemInfoEmptyResponse(t *testing.T) {
	client := &MockClient{response: []byte{}}

	sysInfo, err := GetSystemInfo(client)
	assert.Error(t, err)
	assert.Nil(t, sysInfo)
	assert.Contains(t, err.Error(), "empty response")
}

func TestGetSystemInfoInvalidXML(t *testing.T) {
	client := &MockClient{response: []byte("not valid xml")}

	sysInfo, err := GetSystemInfo(client)
	assert.Error(t, err)
	assert.Nil(t, sysInfo)
	assert.Contains(t, err.Error(), "failed to unmarshal")
}

func TestGetSystemInfoFailedStatus(t *testing.T) {
	xmlData := []byte(`<response status="error"><result><msg>Authentication failed</msg></result></response>`)
	client := &MockClient{response: xmlData}

	sysInfo, err := GetSystemInfo(client)
	assert.Error(t, err)
	assert.Nil(t, sysInfo)
	assert.Contains(t, err.Error(), "non-success status")
}

func TestGetHostnameEmptyHostname(t *testing.T) {
	xmlData := []byte(`<response status="success"><result><system><hostname></hostname></system></result></response>`)
	client := &MockClient{response: xmlData}

	hostname, err := GetHostname(client)
	assert.Error(t, err)
	assert.Empty(t, hostname)
	assert.Contains(t, err.Error(), "hostname is empty")
}

func TestMakeRootFieldsWithHostname(t *testing.T) {
	hostIP := "192.168.1.1"
	hostname := "test-firewall"

	rootFields := MakeRootFields(hostIP, hostname)

	assert.Equal(t, hostIP, rootFields["observer.ip"])
	assert.Equal(t, hostname, rootFields["observer.hostname"])
	assert.Equal(t, hostIP, rootFields["host.ip"])
	assert.Equal(t, "Palo Alto", rootFields["observer.vendor"])
	assert.Equal(t, "firewall", rootFields["observer.type"])
}

func TestMakeRootFieldsEmptyHostname(t *testing.T) {
	hostIP := "192.168.1.1"
	hostname := ""

	rootFields := MakeRootFields(hostIP, hostname)

	assert.Equal(t, hostIP, rootFields["observer.ip"])
	assert.Equal(t, "", rootFields["observer.hostname"])
	assert.Equal(t, hostIP, rootFields["host.ip"])
}
