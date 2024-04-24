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

//go:build integration

package elasticsearch_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	"github.com/elastic/beats/v7/metricbeat/helper/elastic"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/elasticsearch"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/ccr"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/cluster_stats"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/enrich"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index_recovery"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/index_summary"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/ml_job"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/node"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/node_stats"
	_ "github.com/elastic/beats/v7/metricbeat/module/elasticsearch/shard"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

var metricSets = []string{
	"ccr",
	"cluster_stats",
	"enrich",
	"index",
	"index_recovery",
	"index_summary",
	"ml_job",
	"node",
	"node_stats",
	"shard",
}

func TestFetch(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "elasticsearch")
	host := service.Host()

	version, err := getElasticsearchVersion(host)
	require.NoError(t, err)

	setupTest(t, host, version)

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			checkSkip(t, metricSet, version)
			f := mbtest.NewReportingMetricSetV2Error(t, getConfigForMetricset(metricSet, host))
			events, errs := mbtest.ReportingFetchV2Error(f)

			require.Empty(t, errs)
			require.NotEmpty(t, events)

			t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
				events[0].BeatEvent("elasticsearch", metricSet).Fields.StringToPrint())
		})
	}
}

func TestData(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "elasticsearch")
	host := service.Host()

	version, err := getElasticsearchVersion(host)
	require.NoError(t, err)

	for _, metricSet := range metricSets {
		t.Run(metricSet, func(t *testing.T) {
			checkSkip(t, metricSet, version)
			f := mbtest.NewReportingMetricSetV2Error(t, getConfigForMetricset(metricSet, host))
			err := mbtest.WriteEventsReporterV2Error(f, t, metricSet)
			require.NoError(t, err)
		})
	}
}

func TestGetAllIndices(t *testing.T) {
	service := compose.EnsureUpWithTimeout(t, 300, "elasticsearch")
	host := service.Host()

	// Create two indices, one hidden, one not
	indexVisible, err := createIndex(host, false)
	require.NoError(t, err)

	indexHidden, err := createIndex(host, true)
	require.NoError(t, err)

	config := getConfig(host)

	metricSets := mbtest.NewReportingMetricSetV2Errors(t, config)
	for _, metricSet := range metricSets {
		// We only care about the index metricset for this test
		if metricSet.Name() != "index" {
			continue
		}

		events, errs := mbtest.ReportingFetchV2Error(metricSet)

		require.Empty(t, errs)
		require.NotEmpty(t, events)

		// Check that we have events for both indices we created
		var idxVisibleExists, idxHiddenExists bool
		for _, event := range events {

			name, ok := event.MetricSetFields["name"]
			require.True(t, ok)

			switch name {
			case indexVisible:
				idxVisibleExists = true
			case indexHidden:
				idxHiddenExists = true
			}
		}

		require.True(t, idxVisibleExists)
		require.True(t, idxHiddenExists)
	}
}

// GetConfig returns config for elasticsearch module
func getConfigForMetricset(metricset string, host string) map[string]interface{} {
	return map[string]interface{}{
		"module":                     elasticsearch.ModuleName,
		"metricsets":                 []string{metricset},
		"hosts":                      []string{host},
		"index_recovery.active_only": false,
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     elasticsearch.ModuleName,
		"metricsets": metricSets,
		"hosts":      []string{host},
		// index_recovery.active_only is part of the config of the index_recovery Metricset and it is required during the
		// test of that particular metricset to get some data from the ES node (instead of an empty JSON if set to true)
		"index_recovery.active_only": false,
	}
}

func setupTest(t *testing.T, esHost string, esVersion *version.V) {
	_, err := createIndex(esHost, false)
	require.NoError(t, err)

	err = enableTrialLicense(esHost, esVersion)
	require.NoError(t, err)

	err = createMLJob(esHost, esVersion)
	require.NoError(t, err)

	err = createCCRStats(esHost)
	require.NoError(t, err)

	err = createEnrichStats(esHost)
	require.NoError(t, err)
}

// createIndex creates an random elasticsearch index
func createIndex(host string, isHidden bool) (string, error) {
	indexName := randString(5)

	reqBody := fmt.Sprintf(`{ "settings": { "index.hidden": %v } }`, isHidden)

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%v/%v", host, indexName), strings.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("could not build create index request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not send create index request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP error %d: %s, %s", resp.StatusCode, resp.Status, string(respBody))
	}

	return indexName, nil
}

// enableTrialLicense creates and elasticsearch index in case it does not exit yet
func enableTrialLicense(host string, version *version.V) error {
	client := &http.Client{}

	enabled, err := checkTrialLicenseEnabled(host, version)
	if err != nil {
		return err
	}
	if enabled {
		return nil
	}

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

// checkTrialLicenseEnabled creates and elasticsearch index in case it does not exit yet
func checkTrialLicenseEnabled(host string, version *version.V) (bool, error) {
	var licenseURL string
	if version.Major < 7 {
		licenseURL = "/_xpack/license"
	} else {
		licenseURL = "/_license"
	}

	resp, err := http.Get("http://" + host + licenseURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var data struct {
		License struct {
			Status string `json:"status"`
			Type   string `json:"type"`
		} `json:"license"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, err
	}

	active := data.License.Status == "active"
	isTrial := data.License.Type == "trial"
	return active && isTrial, nil
}

func createMLJob(host string, version *version.V) error {

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
		return fmt.Errorf("error doing PUT request when creating ML job: %w", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error loading ml job %d: %s, %s", resp.StatusCode, resp.Status, string(body))
	}

	return nil
}

func createCCRStats(host string) error {
	err := setupCCRRemote(host)
	if err != nil {
		return errors.New("error setup CCR remote settings")
	}

	err = createCCRLeaderIndex(host)
	if err != nil {
		return errors.New("error creating CCR leader index")
	}

	err = createCCRFollowerIndex(host)
	if err != nil {
		return errors.New("error creating CCR follower index")
	}

	// Give ES sufficient time to do the replication and produce stats
	checkCCRStats := func() (bool, error) {
		return checkCCRStatsExists(host)
	}

	exists, err := waitForSuccess(checkCCRStats, 500*time.Millisecond, 10)
	if err != nil {
		return fmt.Errorf("error checking if CCR stats exist: %w", err)
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

func createEnrichStats(host string) error {
	err := createEnrichSourceIndex(host)
	if err != nil {
		return fmt.Errorf("error creating enrich source index: %w", err)
	}

	err = createEnrichPolicy(host)
	if err != nil {
		return fmt.Errorf("error creating enrich policy: %w", err)
	}

	err = executeEnrichPolicy(host)
	if err != nil {
		return fmt.Errorf("error executing enrich policy: %w", err)
	}

	err = createEnrichIngestPipeline(host)
	if err != nil {
		return fmt.Errorf("error creating ingest pipeline with enrich processor: %w", err)
	}

	err = ingestAndEnrichDoc(host)
	if err != nil {
		return fmt.Errorf("error ingesting doc for enrichment: %w", err)
	}

	return nil
}

func createEnrichSourceIndex(host string) error {
	sourceDoc, err := ioutil.ReadFile("enrich/_meta/test/source_doc.json")
	if err != nil {
		return err
	}

	docURL := "/users/_doc/1?refresh=wait_for"
	_, _, err = httpPutJSON(host, docURL, sourceDoc)
	return err
}

func createEnrichPolicy(host string) error {
	policy, err := ioutil.ReadFile("enrich/_meta/test/policy.json")
	if err != nil {
		return err
	}

	policyURL := "/_enrich/policy/users-policy"
	_, _, err = httpPutJSON(host, policyURL, policy)
	return err
}

func executeEnrichPolicy(host string) error {
	executeURL := "/_enrich/policy/users-policy/_execute"
	_, _, err := httpPostJSON(host, executeURL, nil)
	return err
}

func createEnrichIngestPipeline(host string) error {
	pipeline, err := ioutil.ReadFile("enrich/_meta/test/ingest_pipeline.json")
	if err != nil {
		return err
	}

	pipelineURL := "/_ingest/pipeline/user_lookup"
	_, _, err = httpPutJSON(host, pipelineURL, pipeline)
	return err
}

func ingestAndEnrichDoc(host string) error {
	targetDoc, err := ioutil.ReadFile("enrich/_meta/test/target_doc.json")
	if err != nil {
		return err
	}

	docURL := "/my_index/_doc/my_id?pipeline=user_lookup"
	_, _, err = httpPutJSON(host, docURL, targetDoc)
	return err
}

func countIndices(elasticsearchHostPort string) (int, error) {
	return countCatItems(elasticsearchHostPort, "indices", "&expand_wildcards=open,hidden")
}

func countShards(elasticsearchHostPort string) (int, error) {
	return countCatItems(elasticsearchHostPort, "shards", "")
}

func countCatItems(elasticsearchHostPort, catObject, extraParams string) (int, error) {
	resp, err := http.Get("http://" + elasticsearchHostPort + "/_cat/" + catObject + "?format=json" + extraParams)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var data []mapstr.M
	err = json.Unmarshal(body, &data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func checkSkip(t *testing.T, metricset string, ver *version.V) {
	checkSkipFeature := func(name string, availableVersion *version.V) {
		isAPIAvailable := elastic.IsFeatureAvailable(ver, availableVersion)
		if !isAPIAvailable {
			t.Skipf("elasticsearch %s stats API is not available until %s", name, availableVersion)
		}
	}

	switch metricset {
	case "ccr":
		checkSkipFeature("CCR", elasticsearch.CCRStatsAPIAvailableVersion)
	case "enrich":
		checkSkipFeature("Enrich", elasticsearch.EnrichStatsAPIAvailableVersion)
	}
}

func getElasticsearchVersion(elasticsearchHostPort string) (*version.V, error) {
	resp, err := http.Get("http://" + elasticsearchHostPort + "/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data mapstr.M
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	v, err := data.GetValue("version.number")
	if err != nil {
		return nil, err
	}

	return version.New(v.(string))
}

func httpPutJSON(host, path string, body []byte) ([]byte, *http.Response, error) {
	return httpSendJSON(host, path, "PUT", body)
}

func httpPostJSON(host, path string, body []byte) ([]byte, *http.Response, error) {
	return httpSendJSON(host, path, "POST", body)
}

func httpSendJSON(host, path, method string, body []byte) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(method, "http://"+host+path, bytes.NewReader(body))
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

func waitForSuccess(f checkSuccessFunction, retryInterval time.Duration, numAttempts int) (bool, error) {
	for numAttempts > 0 {
		success, err := f()
		if err != nil {
			return false, err
		}

		if success {
			return success, nil
		}

		time.Sleep(retryInterval)
		numAttempts--
	}

	return false, nil
}

func randString(len int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]byte, len)
	aIdx := int('a')
	for i := range b {
		charIdx := aIdx + rand.Intn(26)
		b[i] = byte(charIdx)
	}

	return string(b)
}
