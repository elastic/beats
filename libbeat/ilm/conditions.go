package ilm

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

//EnabledFor checks if the given Elasticsearch client is valid for ILM.
func EnabledFor(client ESClient) bool {
	if client == nil {
		return true
	}
	if !checkElasticsearchVersionIlm(client) {
		return false
	}
	return checkILMFeatureEnabled(client)
}

func checkElasticsearchVersionIlm(esClient ESClient) bool {
	if esClient == nil {
		logp.Warn(noElasticsearchClientSet)
		return false
	}
	esV := esClient.GetVersion()
	requiredVersion, err := common.NewVersion("6.6.0")
	if err != nil {
		return false
	}

	if esV.LessThan(requiredVersion) {
		logp.Warn(ilmNotSupported)
		return false
	}

	return true
}

func checkILMFeatureEnabled(client ESClient) bool {
	if client == nil {
		logp.Warn(noElasticsearchClientSet)
		return false
	}

	code, body, err := client.Request("GET", "/_xpack", "", nil, nil)
	// If we get a 400, it's assumed to be the OSS version of Elasticsearch
	if code == 400 {
		logp.Warn(ilmNotSupported)
		return false
	}
	if err != nil {
		logp.Warn("error occured when checking for ILM features in Elasticsearch %s", err.Error())
		return false
	}

	var response struct {
		Features struct {
			ILM struct {
				Available bool `json:"available"`
				Enabled   bool `json:"enabled"`
			} `json:"ilm"`
		} `json:"features"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		logp.Warn("failed to parse JSON response: %v", err)
		return false
	}

	if !response.Features.ILM.Available || !response.Features.ILM.Enabled {
		logp.Warn(ilmNotSupported)
		return false
	}

	return true
}

const (
	ilmNotSupported          = "current Elasticsearch setup does not qualify for ILM feature"
	noElasticsearchClientSet = "no Elasticsearch client is set"
)
