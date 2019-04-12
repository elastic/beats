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
	"errors"
	"fmt"
	"sync"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/outil"
)

func init() {
	outputs.RegisterType("elasticsearch", makeES)
}

var (
	debugf = logp.MakeDebug("elasticsearch")
)

var (
	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJSONEncodeFailed indicates encoding failures
	ErrJSONEncodeFailed = errors.New("json encode failed")

	// ErrResponseRead indicates error parsing Elasticsearch response
	ErrResponseRead = errors.New("bulk item status parse failed")
)

// Callbacks must not depend on the result of a previous one,
// because the ordering is not fixed.
type callbacksRegistry struct {
	callbacks map[uuid.UUID]ConnectCallback
	mutex     sync.Mutex
}

// XXX: it would be fantastic to do this without a package global
var connectCallbackRegistry = newCallbacksRegistry()

// NOTE(ph): We need to refactor this, right now this is the only way to ensure that every calls
// to an ES cluster executes a callback.
var globalCallbackRegistry = newCallbacksRegistry()

// RegisterGlobalCallback register a global callbacks.
func RegisterGlobalCallback(callback ConnectCallback) (uuid.UUID, error) {
	globalCallbackRegistry.mutex.Lock()
	defer globalCallbackRegistry.mutex.Unlock()

	// find the next unique key
	var key uuid.UUID
	var err error
	exists := true
	for exists {
		key, err = uuid.NewV4()
		if err != nil {
			return uuid.Nil, err
		}
		_, exists = globalCallbackRegistry.callbacks[key]
	}

	globalCallbackRegistry.callbacks[key] = callback
	return key, nil
}

func newCallbacksRegistry() callbacksRegistry {
	return callbacksRegistry{
		callbacks: make(map[uuid.UUID]ConnectCallback),
	}
}

// RegisterConnectCallback registers a callback for the elasticsearch output
// The callback is called each time the client connects to elasticsearch.
// It returns the key of the newly added callback, so it can be deregistered later.
func RegisterConnectCallback(callback ConnectCallback) (uuid.UUID, error) {
	connectCallbackRegistry.mutex.Lock()
	defer connectCallbackRegistry.mutex.Unlock()

	// find the next unique key
	var key uuid.UUID
	var err error
	exists := true
	for exists {
		key, err = uuid.NewV4()
		if err != nil {
			return uuid.Nil, err
		}
		_, exists = connectCallbackRegistry.callbacks[key]
	}

	connectCallbackRegistry.callbacks[key] = callback
	return key, nil
}

// DeregisterConnectCallback deregisters a callback for the elasticsearch output
// specified by its key. If a callback does not exist, nothing happens.
func DeregisterConnectCallback(key uuid.UUID) {
	connectCallbackRegistry.mutex.Lock()
	defer connectCallbackRegistry.mutex.Unlock()

	delete(connectCallbackRegistry.callbacks, key)
}

// DeregisterGlobalCallback deregisters a callback for the elasticsearch output
// specified by its key. If a callback does not exist, nothing happens.
func DeregisterGlobalCallback(key uuid.UUID) {
	globalCallbackRegistry.mutex.Lock()
	defer globalCallbackRegistry.mutex.Unlock()

	delete(globalCallbackRegistry.callbacks, key)
}

func makeES(
	im outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *common.Config,
) (outputs.Group, error) {
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

	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return outputs.Fail(err)
	}
	if proxyURL != nil {
		logp.Info("Using proxy URL: %s", proxyURL)
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, host := range hosts {
		esURL, err := common.MakeURL(config.Protocol, config.Path, host, 9200)
		if err != nil {
			logp.Err("Invalid host param set: %s, Error: %v", host, err)
			return outputs.Fail(err)
		}

		var client outputs.NetworkClient
		client, err = NewClient(ClientSettings{
			URL:              esURL,
			Index:            index,
			Pipeline:         pipeline,
			Proxy:            proxyURL,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
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

// NewConnectedClient creates a new Elasticsearch client based on the given config.
// It uses the NewElasticsearchClients to create a list of clients then returns
// the first from the list that successfully connects.
func NewConnectedClient(cfg *common.Config) (*Client, error) {
	clients, err := NewElasticsearchClients(cfg)
	if err != nil {
		return nil, err
	}

	errors := []string{}

	for _, client := range clients {
		err = client.Connect()
		if err != nil {
			logp.Err("Error connecting to Elasticsearch at %v: %v", client.Connection.URL, err)
			err = fmt.Errorf("Error connection to Elasticsearch %v: %v", client.Connection.URL, err)
			errors = append(errors, err.Error())
			continue
		}
		return &client, nil
	}
	return nil, fmt.Errorf("Couldn't connect to any of the configured Elasticsearch hosts. Errors: %v", errors)
}

// NewElasticsearchClients returns a list of Elasticsearch clients based on the given
// configuration. It accepts the same configuration parameters as the output,
// except for the output specific configuration options (index, pipeline,
// template) .If multiple hosts are defined in the configuration, a client is returned
// for each of them.
func NewElasticsearchClients(cfg *common.Config) ([]Client, error) {
	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return nil, err
	}

	config := defaultConfig
	if err = cfg.Unpack(&config); err != nil {
		return nil, err
	}

	tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return nil, err
	}
	if proxyURL != nil {
		logp.Info("Using proxy URL: %s", proxyURL)
	}

	params := config.Params
	if len(params) == 0 {
		params = nil
	}

	clients := []Client{}
	for _, host := range hosts {
		esURL, err := common.MakeURL(config.Protocol, config.Path, host, 9200)
		if err != nil {
			logp.Err("Invalid host param set: %s, Error: %v", host, err)
			return nil, err
		}

		client, err := NewClient(ClientSettings{
			URL:              esURL,
			Proxy:            proxyURL,
			TLS:              tlsConfig,
			Username:         config.Username,
			Password:         config.Password,
			Parameters:       params,
			Headers:          config.Headers,
			Timeout:          config.Timeout,
			CompressionLevel: config.CompressionLevel,
		}, nil)
		if err != nil {
			return clients, err
		}
		clients = append(clients, *client)
	}
	if len(clients) == 0 {
		return clients, fmt.Errorf("No hosts defined in the Elasticsearch output")
	}
	return clients, nil
}
