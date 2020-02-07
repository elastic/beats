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

package elasticsearch

import (
	"net/url"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/esclientleg"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
)

func init() {
	outputs.RegisterType("elasticsearch", makeES)
}

const logSelector = "elasticsearch"

func makeES(
	im outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	if !cfg.HasField("bulk_max_size") {
		cfg.SetInt("bulk_max_size", -1, defaultBulkSize)
	}

	index, pipeline, err := buildSelectors(im, beat, cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	var proxyURL *url.URL
	if !config.ProxyDisable {
		proxyURL, err = esclientleg.ParseProxyURL(config.ProxyURL)
		if err != nil {
			return outputs.Fail(err)
		}
		if proxyURL != nil {
			log.Infof("Using proxy URL: %s", proxyURL)
		}
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		esURL, err := common.MakeURL(config.Protocol, config.Path, host, 9200)
		if err != nil {
			log.Errorf("Invalid host param set: %s, Error: %+v", host, err)
			return outputs.Fail(err)
		}

		var client outputs.NetworkClient
		client, err = NewClient(ClientSettings{
			URL:              esURL,
			Index:            index,
			Pipeline:         pipeline,
			Proxy:            proxyURL,
			ProxyDisable:     config.ProxyDisable,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			APIKey:           config.APIKey,
			Parameters:       params,
			Headers:          config.Headers,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
			Observer:         observer,
			EscapeHTML:       config.EscapeHTML,
		}, &connectCallbackRegistry)
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
}

func buildSelectors(
	im outputs.IndexManager,
	beat beat.Info,
	cfg *common.Config,
) (index outputs.IndexSelector, pipeline *outil.Selector, err error) {
	index, err = im.BuildSelector(cfg)
	if err != nil {
		return index, pipeline, err
	}

	pipelineSel, err := outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "pipeline",
		MultiKey:         "pipelines",
		EnableSingleOnly: true,
		FailEmpty:        false,
	})
	if err != nil {
		return index, pipeline, err
	}

	if !pipelineSel.IsEmpty() {
		pipeline = &pipelineSel
	}

	return index, pipeline, err
}
