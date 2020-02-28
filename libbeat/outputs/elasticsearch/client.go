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
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
)

// Client is an elasticsearch client.
type Client struct {
	eslegclient.Connection

	index    outputs.IndexSelector
	pipeline *outil.Selector

	observer outputs.Observer

	log *logp.Logger
}

// ClientSettings contains the settings for a client.
type ClientSettings struct {
	eslegclient.ConnectionSettings
	Index    outputs.IndexSelector
	Pipeline *outil.Selector
	Observer outputs.Observer
}

type bulkResultStats struct {
	acked        int // number of events ACKed by Elasticsearch
	duplicates   int // number of events failed with `create` due to ID already being indexed
	fails        int // number of failed events (can be retried)
	nonIndexable int // number of failed events (not indexable -> must be dropped)
	tooMany      int // number of events receiving HTTP 429 Too Many Requests
}

const (
	defaultEventType = "doc"
)

// NewClient instantiates a new client.
func NewClient(
	s ClientSettings,
	onConnect *callbacksRegistry,
) (*Client, error) {
	pipeline := s.Pipeline
	if pipeline != nil && pipeline.IsEmpty() {
		pipeline = nil
	}

	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              s.URL,
		Username:         s.Username,
		Password:         s.Password,
		APIKey:           base64.StdEncoding.EncodeToString([]byte(s.APIKey)),
		Headers:          s.Headers,
		TLS:              s.TLS,
		Proxy:            s.Proxy,
		ProxyDisable:     s.ProxyDisable,
		Parameters:       s.Parameters,
		CompressionLevel: s.CompressionLevel,
		EscapeHTML:       s.EscapeHTML,
		Timeout:          s.Timeout,
	})
	if err != nil {
		return nil, err
	}

	conn.OnConnectCallback = func() error {
		globalCallbackRegistry.mutex.Lock()
		defer globalCallbackRegistry.mutex.Unlock()

		for _, callback := range globalCallbackRegistry.callbacks {
			err := callback(conn)
			if err != nil {
				return err
			}
		}

		if onConnect != nil {
			onConnect.mutex.Lock()
			defer onConnect.mutex.Unlock()

			for _, callback := range onConnect.callbacks {
				err := callback(conn)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	client := &Client{
		Connection: *conn,
		index:      s.Index,
		pipeline:   pipeline,

		observer: s.Observer,

		log: logp.NewLogger("elasticsearch"),
	}

	return client, nil
}

// Clone clones a client.
func (client *Client) Clone() *Client {
	// when cloning the connection callback and params are not copied. A
	// client's close is for example generated for topology-map support. With params
	// most likely containing the ingest node pipeline and default callback trying to
	// create install a template, we don't want these to be included in the clone.

	c, _ := NewClient(
		ClientSettings{
			ConnectionSettings: eslegclient.ConnectionSettings{
				URL:   client.URL,
				Proxy: client.Proxy,
				// Without the following nil check on proxyURL, a nil Proxy field will try
				// reloading proxy settings from the environment instead of leaving them
				// empty.
				ProxyDisable:      client.Proxy == nil,
				TLS:               client.TLS,
				Username:          client.Username,
				Password:          client.Password,
				APIKey:            client.APIKey,
				Parameters:        nil, // XXX: do not pass params?
				Headers:           client.Headers,
				Timeout:           client.HTTP.Timeout,
				CompressionLevel:  client.CompressionLevel,
				OnConnectCallback: nil,
				Observer:          nil,
				EscapeHTML:        false,
			},
			Index:    client.index,
			Pipeline: client.pipeline,
		},
		nil, // XXX: do not pass connection callback?
	)
	return c
}

func (client *Client) Publish(batch publisher.Batch) error {
	events := batch.Events()
	rest, err := client.publishEvents(events)
	if len(rest) == 0 {
		batch.ACK()
	} else {
		batch.RetryEvents(rest)
	}
	return err
}

// PublishEvents sends all events to elasticsearch. On error a slice with all
// events not published or confirmed to be processed by elasticsearch will be
// returned. The input slice backing memory will be reused by return the value.
func (client *Client) publishEvents(
	data []publisher.Event,
) ([]publisher.Event, error) {
	begin := time.Now()
	st := client.observer

	if st != nil {
		st.NewBatch(len(data))
	}

	if len(data) == 0 {
		return nil, nil
	}

	// encode events into bulk request buffer, dropping failed elements from
	// events slice
	origCount := len(data)
	data, bulkItems := bulkEncodePublishRequest(client.GetVersion(), client.index, client.pipeline, data, client.log)
	newCount := len(data)
	if st != nil && origCount > newCount {
		st.Dropped(origCount - newCount)
	}
	if newCount == 0 {
		return nil, nil
	}

	status, result, sendErr := client.Bulk("", "", nil, bulkItems)
	if sendErr != nil {
		client.log.Errorf("Failed to perform any bulk index operations: %s", sendErr)
		return data, sendErr
	}

	client.log.Debugf("PublishEvents: %d events have been published to elasticsearch in %v.",
		len(data),
		time.Now().Sub(begin))

	// check response for transient errors
	var failedEvents []publisher.Event
	var stats bulkResultStats
	if status != 200 {
		failedEvents = data
		stats.fails = len(failedEvents)
	} else {
		failedEvents, stats = bulkCollectPublishFails(result, data, client.log)
	}

	failed := len(failedEvents)
	if st := client.observer; st != nil {
		dropped := stats.nonIndexable
		duplicates := stats.duplicates
		acked := len(data) - failed - dropped - duplicates

		st.Acked(acked)
		st.Failed(failed)
		st.Dropped(dropped)
		st.Duplicate(duplicates)
		st.ErrTooMany(stats.tooMany)
	}

	if failed > 0 {
		if sendErr == nil {
			sendErr = eslegclient.ErrTempBulkFailure
		}
		return failedEvents, sendErr
	}
	return nil, nil
}

// bulkEncodePublishRequest encodes all bulk requests and returns slice of events
// successfully added to the list of bulk items and the list of bulk items.
func bulkEncodePublishRequest(
	log *logp.Logger,
	version common.Version,
	index outputs.IndexSelector,
	pipeline *outil.Selector,
	data []publisher.Event,
) ([]publisher.Event, []interface{}) {

	okEvents := data[:0]
	bulkItems := []interface{}{}
	for i := range data {
		event := &data[i].Content
		meta, err := createEventBulkMeta(log, version, index, pipeline, event)
		if err != nil {
			log.Errorf("Failed to encode event meta data: %+v", err)
			continue
		}
		bulkItems = append(bulkItems, meta, event)
		okEvents = append(okEvents, data[i])
	}
	return okEvents, bulkItems
}

func createEventBulkMeta(
	log *logp.Logger,
	version common.Version,
	indexSel outputs.IndexSelector,
	pipelineSel *outil.Selector,
	event *beat.Event,
	logger *logp.Logger,
) (interface{}, error) {
	eventType := ""
	if version.Major < 7 {
		eventType = defaultEventType
	}

	pipeline, err := getPipeline(event, pipelineSel)
	if err != nil {
		err := fmt.Errorf("failed to select pipeline: %v", err)
		return nil, err
	}

	index, err := indexSel.Select(event)
	if err != nil {
		err := fmt.Errorf("failed to select event index: %v", err)
		return nil, err
	}

	var id string
	if m := event.Meta; m != nil {
		if tmp := m["_id"]; tmp != nil {
			if s, ok := tmp.(string); ok {
				id = s
			} else {
				log.Errorf("Event ID '%v' is no string value", id)
			}
		}
	}

	meta := eslegclient.BulkMeta{
		Index:    index,
		DocType:  eventType,
		Pipeline: pipeline,
		ID:       id,
	}

	if id != "" || version.Major > 7 || (version.Major == 7 && version.Minor >= 5) {
		return eslegclient.BulkCreateAction{Create: meta}, nil
	}
	return eslegclient.BulkIndexAction{Index: meta}, nil
}

func getPipeline(event *beat.Event, pipelineSel *outil.Selector) (string, error) {
	if event.Meta != nil {
		if pipeline, exists := event.Meta["pipeline"]; exists {
			if p, ok := pipeline.(string); ok {
				return p, nil
			}
			return "", errors.New("pipeline metadata is no string")
		}
	}

	if pipelineSel != nil {
		return pipelineSel.Select(event)
	}
	return "", nil
}

// bulkCollectPublishFails checks per item errors returning all events
// to be tried again due to error code returned for that items. If indexing an
// event failed due to some error in the event itself (e.g. does not respect mapping),
// the event will be dropped.
func bulkCollectPublishFails(
	log *logp.Logger,
	result esclientleg.BulkResult,
	data []publisher.Event,
) ([]publisher.Event, bulkResultStats) {
	reader := eslegclient.NewJSONReader(result)
	if err := eslegclient.BulkReadToItems(reader); err != nil {
		log.Errorf("failed to parse bulk response: %v", err.Error())
		return nil, bulkResultStats{}
	}

	count := len(data)
	failed := data[:0]
	stats := bulkResultStats{}
	for i := 0; i < count; i++ {
		status, msg, err := eslegclient.BulkReadItemStatus(log, reader)
		if err != nil {
			log.Error(err)
			return nil, bulkResultStats{}
		}

		if status < 300 {
			stats.acked++
			continue // ok value
		}

		if status == 409 {
			// 409 is used to indicate an event with same ID already exists if
			// `create` op_type is used.
			stats.duplicates++
			continue // ok
		}

		if status < 500 {
			if status == http.StatusTooManyRequests {
				stats.tooMany++
			} else {
				// hard failure, don't collect
				log.Warnf("Cannot index event %#v (status=%v): %s", data[i], status, msg)
				stats.nonIndexable++
				continue
			}
		}

		log.Debugf("Bulk item insert failed (i=%v, status=%v): %s", i, status, msg)
		stats.fails++
		failed = append(failed, data[i])
	}

	return failed, stats
}

func (client *Client) String() string {
	return "elasticsearch(" + client.Connection.URL + ")"
}
