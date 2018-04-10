package fileset

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/logp"
)

// PipelineLoaderFactory builds and returns a PipelineLoader
type PipelineLoaderFactory func() (PipelineLoader, error)

// PipelineLoader is a subset of the Elasticsearch client API capable of loading
// the pipelines.
type PipelineLoader interface {
	LoadJSON(path string, json map[string]interface{}) ([]byte, error)
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() string
}

// LoadPipelines loads the pipelines for each configured fileset.
func (reg *ModuleRegistry) LoadPipelines(esClient PipelineLoader, overwrite bool) error {
	for module, filesets := range reg.registry {
		for name, fileset := range filesets {
			// check that all the required Ingest Node plugins are available
			requiredProcessors := fileset.GetRequiredProcessors()
			logp.Debug("modules", "Required processors: %s", requiredProcessors)
			if len(requiredProcessors) > 0 {
				err := checkAvailableProcessors(esClient, requiredProcessors)
				if err != nil {
					return fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
				}
			}

			pipelineID, content, err := fileset.GetPipeline(esClient.GetVersion())
			if err != nil {
				return fmt.Errorf("Error getting pipeline for fileset %s/%s: %v", module, name, err)
			}
			err = loadPipeline(esClient, pipelineID, content, overwrite)
			if err != nil {
				return fmt.Errorf("Error loading pipeline for fileset %s/%s: %v", module, name, err)
			}
		}
	}
	return nil
}

func loadPipeline(esClient PipelineLoader, pipelineID string, content map[string]interface{}, overwrite bool) error {
	path := "/_ingest/pipeline/" + pipelineID
	if !overwrite {
		status, _, _ := esClient.Request("GET", path, "", nil, nil)
		if status == 200 {
			logp.Debug("modules", "Pipeline %s already loaded", pipelineID)
			return nil
		}
	}
	body, err := esClient.LoadJSON(path, content)
	if err != nil {
		return interpretError(err, body)
	}
	logp.Info("Elasticsearch pipeline with ID '%s' loaded", pipelineID)
	return nil
}

func interpretError(initialErr error, body []byte) error {
	var response struct {
		Error struct {
			RootCause []struct {
				Type   string `json:"type"`
				Reason string `json:"reason"`
				Header struct {
					ProcessorType string `json:"processor_type"`
				} `json:"header"`
				Index string `json:"index"`
			} `json:"root_cause"`
		} `json:"error"`
	}
	err := json.Unmarshal(body, &response)
	if err != nil {
		// this might be ES < 2.0. Do a best effort to check for ES 1.x
		var response1x struct {
			Error string `json:"error"`
		}
		err1x := json.Unmarshal(body, &response1x)
		if err1x == nil && response1x.Error != "" {
			return fmt.Errorf("The Filebeat modules require Elasticsearch >= 5.0. "+
				"This is the response I got from Elasticsearch: %s", body)
		}

		return fmt.Errorf("couldn't load pipeline: %v. Additionally, error decoding response body: %s",
			initialErr, body)
	}

	// missing plugins?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "parse_exception" &&
		strings.HasPrefix(response.Error.RootCause[0].Reason, "No processor type exists with name") &&
		response.Error.RootCause[0].Header.ProcessorType != "" {

		plugins := map[string]string{
			"geoip":      "ingest-geoip",
			"user_agent": "ingest-user-agent",
		}
		plugin, ok := plugins[response.Error.RootCause[0].Header.ProcessorType]
		if !ok {
			return fmt.Errorf("This module requires an Elasticsearch plugin that provides the %s processor. "+
				"Please visit the Elasticsearch documentation for instructions on how to install this plugin. "+
				"Response body: %s", response.Error.RootCause[0].Header.ProcessorType, body)
		}

		return fmt.Errorf("This module requires the %s plugin to be installed in Elasticsearch. "+
			"You can install it using the following command in the Elasticsearch home directory:\n"+
			"    sudo bin/elasticsearch-plugin install %s", plugin, plugin)
	}

	// older ES version?
	if len(response.Error.RootCause) > 0 &&
		response.Error.RootCause[0].Type == "invalid_index_name_exception" &&
		response.Error.RootCause[0].Index == "_ingest" {

		return fmt.Errorf("The Ingest Node functionality seems to be missing from Elasticsearch. "+
			"The Filebeat modules require Elasticsearch >= 5.0. "+
			"This is the response I got from Elasticsearch: %s", body)
	}

	return fmt.Errorf("couldn't load pipeline: %v. Response body: %s", initialErr, body)
}
