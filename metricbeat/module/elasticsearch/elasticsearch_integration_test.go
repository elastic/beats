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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	"github.com/elastic/beats/metricbeat/helper/elastic"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/elasticsearch"
	_ "github.com/elastic/beats/metricbeat/module/elasticsearch/ccr"
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
	"ccr",
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

	version, err := getElasticsearchVersion(host)
	if err != nil {
		t.Fatal("getting elasticsearch version", err)
	}

	err = enableTrialLicense(host, version)
	assert.NoError(t, err)

	err = createMLJob(host, version)
	assert.NoError(t, err)

	err = createCCRStats(host)
	assert.NoError(t, err)

	for _, metricSet := range metricSets {
		checkSkip(t, metricSet, version)
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2Error(t, getConfig(metricSet))
			events, errs := mbtest.ReportingFetchV2Error(f)

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

	host := net.JoinHostPort(getEnvHost(), getEnvPort())

	version, err := getElasticsearchVersion(host)
	if err != nil {
		t.Fatal("getting elasticsearch version", err)
	}

	for _, metricSet := range metricSets {
		checkSkip(t, metricSet, version)
		t.Run(metricSet, func(t *testing.T) {
			f := mbtest.NewReportingMetricSetV2Error(t, getConfig(metricSet))
			err := mbtest.WriteEventsReporterV2Error(f, t, metricSet)
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
		"module":                     elasticsearch.ModuleName,
		"metricsets":                 []string{metricset},
		"hosts":                      []string{getEnvHost() + ":" + getEnvPort()},
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
func enableTrialLicense(host string, version *common.Version) error {
	client := &http.Client{}

	var enableXPackURL string
	if version.Major < 7 {
		enableXPackURL = "/_xpack/license/start_trial?acknowledge=true"
	} else {
		enableXPackURL = "/_license/start_trial?acknowledge=true"
	}

	req, err := http.NewRequest("POST", "http://"+host+enableXPackURL, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("could not enable trial license, response = %v", string(body))
	}

	return nil
}

func createMLJob(host string, version *common.Version) error {

	mlJob, err := ioutil.ReadFile("ml_job/_meta/test/test_job.json")
	if err != nil {
		return err
	}

	var jobURL string
	if version.Major < 7 {
		jobURL = "/_xpack/ml/anomaly_detectors/total-requests"
	} else {
		jobURL = "/_ml/anomaly_detectors/total-requests"
	}

	if checkExists("http://" + host + jobURL) {
		return nil
	}

	body, resp, err := httpPutJSON(host, jobURL, mlJob)
	if err != nil {
		return errors.Wrap(err, "error doing PUT request when creating ML job")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error loading ml job %d: %s, %s", resp.StatusCode, resp.Status, string(body))
	}

	return nil
}

func createCCRStats(host string) error {
	err := setupCCRRemote(host)
	if err != nil {
		return errors.Wrap(err, "error setup CCR remote settings")
	}

	err = createCCRLeaderIndex(host)
	if err != nil {
		return errors.Wrap(err, "error creating CCR leader index")
	}

	err = createCCRFollowerIndex(host)
	if err != nil {
		return errors.Wrap(err, "error creating CCR follower index")
	}

	// Give ES sufficient time to do the replication and produce stats
	checkCCRStats := func() (bool, error) {
		return checkCCRStatsExists(host)
	}

	exists, err := waitForSuccess(checkCCRStats, 300, 5)
	if err != nil {
		return errors.Wrap(err, "error checking if CCR stats exist")
	}

	if !exists {
		return fmt.Errorf("expected to find CCR stats but not found")
	}

	return nil
}

func checkCCRStatsExists(host string) (bool, error) {
	resp, err := http.Get("http://" + host + "/_ccr/stats")
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var data struct {
		FollowStats struct {
			Indices []map[string]interface{} `json:"indices"`
		} `json:"follow_stats"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, err
	}

	return len(data.FollowStats.Indices) > 0, nil
}

func setupCCRRemote(host string) error {
	remoteSettings, err := ioutil.ReadFile("ccr/_meta/test/test_remote_settings.json")
	if err != nil {
		return err
	}

	settingsURL := "/_cluster/settings"
	_, _, err = httpPutJSON(host, settingsURL, remoteSettings)
	return err
}

func createCCRLeaderIndex(host string) error {
	leaderIndex, err := ioutil.ReadFile("ccr/_meta/test/test_leader_index.json")
	if err != nil {
		return err
	}

	indexURL := "/pied_piper"
	_, _, err = httpPutJSON(host, indexURL, leaderIndex)
	return err
}

func createCCRFollowerIndex(host string) error {
	followerIndex, err := ioutil.ReadFile("ccr/_meta/test/test_follower_index.json")
	if err != nil {
		return err
	}

	followURL := "/rats/_ccr/follow"
	_, _, err = httpPutJSON(host, followURL, followerIndex)
	return err
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

func checkSkip(t *testing.T, metricset string, version *common.Version) {
	if metricset != "ccr" {
		return
	}

	isCCRStatsAPIAvailable := elastic.IsFeatureAvailable(version, elasticsearch.CCRStatsAPIAvailableVersion)

	if !isCCRStatsAPIAvailable {
		t.Skip("elasticsearch CCR stats API is not available until " + elasticsearch.CCRStatsAPIAvailableVersion.String())
	}
}

func getElasticsearchVersion(elasticsearchHostPort string) (*common.Version, error) {
	resp, err := http.Get("http://" + elasticsearchHostPort + "/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data common.MapStr
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	version, err := data.GetValue("version.number")
	if err != nil {
		return nil, err
	}

	return common.NewVersion(version.(string))
}

func httpPutJSON(host, path string, body []byte) ([]byte, *http.Response, error) {
	req, err := http.NewRequest("PUT", "http://"+host+path, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	return body, resp, nil
}

type checkSuccessFunction func() (bool, error)

func waitForSuccess(f checkSuccessFunction, retryIntervalMs time.Duration, numAttempts int) (bool, error) {
	for numAttempts > 0 {
		success, err := f()
		if err != nil {
			return false, err
		}

		if success {
			return success, nil
		}

		time.Sleep(retryIntervalMs * time.Millisecond)
		numAttempts--
	}

	return false, nil
}
