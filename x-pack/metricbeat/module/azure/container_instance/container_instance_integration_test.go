// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package container_instance

import (
	"errors"
	"os"
	"testing"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestData(t *testing.T) {
	config, err := getConfig()
	if err != nil {
		t.Skip("Skipping TestData: " + err.Error())
	}

	metricSet := mbtest.NewFetcher(t, config)
	metricSet.WriteEvents(t, "/")
}

func getConfig() (map[string]interface{}, error) {
	clientId, ok := os.LookupEnv("AZURE_CLIENT_ID")
	if !ok {
		return nil, errors.New("missing AZURE_CLIENT_ID key")
	}
	clientSecret, ok := os.LookupEnv("AZURE_CLIENT_SECRET")
	if !ok {
		return nil, errors.New("missing AZURE_CLIENT_SECRET key")
	}
	tenantId, ok := os.LookupEnv("AZURE_TENANT_ID")
	if !ok {
		return nil, errors.New("missing AZURE_TENANT_ID key")
	}
	subscriptionId, ok := os.LookupEnv("AZURE_SUBSCRIPTION_ID")
	if !ok {
		return nil, errors.New("missing AZURE_SUBSCRIPTION_ID key")
	}
	config := map[string]interface{}{
		"module":          "azure",
		"metricsets":      []string{"container_instance"},
		"client_id":       clientId,
		"client_secret":   clientSecret,
		"tenant_id":       tenantId,
		"subscription_id": subscriptionId,
	}
	return config, nil
}
