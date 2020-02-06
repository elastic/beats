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

package template

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	commonP "github.com/elastic/beats/libbeat/common/pipeline"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	esIngestPipelinePath = "/_ingest/pipeline/"

	beatsFinalPipeline = common.MapStr{
		"description": "Beats final pipeline for enrichment with ingest timestamp, GeoIP, and ASN",
		"processors": []common.MapStr{
			common.MapStr{
				"geoip": common.MapStr{
					"if":             "ctx.source?.geo == null",
					"field":          "source.ip",
					"target_field":   "source.geo",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"geoip": common.MapStr{
					"if":             "ctx.destination?.geo == null",
					"field":          "destination.ip",
					"target_field":   "destination.geo",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"geoip": common.MapStr{
					"if":            "ctx.source?.as == null",
					"field":         "source.ip",
					"target_field":  "source.as",
					"database_file": "GeoLite2-ASN.mmdb",
					"properties": []string{
						"asn",
						"organization_name",
					},
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"geoip": common.MapStr{
					"if":            "ctx.destination?.as == null",
					"field":         "destination.ip",
					"target_field":  "destination.as",
					"database_file": "GeoLite2-ASN.mmdb",
					"properties": []string{
						"asn",
						"organization_name",
					},
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"rename": common.MapStr{
					"field":          "source.as.asn",
					"target_field":   "source.as.number",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"rename": common.MapStr{
					"field":          "source.as.organization_name",
					"target_field":   "source.as.organization.name",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"rename": common.MapStr{
					"field":          "destination.as.asn",
					"target_field":   "destination.as.number",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"rename": common.MapStr{
					"field":          "destination.as.organization_name",
					"target_field":   "destination.as.organization.name",
					"ignore_missing": true,
				},
			},
			common.MapStr{
				"set": common.MapStr{
					"field": "event.ingested",
					"value": "{{_ingest.timestamp}}",
				},
			},
		},
	}
)

// PipelineConfig holds config information about an Elasticsearch pipeline.
type PipelineConfig struct {
	Enabled   bool   `config:"enabled"`
	Overwrite bool   `config:"overwrite"`
	Name      string `config:"name"`
	File      string `config:"file"`
}

func getFinalPipeline(config TemplateConfig) (common.MapStr, error) {
	pipeline := beatsFinalPipeline

	if len(config.FinalPipeline.File) > 0 {
		fileContents, err := ioutil.ReadFile(config.FinalPipeline.File)
		if err != nil {
			return nil, fmt.Errorf("could not read pipeline file '%s'. Error: %v", config.FinalPipeline.File, err)
		}

		pipeline, err = commonP.UnmarshalPipeline(config.FinalPipeline.File, fileContents)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal pipeline file '%s'. Error: %v", config.FinalPipeline.File, err)
		}
	}

	return pipeline, nil
}

// loadIngestPipeline loads an ingest pipeline into Elasticsearch,
// overwriting an existing pipeline if it exists.
// If you wish to not overwrite an existing pipeline then use ingestPipelineExists
// prior to calling this method.
func (l *ESLoader) loadIngestPipeline(pipelineName string, config TemplateConfig) error {
	logp.Info("Try loading ingest pipeline '%s' into Elasticsearch", pipelineName)

	pipeline, err := getFinalPipeline(config)
	if err != nil {
		return err
	}

	status, body, err := l.client.Request("PUT", esIngestPipelinePath+pipelineName, "", nil, pipeline)
	if status != http.StatusOK || err != nil {
		return fmt.Errorf("could not load ingest pipeline. Status: %v. Error: %v. Response body: %s", status, err, body)
	}

	return nil
}

// ingestPipelineExists checks if the final ingest pipeline already exists.
func (l *ESLoader) ingestPipelineExists(name string) (bool, error) {
	if l.client == nil {
		return false, nil
	}

	status, body, err := l.client.Request("GET", esIngestPipelinePath+name, "", nil, nil)
	if status == http.StatusNotFound {
		return false, nil
	} else if status == http.StatusOK && err == nil {
		return strings.Contains(string(body), name), nil
	} else {
		return false, fmt.Errorf("failed to check if ingest pipeline '%s' exists. Status: %v. Error: %v. Response body: %s",
			name, status, err, body)
	}
}
