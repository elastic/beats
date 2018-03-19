// +build integration

package mlimporter

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch/estest"
)

const sampleJob = `
{
  "description" : "Anomaly detector for changes in event rates of nginx.access.response_code responses",
  "analysis_config" : {
    "bucket_span": "1h",
    "summary_count_field_name": "doc_count",
    "detectors": [
      {
        "detector_description": "Event rate for nginx.access.response_code",
        "function": "count",
        "partition_field_name": "nginx.access.response_code"
      }
    ],
    "influencers": ["nginx.access.response_code"]
  },
  "data_description": {
    "time_field": "@timestamp",
    "time_format": "epoch_ms"
  },
  "model_plot_config": {
    "enabled": true
  }
}
`

const sampleDatafeed = `
{
    "job_id": "PLACEHOLDER",
    "indexes": [
      "filebeat-*"
    ],
    "types": [
      "doc",
      "log"
    ],
    "query": {
      "match_all": {
        "boost": 1
      }
    },
    "aggregations": {
      "buckets": {
        "date_histogram": {
          "field": "@timestamp",
          "interval": 3600000,
          "offset": 0,
          "order": {
            "_key": "asc"
          },
          "keyed": false,
          "min_doc_count": 0
        },
        "aggregations": {
          "@timestamp": {
            "max": {
              "field": "@timestamp"
            }
          },
          "nginx.access.response_code": {
              "terms": {
                "field": "nginx.access.response_code",
                "size": 10000
              }
          }
        }
      }
    }
}
`

func TestImportJobs(t *testing.T) {
	logp.TestingSetup()

	client := estest.GetTestingElasticsearch(t)

	haveXpack, err := HaveXpackML(client)
	assert.NoError(t, err)
	if !haveXpack {
		t.Skip("Skip ML tests because xpack/ML is not available in Elasticsearch")
	}

	workingDir, err := ioutil.TempDir("", "machine-learning")
	assert.NoError(t, err)
	defer os.RemoveAll(workingDir)

	assert.NoError(t, ioutil.WriteFile(workingDir+"/job.json", []byte(sampleJob), 0644))
	assert.NoError(t, ioutil.WriteFile(workingDir+"/datafeed.json", []byte(sampleDatafeed), 0644))

	mlconfig := MLConfig{
		ID:           "test-ml-config",
		JobPath:      workingDir + "/job.json",
		DatafeedPath: workingDir + "/datafeed.json",
	}

	err = ImportMachineLearningJob(client, &mlconfig)
	assert.NoError(t, err)

	// check by GETing back

	status, response, err := client.Request("GET", "/_xpack/ml/anomaly_detectors", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	logp.Debug("mltest", "Response: %s", response)

	type jobRes struct {
		Count int `json:"count"`
		Jobs  []struct {
			JobId   string `json:"job_id"`
			JobType string `json:"job_type"`
		}
	}
	var res jobRes

	err = json.Unmarshal(response, &res)
	assert.NoError(t, err)
	assert.True(t, res.Count >= 1)
	found := false
	for _, job := range res.Jobs {
		if job.JobId == "test-ml-config" {
			found = true
			assert.Equal(t, job.JobType, "anomaly_detector")
		}
	}
	assert.True(t, found)

	status, response, err = client.Request("GET", "/_xpack/ml/datafeeds", "", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 200, status)

	logp.Debug("mltest", "Response: %s", response)
	type datafeedRes struct {
		Count     int `json:"count"`
		Datafeeds []struct {
			DatafeedId string `json:"datafeed_id"`
			JobId      string `json:"job_id"`
			QueryDelay string `json:"query_delay"`
		}
	}
	var df datafeedRes
	err = json.Unmarshal(response, &df)
	assert.NoError(t, err)
	assert.True(t, df.Count >= 1)
	found = false
	for _, datafeed := range df.Datafeeds {
		if datafeed.DatafeedId == "datafeed-test-ml-config" {
			found = true
			assert.Equal(t, datafeed.JobId, "test-ml-config")
			assert.Equal(t, datafeed.QueryDelay, "87034ms")
		}
	}
	assert.True(t, found)

	// importing again should not error out
	err = ImportMachineLearningJob(client, &mlconfig)
	assert.NoError(t, err)
}
