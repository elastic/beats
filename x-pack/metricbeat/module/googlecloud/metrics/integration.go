// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package metrics

import (
	"os"
	"testing"
)

// GetConfigForTest function gets aws credentials for integration tests.
// GCP_REGION, GCP_PROJECT_ID and GCP_CREDENTIALS_FILE_PATH are required.
func GetConfigForTest(t *testing.T, metricSetName string) map[string]interface{} {
	t.Helper()
	region, okRegion := os.LookupEnv("GCP_REGION")
	projectID, okProjectID := os.LookupEnv("GCP_PROJECT_ID")
	credentialsFilePath, okCredentialsFilePath := os.LookupEnv("GCP_CREDENTIALS_FILE_PATH")

	config := map[string]interface{}{}
	if !okProjectID || projectID == "" {
		t.Fatal("$GCP_PROJECT_ID not set or set to empty")
	} else if !okCredentialsFilePath || credentialsFilePath == "" {
		t.Fatal("$GCP_CREDENTIALS_FILE_PATH not set or set to empty")
	} else {
		config = map[string]interface{}{
			"module":                "googlecloud",
			"period":                "1m",
			"metricsets":            []string{metricSetName},
			"project_id":            projectID,
			"credentials_file_path": credentialsFilePath,
		}

		if okRegion {
			config["region"] = region
		}

		if metricSetName == "metrics" {
			config["metrics"] = []metricsConfig{
				{
					ServiceName: "compute",
					Aligner:     "ALIGN_NONE",
					MetricTypes: []string{"instance/uptime"},
				},
			}
		}
	}
	return config
}
