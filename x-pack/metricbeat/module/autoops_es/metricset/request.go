// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metricset

import (
	"github.com/elastic/beats/v9/metricbeat/module/elasticsearch"
	"github.com/elastic/beats/v9/x-pack/metricbeat/module/autoops_es/utils"
)

const (
	CLOUD_CONNECTED_MODE_API_KEY_NAME    = "ELASTIC_CLOUD_CONNECTED_MODE_API_KEY"
	CLOUD_CONNECTED_MODE_API_URL_NAME    = "ELASTIC_CLOUD_CONNECTED_MODE_API_URL"
	DEFAULT_CLOUD_CONNECTED_MODE_API_URL = "https://api.elastic-cloud.com"
)

// getCloudConnectedModeApiKey returns the API key for the Cloud Connected Mode API, which can be overridden
// by setting the environment variable `ELASTIC_CLOUD_CONNECTED_MODE_API_KEY`.
func getCloudConnectedModeApiKey(m *elasticsearch.MetricSet) (string, error) {
	if apiKey := utils.GetStrenv(CLOUD_CONNECTED_MODE_API_KEY_NAME, ""); apiKey != "" {
		return apiKey, nil
	}

	config := struct {
		ApiKey string `config:"ccm.api_key"`
	}{
		ApiKey: "",
	}

	if err := m.Module().UnpackConfig(&config); err != nil {
		return "", err
	}

	return config.ApiKey, nil
}

// getCloudConnectedModeAPIURL returns the URL for the Cloud Connected Mode API, which can be overridden
// by setting the environment variable `ELASTIC_CLOUD_CONNECTED_MODE_API_URL`.
func getCloudConnectedModeAPIURL() string {
	return utils.GetStrenv(CLOUD_CONNECTED_MODE_API_URL_NAME, DEFAULT_CLOUD_CONNECTED_MODE_API_URL)
}
