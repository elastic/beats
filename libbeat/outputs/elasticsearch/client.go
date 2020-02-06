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
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/esclientleg"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/beats/v7/libbeat/testing"
)

// Client is an elasticsearch client.
type Client struct {
	esclientleg.Connection
	tlsConfig *tlscommon.TLSConfig

	index    outputs.IndexSelector
	pipeline *outil.Selector
	params   map[string]string
	timeout  time.Duration

	// buffered bulk requests
	bulkRequ *esclientleg.BulkRequest

	// buffered json response reader
	json JSONReader

	// additional configs
	compressionLevel int
	proxyURL         *url.URL

	observer outputs.Observer
}

// ClientSettings contains the settings for a client.
type ClientSettings struct {
	URL                string
	Proxy              *url.URL
	ProxyDisable       bool
	TLS                *tlscommon.TLSConfig
	Username, Password string
	APIKey             string
	EscapeHTML         bool
	Parameters         map[string]string
	Headers            map[string]string
	Index              outputs.IndexSelector
	Pipeline           *outil.Selector
	Timeout            time.Duration
	CompressionLevel   int
	Observer           outputs.Observer
}

// ConnectCallback defines the type for the function to be called when the Elasticsearch client successfully connects to the cluster
type ConnectCallback func(client *Client) error

type bulkIndexAction struct {
	Index bulkEventMeta `json:"index" struct:"index"`
}

type bulkCreateAction struct {
	Create bulkEventMeta `json:"create" struct:"create"`
}

type bulkEventMeta struct {
	Index    string `json:"_index" struct:"_index"`
	DocType  string `json:"_type,omitempty" struct:"_type,omitempty"`
	Pipeline string `json:"pipeline,omitempty" struct:"pipeline,omitempty"`
	ID       string `json:"_id,omitempty" struct:"_id,omitempty"`
}

type bulkResultStats struct {
	acked        int // number of events ACKed by Elasticsearch
	duplicates   int // number of events failed with `create` due to ID already being indexed
	fails        int // number of failed events (can be retried)
	nonIndexable int // number of failed events (not indexable -> must be dropped)
	tooMany      int // number of events receiving HTTP 429 Too Many Requests
}

var (
	nameItems  = []byte("items")
	nameStatus = []byte("status")
	nameError  = []byte("error")
)

var (
	errExpectedItemsArray    = errors.New("expected items array")
	errExpectedItemObject    = errors.New("expected item response object")
	errExpectedStatusCode    = errors.New("expected item status code")
	errUnexpectedEmptyObject = errors.New("empty object")
	errExpectedObjectEnd     = errors.New("expected end of object")
	errTempBulkFailure       = errors.New("temporary bulk send failure")
)

const (
	defaultEventType = "doc"
)

// NewClient instantiates a new client.
func NewClient(
	s ClientSettings,
	onConnect *callbacksRegistry,
) (*Client, error) {
	var proxy func(*http.Request) (*url.URL, error)
	if !s.ProxyDisable {
		proxy = http.ProxyFromEnvironment
		if s.Proxy != nil {
			proxy = http.ProxyURL(s.Proxy)
		}
	}

	pipeline := s.Pipeline
	if pipeline != nil && pipeline.IsEmpty() {
		pipeline = nil
	}

	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse elasticsearch URL: %v", err)
	}
	if u.User != nil {
		s.Username = u.User.Username()
		s.Password, _ = u.User.Password()
		u.User = nil

		// Re-write URL without credentials.
		s.URL = u.String()
	}

	log := logp.NewLogger(logSelector)
	log.Infof("Elasticsearch url: %s", s.URL)

	// TODO: add socks5 proxy support
	var dialer, tlsDialer transport.Dialer

	dialer = transport.NetDialer(s.Timeout)
	tlsDialer, err = transport.TLSDialer(dialer, s.TLS, s.Timeout)
	if err != nil {
		return nil, err
	}

	if st := s.Observer; st != nil {
		dialer = transport.StatsDialer(dialer, st)
		tlsDialer = transport.StatsDialer(tlsDialer, st)
	}

	params := s.Parameters
	bulkRequ, err := esclientleg.NewBulkRequest(s.URL, "", "", params, nil)
	if err != nil {
		return nil, err
	}

	var encoder esclientleg.BodyEncoder
	compression := s.CompressionLevel
	if compression == 0 {
		encoder = esclientleg.NewJSONEncoder(nil, s.EscapeHTML)
	} else {
		encoder, err = esclientleg.NewGzipEncoder(compression, nil, s.EscapeHTML)
		if err != nil {
			return nil, err
		}
	}

	client := &Client{
		Connection: esclientleg.Connection{
			URL:      s.URL,
			Username: s.Username,
			Password: s.Password,
			APIKey:   base64.StdEncoding.EncodeToString([]byte(s.APIKey)),
			Headers:  s.Headers,
			HTTP: &http.Client{
				Transport: &http.Transport{
					Dial:            dialer.Dial,
					DialTLS:         tlsDialer.Dial,
					TLSClientConfig: s.TLS.ToConfig(),
					Proxy:           proxy,
				},
				Timeout: s.Timeout,
			},
			Encoder: encoder,
		},
		tlsConfig: s.TLS,
		index:     s.Index,
		pipeline:  pipeline,
		params:    params,
		timeout:   s.Timeout,

		bulkRequ: bulkRequ,

		compressionLevel: compression,
		proxyURL:         s.Proxy,
		observer:         s.Observer,
	}

	client.Connection.OnConnectCallback = func() error {
		globalCallbackRegistry.mutex.Lock()
		defer globalCallbackRegistry.mutex.Unlock()

		for _, callback := range globalCallbackRegistry.callbacks {
			err := callback(client)
			if err != nil {
				return err
			}
		}

		if onConnect != nil {
			onConnect.mutex.Lock()
			defer onConnect.mutex.Unlock()

			for _, callback := range onConnect.callbacks {
				err := callback(client)
				if err != nil {
					return err
				}
			}
		}
		return nil
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
			URL:      client.URL,
			Index:    client.index,
			Pipeline: client.pipeline,
			Proxy:    client.proxyURL,
			// Without the following nil check on proxyURL, a nil Proxy field will try
			// reloading proxy settings from the environment instead of leaving them
			// empty.
			ProxyDisable:     client.proxyURL == nil,
			TLS:              client.tlsConfig,
			Username:         client.Username,
			Password:         client.Password,
			APIKey:           client.APIKey,
			Parameters:       nil, // XXX: do not pass params?
			Headers:          client.Headers,
			Timeout:          client.HTTP.Timeout,
			CompressionLevel: client.compressionLevel,
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

	body := client.Encoder
	body.Reset()

	// encode events into bulk request buffer, dropping failed elements from
	// events slice

	eventType := ""
	if client.GetVersion().Major < 7 {
		eventType = defaultEventType
	}

	origCount := len(data)
	data = bulkEncodePublishRequest(client.Connection.log, client.GetVersion(), body, client.index, client.pipeline, eventType, data)
	newCount := len(data)
	if st != nil && origCount > newCount {
		st.Dropped(origCount - newCount)
	}
	if newCount == 0 {
		return nil, nil
	}

	requ := client.bulkRequ
	requ.Reset(body)
	status, result, sendErr := client.SendBulkRequest(requ)
	if sendErr != nil {
		client.Connection.log.Error("Failed to perform any bulk index operations: %+v", sendErr)
		return data, sendErr
	}

	client.Connection.log.Debugf("PublishEvents: %d events have been published to elasticsearch in %v.",
		len(data),
		time.Now().Sub(begin))

	// check response for transient errors
	var failedEvents []publisher.Event
	var stats bulkResultStats
	if status != 200 {
		failedEvents = data
		stats.fails = len(failedEvents)
	} else {
		client.json.init(result)
		failedEvents, stats = bulkCollectPublishFails(client.Connection.log, &client.json, data)
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
			sendErr = errTempBulkFailure
		}
		return failedEvents, sendErr
	}
	return nil, nil
}

// fillBulkRequest encodes all bulk requests and returns slice of events
// successfully added to bulk request.
func bulkEncodePublishRequest(
	log *logp.Logger,
	version common.Version,
	body esclientleg.BulkWriter,
	index outputs.IndexSelector,
	pipeline *outil.Selector,
	eventType string,
	data []publisher.Event,
) []publisher.Event {
	okEvents := data[:0]
	for i := range data {
		event := &data[i].Content
		meta, err := createEventBulkMeta(log, version, index, pipeline, eventType, event)
		if err != nil {
			log.Errorf("Failed to encode event meta data: %+v", err)
			continue
		}
		if err := body.Add(meta, event); err != nil {
			log.Errorf("Failed to encode event: %+v", err)
			log.Debugf("Failed event: %v", event)
			continue
		}
		okEvents = append(okEvents, data[i])
	}
	return okEvents
}

func createEventBulkMeta(
	log *logp.Logger,
	version common.Version,
	indexSel outputs.IndexSelector,
	pipelineSel *outil.Selector,
	eventType string,
	event *beat.Event,
) (interface{}, error) {
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

	meta := bulkEventMeta{
		Index:    index,
		DocType:  eventType,
		Pipeline: pipeline,
		ID:       id,
	}

	if id != "" || version.Major > 7 || (version.Major == 7 && version.Minor >= 5) {
		return bulkCreateAction{meta}, nil
	}
	return bulkIndexAction{meta}, nil
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
	reader *JSONReader,
	data []publisher.Event,
) ([]publisher.Event, bulkResultStats) {
	if err := BulkReadToItems(reader); err != nil {
		log.Errorf("failed to parse bulk response: %+v", err)
		return nil, bulkResultStats{}
	}

	count := len(data)
	failed := data[:0]
	stats := bulkResultStats{}
	for i := 0; i < count; i++ {
		status, msg, err := BulkReadItemStatus(log, reader)
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

// BulkReadToItems reads the bulk response up to (but not including) items
func BulkReadToItems(reader *JSONReader) error {
	if err := reader.ExpectDict(); err != nil {
		return errExpectedObject
	}

	// find 'items' field in response
	for {
		kind, name, err := reader.nextFieldName()
		if err != nil {
			return err
		}

		if kind == dictEnd {
			return errExpectedItemsArray
		}

		// found items array -> continue
		if bytes.Equal(name, nameItems) {
			break
		}

		reader.ignoreNext()
	}

	// check items field is an array
	if err := reader.ExpectArray(); err != nil {
		return errExpectedItemsArray
	}

	return nil
}

// BulkReadItemStatus reads the status and error fields from the bulk item
func BulkReadItemStatus(log *logp.Logger, reader *JSONReader) (int, []byte, error) {
	// skip outer dictionary
	if err := reader.ExpectDict(); err != nil {
		return 0, nil, errExpectedItemObject
	}

	// find first field in outer dictionary (e.g. 'create')
	kind, _, err := reader.nextFieldName()
	parserErr := func(err error) error {
		return errors.Wrapf(err, "Failed to parse bulk response item")
	}
	if err != nil {
		return 0, nil, parserErr(err)
	}
	if kind == dictEnd {
		err = errUnexpectedEmptyObject
		return 0, nil, parserErr(err)
	}

	// parse actual item response code and error message
	status, msg, err := itemStatusInner(log, reader)
	if err != nil {
		return 0, nil, parserErr(err)
	}

	// close dictionary. Expect outer dictionary to have only one element
	kind, _, err = reader.step()
	if err != nil {
		return 0, nil, parserErr(err)
	}
	if kind != dictEnd {
		err = errExpectedObjectEnd
		return 0, nil, parserErr(err)
	}

	return status, msg, nil
}

func itemStatusInner(log *logp.Logger, reader *JSONReader) (int, []byte, error) {
	if err := reader.ExpectDict(); err != nil {
		return 0, nil, errExpectedItemObject
	}

	status := -1
	var msg []byte
	for {
		kind, name, err := reader.nextFieldName()
		if err != nil {
			log.Errorf("Failed to parse bulk response item: %+v", err)
		}
		if kind == dictEnd {
			break
		}

		switch {
		case bytes.Equal(name, nameStatus): // name == "status"
			status, err = reader.nextInt()
			if err != nil {
				return 0, nil, err
			}

		case bytes.Equal(name, nameError): // name == "error"
			msg, err = reader.ignoreNext() // collect raw string for "error" field
			if err != nil {
				return 0, nil, err
			}

		default: // ignore unknown fields
			_, err = reader.ignoreNext()
			if err != nil {
				return 0, nil, err
			}
		}
	}

	if status < 0 {
		return 0, nil, errExpectedStatusCode
	}
	return status, msg, nil
}

// LoadJSON creates a PUT request based on a JSON document.
func (client *Client) LoadJSON(path string, json map[string]interface{}) ([]byte, error) {
	status, body, err := client.Request("PUT", path, "", nil, json)
	if err != nil {
		return body, fmt.Errorf("couldn't load json. Error: %s", err)
	}
	if status > 300 {
		return body, fmt.Errorf("couldn't load json. Status: %v", status)
	}

	return body, nil
}

func (client *Client) Test(d testing.Driver) {
	d.Run("elasticsearch: "+client.URL, func(d testing.Driver) {
		u, err := url.Parse(client.URL)
		d.Fatal("parse url", err)

		address := u.Host

		d.Run("connection", func(d testing.Driver) {
			netDialer := transport.TestNetDialer(d, client.timeout)
			_, err = netDialer.Dial("tcp", address)
			d.Fatal("dial up", err)
		})

		if u.Scheme != "https" {
			d.Warn("TLS", "secure connection disabled")
		} else {
			d.Run("TLS", func(d testing.Driver) {
				netDialer := transport.NetDialer(client.timeout)
				tlsDialer, err := transport.TestTLSDialer(d, netDialer, client.tlsConfig, client.timeout)
				_, err = tlsDialer.Dial("tcp", address)
				d.Fatal("dial up", err)
			})
		}

		err = client.Connect()
		d.Fatal("talk to server", err)
		version := client.GetVersion()
		d.Info("version", version.String())
	})
}

func (client *Client) String() string {
	return "elasticsearch(" + client.Connection.URL + ")"
}
