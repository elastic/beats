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

	indexSelector    outputs.IndexSelector
	pipelineSelector *outil.Selector

	observer outputs.Observer

	// If deadLetterIndex is set, events with bulk-ingest errors will be
	// forwarded to this index. Otherwise, they will be dropped.
	deadLetterIndex string

	log *logp.Logger
}

// clientSettings contains the settings for a client.
type clientSettings struct {
	connection       eslegclient.ConnectionSettings
	indexSelector    outputs.IndexSelector
	pipelineSelector *outil.Selector

	// The metrics observer from the clientSettings, or a no-op placeholder if
	// none is provided. This variable is always non-nil for a client created
	// via NewClient.
	observer outputs.Observer

	// If deadLetterIndex is set, events with bulk-ingest errors will be
	// forwarded to this index. Otherwise, they will be dropped.
	deadLetterIndex string
}

type bulkResultStats struct {
	acked        int // number of events ACKed by Elasticsearch
	duplicates   int // number of events failed with `create` due to ID already being indexed
	fails        int // number of events with retryable failures.
	nonIndexable int // number of events with permanent failures.
	deadLetter   int // number of failed events ingested to the dead letter index.
	tooMany      int // number of events receiving HTTP 429 Too Many Requests
}

type bulkResult struct {
	// A connection-level error if the request couldn't be sent or the response
	// couldn't be read. This error is returned from (*Client).Publish to signal
	// to the pipeline that this output worker needs to be reconnected before the
	// next Publish call.
	connErr error

	// The array of events sent via bulk request. This excludes any events that
	// had encoding errors while assembling the request.
	events []publisher.Event

	// The http status returned by the bulk request.
	status int

	// The API response from Elasticsearch.
	response eslegclient.BulkResponse
}

const (
	defaultEventType = "doc"
)

// Flags passed with the Bulk API request: we filter the response to include
// only the fields we need for checking request/item state.
var bulkRequestParams = map[string]string{
	"filter_path": "errors,items.*.error,items.*.status",
}

// NewClient instantiates a new client.
func NewClient(
	s clientSettings,
	onConnect *callbacksRegistry,
) (*Client, error) {
	pipeline := s.pipelineSelector
	if pipeline != nil && pipeline.IsEmpty() {
		pipeline = nil
	}

	conn, err := eslegclient.NewConnection(s.connection)
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

	// Make sure there's a non-nil obser
	observer := s.observer
	if observer == nil {
		observer = outputs.NewNilObserver()
	}

	client := &Client{
		conn:             *conn,
		indexSelector:    s.indexSelector,
		pipelineSelector: pipeline,
		observer:         observer,
		deadLetterIndex:  s.deadLetterIndex,

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
		clientSettings{
			connection:       connection,
			indexSelector:    client.indexSelector,
			pipelineSelector: client.pipelineSelector,
			deadLetterIndex:  client.deadLetterIndex,
		},
		nil, // XXX: do not pass connection callback?
	)
	return c
}

func (client *Client) Publish(ctx context.Context, batch publisher.Batch) error {
	span, ctx := apm.StartSpan(ctx, "publishEvents", "output")
	defer span.End()
	span.Context.SetLabel("events_original", len(batch.Events()))
	client.observer.NewBatch(len(batch.Events()))

	// Create and send the bulk request.
	bulkResult := client.doBulkRequest(ctx, batch)
	span.Context.SetLabel("events_encoded", len(bulkResult.events))
	if bulkResult.connErr != nil {
		// If there was a connection-level error there is no per-item response,
		// handle it and return.
		return client.handleBulkResultError(ctx, batch, bulkResult)
	}
	span.Context.SetLabel("events_published", len(bulkResult.events))

	// At this point we have an Elasticsearch response for our request,
	// check and report the per-item results.
	eventsToRetry, stats := client.bulkCollectPublishFails(bulkResult)
	stats.reportToObserver(client.observer)

	if len(eventsToRetry) > 0 {
		span.Context.SetLabel("events_failed", len(eventsToRetry))
		batch.RetryEvents(eventsToRetry)
	} else {
		batch.ACK()
	}
	return nil
}

// Encode a batch's events into a bulk publish request, send the request to
// Elasticsearch, and return the resulting metadata.
// Reports the network request latency to the client's metrics observer.
// The events list in the result will be shorter than the original batch if
// some events couldn't be encoded. In this case, the removed events will
// be reported to the Client's metrics observer via PermanentErrors.
func (client *Client) doBulkRequest(
	ctx context.Context,
	batch publisher.Batch,
) bulkResult {
	var result bulkResult

	rawEvents := batch.Events()

	// encode events into bulk request buffer, dropping failed elements from
	// events slice
	resultEvents, bulkItems := client.bulkEncodePublishRequest(client.conn.GetVersion(), rawEvents)
	result.events = resultEvents
	client.observer.PermanentErrors(len(rawEvents) - len(resultEvents))

	// If we encoded any events, send the network request.
	if len(result.events) > 0 {
		begin := time.Now()
		result.status, result.response, result.connErr =
			client.conn.Bulk(ctx, "", "", bulkRequestParams, bulkItems)
		if result.connErr == nil {
			duration := time.Since(begin)
			client.observer.ReportLatency(duration)
			client.log.Debugf(
				"doBulkRequest: %d events have been sent to elasticsearch in %v.",
				len(result.events), duration)
		}
	}

	return result
}

func (client *Client) handleBulkResultError(
	ctx context.Context, batch publisher.Batch, bulkResult bulkResult,
) error {
	if bulkResult.status == http.StatusRequestEntityTooLarge {
		if batch.SplitRetry() {
			// Report that we split a batch
			client.observer.BatchSplit()
			client.observer.RetryableErrors(len(bulkResult.events))
		} else {
			// If the batch could not be split, there is no option left but
			// to drop it and log the error state.
			batch.Drop()
			client.observer.PermanentErrors(len(bulkResult.events))
			client.log.Error(errPayloadTooLarge)
		}
		// Don't propagate a too-large error since it doesn't indicate a problem
		// with the connection.
		return nil
	}
	err := apm.CaptureError(ctx, fmt.Errorf("failed to perform any bulk index operations: %w", bulkResult.connErr))
	err.Send()
	client.log.Error(err)

	if len(bulkResult.events) > 0 {
		// At least some events failed, retry them
		batch.RetryEvents(bulkResult.events)
	} else {
		// All events were sent successfully
		batch.ACK()
	}
	client.observer.RetryableErrors(len(bulkResult.events))
	return bulkResult.connErr
}

// bulkEncodePublishRequest encodes all bulk requests and returns slice of events
// successfully added to the list of bulk items and the list of bulk items.
func (client *Client) bulkEncodePublishRequest(version version.V, data []publisher.Event) ([]publisher.Event, []interface{}) {
	okEvents := data[:0]
	bulkItems := []interface{}{}
	for i := range data {
		if data[i].EncodedEvent == nil {
			client.log.Error("Elasticsearch output received unencoded publisher.Event")
			continue
		}
		event := data[i].EncodedEvent.(*encodedEvent)
		if event.err != nil {
			// This means there was an error when encoding the event and it isn't
			// ingestable, so report the error and continue.
			client.log.Error(event.err)
			continue
		}
		meta, err := client.createEventBulkMeta(version, event)
		if err != nil {
			client.log.Errorf("Failed to encode event meta data: %+v", err)
			continue
		}
		if event.opType == events.OpTypeDelete {
			// We don't include the event source in a bulk DELETE
			bulkItems = append(bulkItems, meta)
		} else {
			// Wrap the encoded event in a RawEncoding so the Elasticsearch client
			// knows not to re-encode it
			bulkItems = append(bulkItems, meta, eslegclient.RawEncoding{Encoding: event.encoding})
		}
		okEvents = append(okEvents, data[i])
	}
	return okEvents, bulkItems
}

func (client *Client) createEventBulkMeta(version version.V, event *encodedEvent) (interface{}, error) {
	eventType := ""
	if version.Major < 7 {
		eventType = defaultEventType
	}

	meta := eslegclient.BulkMeta{
		Index:    event.index,
		DocType:  eventType,
		Pipeline: event.pipeline,
		ID:       event.id,
	}

	if event.opType == events.OpTypeDelete {
		if event.id != "" {
			return eslegclient.BulkDeleteAction{Delete: meta}, nil
		} else {
			return nil, fmt.Errorf("%s %s requires _id", events.FieldMetaOpType, events.OpTypeDelete)
		}
	}
	if event.id != "" || version.Major > 7 || (version.Major == 7 && version.Minor >= 5) {
		if event.opType == events.OpTypeIndex {
			return eslegclient.BulkIndexAction{Index: meta}, nil
		}
		return eslegclient.BulkCreateAction{Create: meta}, nil
	}
	return eslegclient.BulkIndexAction{Index: meta}, nil
}

func getPipeline(event *beat.Event, defaultSelector *outil.Selector) (string, error) {
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

	if defaultSelector != nil {
		return defaultSelector.Select(event)
	}
	return "", nil
}

// bulkCollectPublishFails checks per item errors returning all events
// to be tried again due to error code returned for that items. If indexing an
// event failed due to some error in the event itself (e.g. does not respect mapping),
// the event will be dropped.
// Each of the events will be reported in the returned stats as exactly one of
// acked, duplicates, fails, nonIndexable, or deadLetter.
func (client *Client) bulkCollectPublishFails(bulkResult bulkResult) ([]publisher.Event, bulkResultStats) {
	events := bulkResult.events

	if len(bulkResult.events) == 0 {
		// No events to process
		return nil, bulkResultStats{}
	}
	if bulkResult.status != 200 {
		return events, bulkResultStats{fails: len(events)}
	}
	reader := newJSONReader(bulkResult.response)
	if err := bulkReadToItems(reader); err != nil {
		client.log.Errorf("failed to parse bulk response: %v", err.Error())
		return events, bulkResultStats{fails: len(events)}
	}

	count := len(events)
	eventsToRetry := events[:0]
	stats := bulkResultStats{}
	for i := 0; i < count; i++ {
		itemStatus, itemMessage, err := bulkReadItemStatus(client.log, reader)
		if err != nil {
			// The response json is invalid, mark the remaining events for retry.
			stats.fails += count - i
			eventsToRetry = append(eventsToRetry, events[i:]...)
			break
		}

		if client.applyItemStatus(events[i], itemStatus, itemMessage, &stats) {
			eventsToRetry = append(eventsToRetry, events[i])
			client.log.Debugf("Bulk item insert failed (i=%v, status=%v): %s", i, itemStatus, itemMessage)
		}
	}

	return eventsToRetry, stats
}

// applyItemStatus processes the ingestion status of one event from a bulk request.
// Returns true if the item should be retried.
// In the provided bulkResultStats, applyItemStatus increments exactly one of:
// acked, duplicates, deadLetter, fails, nonIndexable.
func (client *Client) applyItemStatus(
	event publisher.Event,
	itemStatus int,
	itemMessage []byte,
	stats *bulkResultStats,
) bool {
	encodedEvent := event.EncodedEvent.(*encodedEvent)
	if itemStatus < 300 {
		if encodedEvent.deadLetter {
			// This was ingested into the dead letter index, not the original target
			stats.deadLetter++
		} else {
			stats.acked++
		}
		return false // no retry needed
	}

	if itemStatus == 409 {
		// 409 is used to indicate there is already an event with the same ID, or
		// with identical Time Series Data Stream dimensions when TSDS is active.
		stats.duplicates++
		return false // no retry needed
	}

	if itemStatus == http.StatusTooManyRequests {
		stats.fails++
		stats.tooMany++
		return true
	}

	if itemStatus < 500 {
		// hard failure, apply policy action
		if encodedEvent.deadLetter {
			// Fatal error while sending an already-failed event to the dead letter
			// index, drop.
			client.log.Errorf("Can't deliver to dead letter index event (status=%v). Look at the event log to view the event and cause.", itemStatus)
			client.log.Errorw(fmt.Sprintf("Can't deliver to dead letter index event %#v (status=%v): %s", event, itemStatus, itemMessage), logp.TypeKey, logp.EventType)
			stats.nonIndexable++
			return false
		}
		if client.deadLetterIndex == "" {
			// Fatal error and no dead letter index, drop.
			client.log.Warnf("Cannot index event (status=%v): dropping event! Look at the event log to view the event and cause.", itemStatus)
			client.log.Warnw(fmt.Sprintf("Cannot index event %#v (status=%v): %s, dropping event!", event, itemStatus, itemMessage), logp.TypeKey, logp.EventType)
			stats.nonIndexable++
			return false
		}
		// Send this failure to the dead letter index and "retry".
		// We count this as a "retryable failure", and then if the dead letter
		// ingestion succeeds it is counted in the "deadLetter" counter
		// rather than the "acked" counter.
		client.log.Warnf("Cannot index event (status=%v), trying dead letter index. Look at the event log to view the event and cause.", itemStatus)
		client.log.Warnw(fmt.Sprintf("Cannot index event %#v (status=%v): %s, trying dead letter index", event, itemStatus, itemMessage), logp.TypeKey, logp.EventType)
		encodedEvent.setDeadLetter(client.deadLetterIndex, itemStatus, string(itemMessage))
	}

	// Everything else gets retried.
	stats.fails++
	return true
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

func (stats bulkResultStats) reportToObserver(ob outputs.Observer) {
	ob.AckedEvents(stats.acked)
	ob.RetryableErrors(stats.fails)
	ob.PermanentErrors(stats.nonIndexable)
	ob.DuplicateEvents(stats.duplicates)
	ob.DeadLetterEvents(stats.deadLetter)

	ob.ErrTooMany(stats.tooMany)
}
