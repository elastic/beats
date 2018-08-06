// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// +build integration

package elasticsearch_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"bytes"

	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/cluster_stats"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/index"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/index_recovery"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/index_summary"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/ml_job"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/node"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/node_stats"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/shard"
)

var metricSets = []string{
	"cluster_stats",
	"index",
	"index_recovery",
	"index_summary",
	"ml_job",
	"node",
	"node_stats",
	"shard",
}

func TestFetch(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	host := net.JoinHostPort(getEnvHost(), getEnvPort())
	err := createIndex(host)
	assert.NoError(t, err)

	err = enableTrialLicense(host)
	assert.NoError(t, err)

	err = createMLJob(host)
	assert.NoError(t, err)

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2(t, getConfig(metricSet))
			events, errs := mbtest.ReportingFetchV2(f)

			assert.Empty(t, errs)
			if !assert.NotEmpty(t, events) {
				t.FailNow()
			}
			t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
				events[0].BeatEvent("elasticsearch", metricSet).Fields.StringToPrint())
		})
	}
}

func TestData(t *testing.T) {
	compose.EnsureUp(t, "elasticsearch")

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2(t, getConfig(metricSet))
			err := mbtest.WriteEventsReporterV2(f, t, metricSet)
			if err != nil {
				t.Fatal("write", err)
			}
		})
	}
}

// GetEnvHost returns host for Elasticsearch
func getEnvHost() string {
	host := os.Getenv("ES_HOST")

	if len(host) == 0 {
		host = "127.0.0.1"
	}
	return host
}

// GetEnvPort returns port for Elasticsearch
func getEnvPort() string {
	port := os.Getenv("ES_PORT")

	if len(port) == 0 {
		port = "9200"
	}
	return port
}

// GetConfig returns config for elasticsearch module
func getConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "elasticsearch",
		"metricsets": []string{metricset},
		"hosts":      []string{getEnvHost() + ":" + getEnvPort()},
		"index_recovery.active_only": false,
	}
}

// createIndex creates and elasticsearch index in case it does not exit yet
func createIndex(host string) error {
	client := &http.Client{}

	if checkExists("http://" + host + "/testindex") {
		return nil
	}

	req, err := http.NewRequest("PUT", "http://"+host+"/testindex", nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// createIndex creates and elasticsearch index in case it does not exit yet
func enableTrialLicense(host string) error {
	client := &http.Client{}

	enableXPackURL := "/_xpack/license/start_trial?acknowledge=true"

	req, err := http.NewRequest("POST", "http://"+host+enableXPackURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func createMLJob(host string) error {

	mlJob, err := ioutil.ReadFile("ml_job/_meta/test/test_job.json")
	if err != nil {
		return err
	}

	client := &http.Client{}

	jobURL := "/_xpack/ml/anomaly_detectors/total-requests"

	if checkExists("http://" + host + jobURL) {
		return nil
	}

	req, err := http.NewRequest("PUT", "http://"+host+jobURL, bytes.NewReader(mlJob))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error loading ml job %d: %s, %s", resp.StatusCode, resp.Status, body)
	}

	return nil
}

func checkExists(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}
	resp.Body.Close()

	// Entry exists
	if resp.StatusCode == 200 {
		return true
	}
	return false
}
