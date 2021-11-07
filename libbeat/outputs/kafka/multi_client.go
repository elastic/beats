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

package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outest"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

type (
	multiClientBuilder func(clientId string) (outputs.Client, error)

	clientInfo struct {
		Client     outputs.Client
		LastUsedAt time.Time
	}

	MultiClient struct {
		builder          multiClientBuilder
		log              *logp.Logger
		clientIdSelector outil.Selector

		clients   map[string]*clientInfo
		clientsMu sync.RWMutex

		clientsGCStopper chan struct{}
	}
)

var (
	_ outputs.NetworkClient = &MultiClient{}
)

const (
	clientsGCPeriod = 15 * time.Minute
)

func NewKafkaMultiClient(
	beat beat.Info,
	observer outputs.Observer,
	hosts []string,
	topic outil.Selector,
	config *kafkaConfig,
	log *logp.Logger,
	clientIdSelector outil.Selector,
) (*MultiClient, error) {

	k := &MultiClient{
		log:              log,
		clientIdSelector: clientIdSelector,
		clients:          make(map[string]*clientInfo),
	}

	k.builder = func(clientId string) (outputs.Client, error) {
		goMetricsName := defaultGoMetricsName + "." + clientId // we must use separate metricsNames to avoid data races
		cfg, err := newSaramaConfig(log, config, goMetricsName)
		if err != nil {
			return nil, err
		}
		if clientId != "" {
			cfg.ClientID = clientId
		}

		writer, err := codec.CreateEncoder(beat, config.Codec)
		if err != nil {
			return nil, err
		}

		client, err := newKafkaClient(observer, hosts, beat.IndexPrefix, config.Key, topic, writer, cfg)
		if err != nil {
			return nil, err
		}

		log.Debugf("Connect to kafka hosts %v with clientId: %q", hosts, cfg.ClientID)
		err = client.Connect()
		if err != nil {
			return nil, err
		}

		return client, err
	}

	return k, nil
}

func (k *MultiClient) Connect() error {
	k.clientsMu.Lock()
	defer k.clientsMu.Unlock()

	// Checking a builder
	client, err := k.builder("")
	if err != nil {
		return fmt.Errorf(`could not create default client: %w`, err)
	}
	_ = client.Close()

	k.clientsGCStopper = make(chan struct{})
	go k.clientsGC()

	return nil
}

func (k *MultiClient) Close() error {
	k.clientsMu.Lock()
	defer k.clientsMu.Unlock()

	close(k.clientsGCStopper)

	var lastErr error

	for clientId, client := range k.clients {
		err := client.Client.Close()
		if err != nil {
			lastErr = err
		}

		delete(k.clients, clientId)
	}

	return lastErr
}

func (k *MultiClient) Publish(ctx context.Context, batch publisher.Batch) error {
	eventsByClientId := map[string][]beat.Event{}

	// Separate events by kafka clientId
	for _, event := range batch.Events() {
		kafkaClientId, _ := k.clientIdSelector.Select(&event.Content)

		eventsByClientId[kafkaClientId] = append(eventsByClientId[kafkaClientId], event.Content)
	}

	var (
		forRetry   []publisher.Event
		forRetryMu sync.Mutex
		wg         sync.WaitGroup
	)

	retryBeatEvents := func(events []beat.Event) {
		forRetryMu.Lock()
		for _, event := range events {
			forRetry = append(forRetry, publisher.Event{Content: event})
		}
		forRetryMu.Unlock()
	}

	for clientId := range eventsByClientId {
		clientId := clientId

		wg.Add(1)
		go func() {
			events := eventsByClientId[clientId]

			client, err := k.getClient(clientId)
			if err != nil {
				k.log.Warnf("getClient failed: %s", err)
				retryBeatEvents(events)
				wg.Done()
				return
			}
			if client == nil {
				k.log.Error("there is no client connection")
				retryBeatEvents(events)
				wg.Done()
				return
			}

			monoBatch := outest.NewBatch(events...)
			monoBatch.OnSignal = func(sig outest.BatchSignal) {
				defer wg.Done()

				switch sig.Tag {
				case outest.BatchRetryEvents:
					forRetryMu.Lock()
					forRetry = append(forRetry, sig.Events...)
					forRetryMu.Unlock()

				case outest.BatchACK:
					// all ok

				default:
					k.log.Warnf("unsupported signal tag %d", sig.Tag)
				}
			}

			err = client.Publish(ctx, monoBatch)
			if err != nil {
				k.log.Warnf("publish error: %v", err)
			}
		}()
	}

	wg.Wait()

	if len(forRetry) > 0 {
		batch.RetryEvents(forRetry)
	} else {
		batch.ACK()
	}

	return nil
}

func (k *MultiClient) String() string {
	return "kafkaMultiClient"
}

func (k *MultiClient) getClient(kafkaClientId string) (outputs.Client, error) {
	var (
		ci *clientInfo
		ok bool
	)

	defer func() {
		if ci != nil {
			ci.LastUsedAt = time.Now()
		}
	}()

	k.clientsMu.RLock()
	ci, ok = k.clients[kafkaClientId]
	k.clientsMu.RUnlock()

	if ok {
		return ci.Client, nil
	}

	k.clientsMu.Lock()
	defer k.clientsMu.Unlock()

	ci, ok = k.clients[kafkaClientId]
	if ok {
		return ci.Client, nil
	}

	client, err := k.builder(kafkaClientId)
	if err != nil {
		return nil, fmt.Errorf(`could not create kafka client for clientId %q: %w`, kafkaClientId, err)
	}

	k.clients[kafkaClientId] = &clientInfo{
		Client:     client,
		LastUsedAt: time.Now(),
	}

	return client, nil
}

func (k *MultiClient) clientsGC() {
	ticker := time.NewTicker(clientsGCPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// pass
		case <-k.clientsGCStopper:
			return
		}

		dropOlderThan := time.Now().Add(-clientsGCPeriod) // Should add some random?
		k.clientsMu.Lock()
		for clientId, ci := range k.clients {
			if ci.LastUsedAt.Before(dropOlderThan) {
				err := ci.Client.Close()
				delete(k.clients, clientId)

				k.log.Debugf("Drop useless kafka connection with clientId %q (close error: %v)", clientId, err)
			}
		}
		k.clientsMu.Unlock()
	}
}
