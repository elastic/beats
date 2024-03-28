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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	outputs.RegisterType("elasticsearch", makeES)
}

const logSelector = "elasticsearch"

func makeES(
	im outputs.IndexManager,
	beatInfo beat.Info,
	observer outputs.Observer,
	cfg *config.C,
) (outputs.Group, error) {
	log := logp.NewLogger(logSelector)
	if !cfg.HasField("bulk_max_size") {
		if err := cfg.SetInt("bulk_max_size", -1, defaultBulkSize); err != nil {
			return outputs.Fail(err)
		}
	}

	index, pipeline, err := buildSelectors(im, beatInfo, cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	esConfig := defaultConfig
	preset, err := cfg.String("preset", -1)
	if err == nil && preset != "" {
		// Performance preset is present, apply it and log any fields that
		// were overridden
		overriddenFields, presetConfig, err := applyPreset(preset, cfg)
		if err != nil {
			return outputs.Fail(err)
		}
		log.Infof("Applying performance preset '%v': %v",
			preset, config.DebugString(presetConfig, false))
		for _, field := range overriddenFields {
			log.Warnf("Performance preset '%v' overrides user setting for field '%v'", preset, field)
		}
	}

	// Unpack the full config, including any performance preset overrides,
	// into the config struct.
	if err := cfg.Unpack(&esConfig); err != nil {
		return outputs.Fail(err)
	}

	deadLetterIndex, err := deadLetterIndexForPolicy(esConfig.NonIndexablePolicy)
	if err != nil {
		log.Errorf("error in non_indexable_policy: %v", err)
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	if proxyURL := esConfig.Transport.Proxy.URL; proxyURL != nil && !esConfig.Transport.Proxy.Disable {
		log.Debugf("breaking down proxy URL. Scheme: '%s', host[:port]: '%s', path: '%s'", proxyURL.Scheme, proxyURL.Host, proxyURL.Path)
		log.Infof("Using proxy URL: %s", proxyURL)
	}

	params := esConfig.Params
	if len(params) == 0 {
		params = nil
	}

	encoderFactory := func() queue.Encoder {
		return newEventEncoder(
			esConfig.EscapeHTML,
			index,
			pipeline,
		)
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		esURL, err := common.MakeURL(esConfig.Protocol, esConfig.Path, host, 9200)
		if err != nil {
			log.Errorf("Invalid host param set: %s, Error: %+v", host, err)
			return outputs.Fail(err)
		}

		var client outputs.NetworkClient
		client, err = NewClient(clientSettings{
			connection: eslegclient.ConnectionSettings{
				URL:              esURL,
				Beatname:         beatInfo.Beat,
				Kerberos:         esConfig.Kerberos,
				Username:         esConfig.Username,
				Password:         esConfig.Password,
				APIKey:           esConfig.APIKey,
				Parameters:       params,
				Headers:          esConfig.Headers,
				CompressionLevel: esConfig.CompressionLevel,
				Observer:         observer,
				EscapeHTML:       esConfig.EscapeHTML,
				Transport:        esConfig.Transport,
				IdleConnTimeout:  esConfig.Transport.IdleConnTimeout,
			},
			indexSelector:    index,
			pipelineSelector: pipeline,
			observer:         observer,
			deadLetterIndex:  deadLetterIndex,
		}, &connectCallbackRegistry)
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, esConfig.Backoff.Init, esConfig.Backoff.Max)
		clients[i] = client
	}

	return outputs.SuccessNet(esConfig.Queue, esConfig.LoadBalance, esConfig.BulkMaxSize, esConfig.MaxRetries, encoderFactory, clients)
}

func buildSelectors(
	im outputs.IndexManager,
	_ beat.Info,
	cfg *config.C,
) (index outputs.IndexSelector, pipeline *outil.Selector, err error) {
	index, err = im.BuildSelector(cfg)
	if err != nil {
		return index, pipeline, err
	}

	pipelineSel, err := buildPipelineSelector(cfg)
	if err != nil {
		return index, pipeline, err
	}

	if !pipelineSel.IsEmpty() {
		pipeline = &pipelineSel
	}

	return index, pipeline, err
}

func buildPipelineSelector(cfg *config.C) (outil.Selector, error) {
	return outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "pipeline",
		MultiKey:         "pipelines",
		EnableSingleOnly: true,
		FailEmpty:        false,
		Case:             outil.SelectorLowerCase,
	})
}
