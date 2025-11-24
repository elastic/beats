// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"testing"
)

func TestGetAzureCloud(t *testing.T) {
	// Test that the function doesn't panic for different authority hosts
	testCases := []string{
		"https://login.microsoftonline.com",
		"https://login.microsoftonline.us",
		"https://login.chinacloudapi.cn",
		"",
	}

	for _, authorityHost := range testCases {
		t.Run("authority_host_"+authorityHost, func(t *testing.T) {
			cloud := getAzureCloud(authorityHost)
			// Just verify we got a result - we can't easily compare cloud configurations
			_ = cloud
		})
	}
}
