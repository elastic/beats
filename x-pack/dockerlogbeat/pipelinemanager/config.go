// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"crypto/sha1"
	"reflect"
	"sort"

	"github.com/pkg/errors"
)

// ContainerOutputConfig has all the options we'll expect from --log-opts
type ContainerOutputConfig struct {
	Endpoint        string `config:"output.elasticsearch.hosts"`
	User            string `config:"output.elasticsearch.username"`
	Password        string `config:"output.elasticsearch.password"`
	Index           string `config:"output.elasticsearch.index"`
	Pipeline        string `config:"output.elasticsearch.pipeline"`
	APIKey          string `config:"output.elasticsearch.api_key"`
	Timeout         string `config:"output.elasticsearch.timeout"`
	BackoffInit     string `config:"output.elasticsearch.backoff.init"`
	BackoffMax      string `config:"output.elasticsearch.backoff.max"`
	CloudID         string `config:"cloud.id"`
	CloudAuth       string `config:"cloud.auth"`
	ProxyURL        string `config:"output.elasticsearch.proxy_url"`
	ILMEnabled      bool   `config:"setup.ilm.enabled"`
	ILMRollverAlias string `config:"setup.ilm.rollover_alias"`
	ILMPatterns     string `config:"setup.ilm.pattern"`
	TemplateName    string `config:"setup.template.name"`
	TempatePattern  string `config:"setup.template.pattern"`
}

// NewCfgFromRaw returns a ContainerOutputConfig based on a raw config we get from the API
func NewCfgFromRaw(input map[string]string) (ContainerOutputConfig, error) {

	newCfg := ContainerOutputConfig{}
	endpoint, ok := input["endpoint"]
	if !ok {
		return newCfg, errors.New("An endpoint flag is required")
	}
	newCfg.Endpoint = endpoint

	var isIndex bool

	newCfg.User, _ = input["user"]
	newCfg.Password, _ = input["password"]
	newCfg.Index, isIndex = input["index"]
	newCfg.Pipeline, _ = input["pipeline"]
	newCfg.CloudID, _ = input["cloud_id"]
	newCfg.CloudAuth, _ = input["cloud_auth"]
	newCfg.ProxyURL, _ = input["proxy_url"]
	newCfg.APIKey, _ = input["api_key"]
	newCfg.Timeout, _ = input["timeout"]

	rawILM, isILM := input["ilm_enabled"]

	if isILM {
		if rawILM == "true" {
			newCfg.ILMEnabled = true
		} else if rawILM == "false" {
			newCfg.ILMEnabled = false
		} else {
			return newCfg, errors.New("ilm_enabled must be 'true' or 'false'")
		}

		if isIndex && newCfg.ILMEnabled {
			return newCfg, errors.New("Cannot set output index while ILM is enabled")
		}

	}

	newCfg.ILMRollverAlias, _ = input["ilm_rollover_alias"]
	newCfg.ILMPatterns, _ = input["ilm_pattern"]

	if isIndex {
		tname, tnameOk := input["template_name"]
		tpattern, tpatternOk := input["template_pattern"]

		if !tnameOk || !tpatternOk {
			return newCfg, errors.New("template_pattern and template_name must be set if index is set")
		}
		newCfg.TempatePattern = tpattern
		newCfg.TemplateName = tname
	}

	return newCfg, nil
}

// GetHash returns a sha1 hash of the config
func (cfg ContainerOutputConfig) GetHash() string {
	var hashString string
	var orderedVal []string

	values := reflect.ValueOf(cfg)
	for i := 0; i < values.NumField(); i++ {
		valRaw := values.Field(i).Interface()
		if parsedVal, ok := valRaw.(string); ok {
			orderedVal = append(orderedVal, parsedVal)
		} else if parsedVal, ok := valRaw.(bool); ok {
			if parsedVal {
				orderedVal = append(orderedVal, "true")
			} else {
				orderedVal = append(orderedVal, "false")
			}
		}

	}

	sort.Strings(orderedVal)

	for _, val := range orderedVal {
		hashString = hashString + val
	}

	sum := sha1.Sum([]byte(hashString))

	return string(sum[:])
}
