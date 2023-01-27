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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func init() {
	outputs.RegisterType("elasticsearch", makeES)
}

const logSelector = "elasticsearch"

func doSomething(cfg *config.C, esconf elasticsearchConfig) {
	for {
		time.Sleep(5 * time.Second)
		fmt.Println("**************************** here! ")
		if output := reload.RegisterV2.GetReloadableOutput(); output != nil {
			fmt.Println("Got an output reloader")

			// The outptu reloader needs the config to be "namespaced",
			// so we add it manually.
			tmp := map[string]any{
				"elasticsearch": cfg,
			}

			uc, err := config.NewConfigFrom(tmp)
			if err != nil {
				fmt.Println("cannot create config")
				panic("err")
			}

			// Calling this too quicky in a row (like every second) seems to cause panics
			// however if we wait a bit, it's fine
			// This might panic because of the metrics registry, I just don't know if will be an
			// issue in the real world.
			// The panic is happening on
			// /home/tiago/go/pkg/mod/github.com/elastic/elastic-agent-libs@v0.2.16/monitoring/metrics.go:264
			// something there is not safe there.
			if err := output.Reload(&reload.ConfigWithMeta{Config: uc}); err != nil {
				fmt.Println("cannot reload output")
				panic(err)
			}

			fmt.Println("Output reloaded, waiting 1min")
			time.Sleep(time.Minute)
		}
	}
}

func makeES(
	im outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C,
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

	policy, err := newNonIndexablePolicy(config.NonIndexablePolicy)
	if err != nil {
		log.Errorf("error while creating file identifier: %v", err)
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	if proxyURL := config.Transport.Proxy.URL; proxyURL != nil && !config.Transport.Proxy.Disable {
		log.Debugf("breaking down proxy URL. Scheme: '%s', host[:port]: '%s', path: '%s'", proxyURL.Scheme, proxyURL.Host, proxyURL.Path)
		log.Infof("Using proxy URL: %s", proxyURL)
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	if policy.action() == dead_letter_index {
		index = DeadLetterSelector{
			Selector:        index,
			DeadLetterIndex: policy.index(),
		}
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
			ConnectionSettings: eslegclient.ConnectionSettings{
				URL:              esURL,
				Beatname:         beat.Beat,
				Kerberos:         config.Kerberos,
				Username:         config.Username,
				Password:         config.Password,
				APIKey:           config.APIKey,
				Parameters:       params,
				Headers:          config.Headers,
				CompressionLevel: config.CompressionLevel,
				Observer:         observer,
				EscapeHTML:       config.EscapeHTML,
				Transport:        config.Transport,
			},
			Index:              index,
			Pipeline:           pipeline,
			Observer:           observer,
			NonIndexableAction: policy.action(),
		}, &connectCallbackRegistry)
		if err != nil {
			return outputs.Fail(err)
		}

		client = outputs.WithBackoff(client, config.Backoff.Init, config.Backoff.Max)
		clients[i] = client
	}

	// As the last thing, start the 'reloader'
	// go doSomething(cfg, config)

	return outputs.SuccessNet(config.LoadBalance, config.BulkMaxSize, config.MaxRetries, clients)
}

func buildSelectors(
	im outputs.IndexManager,
	beat beat.Info,
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
