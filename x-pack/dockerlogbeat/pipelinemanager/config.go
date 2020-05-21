// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
)

// ContainerOutputConfig has all the options we'll expect from --log-opts
type ContainerOutputConfig struct {
	Endpoint       []string `struct:"output.elasticsearch.hosts,omitempty"`
	User           string   `struct:"output.elasticsearch.username,omitempty"`
	Password       string   `struct:"output.elasticsearch.password,omitempty"`
	Index          string   `struct:"output.elasticsearch.index,omitempty"`
	Pipeline       string   `struct:"output.elasticsearch.pipeline,omitempty"`
	APIKey         string   `struct:"output.elasticsearch.api_key,omitempty"`
	Timeout        string   `struct:"output.elasticsearch.timeout,omitempty"`
	BackoffInit    string   `struct:"output.elasticsearch.backoff.init,omitempty"`
	BackoffMax     string   `struct:"output.elasticsearch.backoff.max,omitempty"`
	CloudID        string   `struct:"cloud.id,omitempty"`
	CloudAuth      string   `struct:"cloud.auth,omitempty"`
	ProxyURL       string   `struct:"output.elasticsearch.proxy_url,omitempty"`
	TemplateName   string   `struct:"setup.template.name,omitempty"`
	TempatePattern string   `struct:"setup.template.pattern,omitempty"`
}

// NewCfgFromRaw returns a ContainerOutputConfig based on a raw config we get from the API
func NewCfgFromRaw(input map[string]string) (ContainerOutputConfig, error) {

	newCfg := ContainerOutputConfig{}
	endpoint, ok := input["endpoint"]
	if !ok {
		return newCfg, errors.New("An endpoint flag is required")
	}

	endpointList := strings.Split(endpoint, ",")

	newCfg.Endpoint = endpointList

	var isIndex bool

	newCfg.User = input["user"]
	newCfg.Password = input["password"]
	newCfg.Index, isIndex = input["index"]
	newCfg.Pipeline = input["pipeline"]
	newCfg.CloudID = input["cloud_id"]
	newCfg.CloudAuth = input["cloud_auth"]
	newCfg.ProxyURL = input["proxy_url"]
	newCfg.APIKey = input["api_key"]
	newCfg.Timeout = input["timeout"]
	newCfg.BackoffInit = input["backoff_init"]
	newCfg.BackoffMax = input["backoff_max"]

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

// CreateConfig converts the struct into a config object that can be absorbed by libbeat
func (cfg ContainerOutputConfig) CreateConfig() (*common.Config, error) {

	// the use of typeconv is a hacky shim so we can impliment `omitempty` where needed.
	var tmp map[string]interface{}
	err := typeconv.Convert(&tmp, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error converting config struct to interface")
	}
	cfgFinal, err := common.NewConfigFrom(tmp)
	if err != nil {
		return nil, errors.Wrap(err, "error creating config object")
	}

	return cfgFinal, nil
}
