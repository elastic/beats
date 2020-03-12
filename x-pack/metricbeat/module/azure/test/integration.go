// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package test

import (
	"errors"
	"os"
)

// GetConfig function gets azure credentials for integration tests.
func GetConfig(metricSetName string) (map[string]interface{}, error) {
	clientId, ok := os.LookupEnv("AZURE_CLIENT_ID")
	if !ok {
		return nil, errors.New("could not find var AZURE_CLIENT_ID")
	}
	clientSecret, ok := os.LookupEnv("AZURE_CLIENT_SECRET")
	if !ok {
		return nil, errors.New("could not find var AZURE_CLIENT_SECRET")
	}
	tenantId, ok := os.LookupEnv("AZURE_TENANT_ID")
	if !ok {
		return nil, errors.New("could not find var AZURE_TENANT_ID")
	}
	subId, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		return nil, errors.New("could not find var AZURE_SUBSCRIPTION_ID")
	}
	return map[string]interface{}{
		"module":                "azure",
		"period":                "300s",
		"refresh_list_interval": "600s",
		"metricsets":            []string{metricSetName},
		"client_id":             clientId,
		"client_secret":         clientSecret,
		"tenant_id":             tenantId,
		"subscription_id":       subId,
	}, nil
}
