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
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.elastic.co/apm/v2"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/beats/v7/libbeat/publisher"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/testing"
	"github.com/elastic/elastic-agent-libs/version"
)

var (
	errPayloadTooLarge = errors.New("the bulk payload is too large for the server. Consider to adjust `http.max_content_length` parameter in Elasticsearch or `bulk_max_size` in the beat. The batch has been dropped")

	ErrTooOld = errors.New("Elasticsearch is too old. Please upgrade the instance. If you would like to connect to older instances set output.elasticsearch.allow_older_versions to true.")
)

// Client is an elasticsearch client.
type Client struct {
	conn eslegclient.Connection

	index    outputs.IndexSelector
	pipeline *outil.Selector

	observer           outputs.Observer
	NonIndexableAction string

	log *logp.Logger
}

// ClientSettings contains the settings for a client.
type ClientSettings struct {
	eslegclient.ConnectionSettings
	Index              outputs.IndexSelector
	Pipeline           *outil.Selector
	Observer           outputs.Observer
	NonIndexableAction string
}

type bulkResultStats struct {
	acked        int // number of events ACKed by Elasticsearch
	duplicates   int // number of events failed with `create` due to ID already being indexed
	fails        int // number of failed events (can be retried)
	nonIndexable int // number of failed events (not indexable)
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
		Beatname:         s.Beatname,
		Username:         s.Username,
		Password:         s.Password,
		APIKey:           s.APIKey,
		Headers:          s.Headers,
		Kerberos:         s.Kerberos,
		Observer:         s.Observer,
		Parameters:       s.Parameters,
		CompressionLevel: s.CompressionLevel,
		EscapeHTML:       s.EscapeHTML,
		Transport:        s.Transport,
		IdleConnTimeout:  s.IdleConnTimeout,
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
		conn:               *conn,
		index:              s.Index,
		pipeline:           pipeline,
		observer:           s.Observer,
		NonIndexableAction: s.NonIndexableAction,

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
	connection := eslegclient.ConnectionSettings{
		URL:               client.conn.URL,
		Beatname:          client.conn.Beatname,
		Kerberos:          client.conn.Kerberos,
		Username:          client.conn.Username,
		Password:          client.conn.Password,
		APIKey:            client.conn.APIKey,
		Parameters:        nil, // XXX: do not pass params?
		Headers:           client.conn.Headers,
		CompressionLevel:  client.conn.CompressionLevel,
		OnConnectCallback: nil,
		Observer:          nil,
		EscapeHTML:        false,
		Transport:         client.conn.Transport,
	}

	// Without the following nil check on proxyURL, a nil Proxy field will try
	// reloading proxy settings from the environment instead of leaving them
	// empty.
	client.conn.Transport.Proxy.Disable = client.conn.Transport.Proxy.URL == nil

	c, _ := NewClient(
		ClientSettings{
			ConnectionSettings: connection,
			Index:              client.index,
			Pipeline:           client.pipeline,
			NonIndexableAction: client.NonIndexableAction,
		},
		nil, // XXX: do not pass connection callback?
	)
	return c
}

func (client *Client) Publish(ctx context.Context, batch publisher.Batch) error {
	events := batch.Events()
	rest, err := client.publishEvents(ctx, events)

	switch {
	case errors.Is(err, errPayloadTooLarge):
		if batch.SplitRetry() {
			// Report that we split a batch
			client.observer.Split()
		} else {
			// If the batch could not be split, there is no option left but
			// to drop it and log the error state.
			batch.Drop()
			client.observer.Dropped(len(events))
			err := apm.CaptureError(ctx, fmt.Errorf("failed to perform bulk index operation: %w", err))
			err.Send()
			client.log.Error(err)
		}
		// Returning an error from Publish forces a client close / reconnect,
		// so don't pass this error through since it doesn't indicate anything
		// wrong with the connection.
		return nil
	case len(rest) == 0:
		batch.ACK()
	default:
		batch.RetryEvents(rest)
	}
	return err
}

// PublishEvents sends all events to elasticsearch. On error a slice with all
// events not published or confirmed to be processed by elasticsearch will be
// returned. The input slice backing memory will be reused by return the value.
func (client *Client) publishEvents(ctx context.Context, data []publisher.Event) ([]publisher.Event, error) {
	span, ctx := apm.StartSpan(ctx, "publishEvents", "output")
	defer span.End()

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
	span.Context.SetLabel("events_original", origCount)
	data, bulkItems := client.bulkEncodePublishRequest(client.conn.GetVersion(), data)
	newCount := len(data)
	span.Context.SetLabel("events_encoded", newCount)
	if st != nil && origCount > newCount {
		st.Dropped(origCount - newCount)
	}
	if newCount == 0 {
		return nil, nil
	}

	begin := time.Now()
	params := map[string]string{"filter_path": "errors,items.*.error,items.*.status"}
	status, result, sendErr := client.conn.Bulk(ctx, "", "", params, bulkItems)
	timeSinceSend := time.Since(begin)

	if sendErr != nil {
		if status == http.StatusRequestEntityTooLarge {
			// This error must be handled by splitting the batch, propagate it
			// back to Publish instead of reporting it directly
			return data, errPayloadTooLarge
		}
		err := apm.CaptureError(ctx, fmt.Errorf("failed to perform any bulk index operations: %w", sendErr))
		err.Send()
		client.log.Error(err)
		return data, sendErr
	}
	pubCount := len(data)
	span.Context.SetLabel("events_published", pubCount)

	client.log.Debugf("PublishEvents: %d events have been published to elasticsearch in %v.",
		pubCount,
		timeSinceSend)

	// check response for transient errors
	var failedEvents []publisher.Event
	var stats bulkResultStats
	if status != 200 {
		failedEvents = data
		stats.fails = len(failedEvents)
	} else {
		failedEvents, stats = client.bulkCollectPublishFails(result, data)
	}

	failed := len(failedEvents)
	span.Context.SetLabel("events_failed", failed)
	if st := client.observer; st != nil {
		dropped := stats.nonIndexable
		duplicates := stats.duplicates
		acked := len(data) - failed - dropped - duplicates

		st.Acked(acked)
		st.Failed(failed)
		st.Dropped(dropped)
		st.Duplicate(duplicates)
		st.ErrTooMany(stats.tooMany)
		st.ReportLatency(timeSinceSend)

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
func (client *Client) bulkEncodePublishRequest(version version.V, data []publisher.Event) ([]publisher.Event, []interface{}) {
	okEvents := data[:0]
	bulkItems := []interface{}{}
	for i := range data {
		event := &data[i].Content
		meta, err := client.createEventBulkMeta(version, event)
		if err != nil {
			client.log.Errorf("Failed to encode event meta data: %+v", err)
			continue
		}
		if opType := events.GetOpType(*event); opType == events.OpTypeDelete {
			// We don't include the event source in a bulk DELETE
			bulkItems = append(bulkItems, meta)
		} else if data[i].CachedEncoding != nil {
			// If the event has already been encoded, use that
			bulkItems = append(bulkItems, meta,
				eslegclient.EncodingCache{Encoding: data[i].CachedEncoding})
		} else {
			bulkItems = append(bulkItems, meta, event)
		}
		okEvents = append(okEvents, data[i])
	}
	return okEvents, bulkItems
}

func (client *Client) createEventBulkMeta(version version.V, event *beat.Event) (interface{}, error) {
	eventType := ""
	if version.Major < 7 {
		eventType = defaultEventType
	}

	pipeline, err := client.getPipeline(event)
	if err != nil {
		err := fmt.Errorf("failed to select pipeline: %w", err)
		return nil, err
	}

	index, err := client.index.Select(event)
	if err != nil {
		err := fmt.Errorf("failed to select event index: %w", err)
		return nil, err
	}

	id, _ := events.GetMetaStringValue(*event, events.FieldMetaID)
	opType := events.GetOpType(*event)

	meta := eslegclient.BulkMeta{
		Index:    index,
		DocType:  eventType,
		Pipeline: pipeline,
		ID:       id,
	}

	if opType == events.OpTypeDelete {
		if id != "" {
			return eslegclient.BulkDeleteAction{Delete: meta}, nil
		} else {
			return nil, fmt.Errorf("%s %s requires _id", events.FieldMetaOpType, events.OpTypeDelete)
		}
	}
	if id != "" || version.Major > 7 || (version.Major == 7 && version.Minor >= 5) {
		if opType == events.OpTypeIndex {
			return eslegclient.BulkIndexAction{Index: meta}, nil
		}
		return eslegclient.BulkCreateAction{Create: meta}, nil
	}
	return eslegclient.BulkIndexAction{Index: meta}, nil
}

func (client *Client) getPipeline(event *beat.Event) (string, error) {
	if event.Meta != nil {
		pipeline, err := events.GetMetaStringValue(*event, events.FieldMetaPipeline)
		if errors.Is(err, mapstr.ErrKeyNotFound) {
			return "", nil
		}
		if err != nil {
			return "", errors.New("pipeline metadata is no string")
		}

		return strings.ToLower(pipeline), nil
	}

	if client.pipeline != nil {
		return client.pipeline.Select(event)
	}
	return "", nil
}

// bulkCollectPublishFails checks per item errors returning all events
// to be tried again due to error code returned for that items. If indexing an
// event failed due to some error in the event itself (e.g. does not respect mapping),
// the event will be dropped.
func (client *Client) bulkCollectPublishFails(result eslegclient.BulkResult, data []publisher.Event) ([]publisher.Event, bulkResultStats) {
	reader := newJSONReader(result)
	if err := bulkReadToItems(reader); err != nil {
		client.log.Errorf("failed to parse bulk response: %v", err.Error())
		return nil, bulkResultStats{}
	}

	count := len(data)
	failed := data[:0]
	stats := bulkResultStats{}
	for i := 0; i < count; i++ {
		status, msg, err := bulkReadItemStatus(client.log, reader)
		if err != nil {
			client.log.Error(err)
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
				// hard failure, apply policy action
				result, _ := data[i].Content.Meta.HasKey(dead_letter_marker_field)
				if result {
					stats.nonIndexable++
					client.log.Errorf("Can't deliver to dead letter index event (status=%v). Enable debug logs to view the event and cause.", status)
					client.log.Debugf("Can't deliver to dead letter index event %#v (status=%v): %s", data[i], status, msg)
					// poison pill - this will clog the pipeline if the underlying failure is non transient.
				} else if client.NonIndexableAction == dead_letter_index {
					client.log.Warnf("Cannot index event (status=%v), trying dead letter index. Enable debug logs to view the event and cause.", status)
					client.log.Debugf("Cannot index event %#v (status=%v): %s, trying dead letter index", data[i], status, msg)
					if data[i].Content.Meta == nil {
						data[i].Content.Meta = mapstr.M{
							dead_letter_marker_field: true,
						}
					} else {
						data[i].Content.Meta[dead_letter_marker_field] = true
					}
					data[i].Content.Fields = mapstr.M{
						"message":       data[i].Content.Fields.String(),
						"error.type":    status,
						"error.message": string(msg),
					}
				} else { // drop
					stats.nonIndexable++
					client.log.Warnf("Cannot index event (status=%v): dropping event! Enable debug logs to view the event and cause.", status)
					client.log.Debugf("Cannot index event %#v (status=%v): %s, dropping event!", data[i], status, msg)
					continue
				}
			}
		}

		client.log.Debugf("Bulk item insert failed (i=%v, status=%v): %s", i, status, msg)
		stats.fails++
		failed = append(failed, data[i])
	}

	return failed, stats
}

func (client *Client) Connect() error {
	return client.conn.Connect()
}

func (client *Client) Close() error {
	return client.conn.Close()
}

func (client *Client) String() string {
	return "elasticsearch(" + client.conn.URL + ")"
}

func (client *Client) Test(d testing.Driver) {
	client.conn.Test(d)
}
