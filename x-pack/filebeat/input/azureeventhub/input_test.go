// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"go.opentelemetry.io/collector/pdata/pcommon"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/elastic-agent-libs/monitoring"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/otel/otelmap"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var defaultTestConfig = azureInputConfig{
	SAKey:            "",
	SAName:           "",
	SAContainer:      ephContainerName,
	ConnectionString: "",
	ConsumerGroup:    "",
}

func TestGetAzureEnvironment(t *testing.T) {
	resMan := ""
	env, err := getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.PublicCloud)
	resMan = "https://management.microsoftazure.de/"
	env, err = getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.GermanCloud)
	resMan = "http://management.invalidhybrid.com/"
	_, err = getAzureEnvironment(resMan)
	assert.Errorf(t, err, "invalid character 'F' looking for beginning of value")
	resMan = "<no value>"
	env, err = getAzureEnvironment(resMan)
	assert.NoError(t, err)
	assert.Equal(t, env, azure.PublicCloud)
}

func TestProcessEvents(t *testing.T) {
	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))

	metrics := newInputMetrics(monitoring.NewRegistry(), logp.NewNopLogger())

	fakePipelineClient := fakeClient{}

	input := eventHubInputV1{
		config:         defaultTestConfig,
		log:            log,
		metrics:        metrics,
		pipelineClient: &fakePipelineClient,
		messageDecoder: messageDecoder{
			config:  defaultTestConfig,
			log:     log,
			metrics: metrics,
		},
	}

	var sn int64 = 12
	now := time.Now()
	var off int64 = 1234
	var pID int16 = 1

	properties := eventhub.SystemProperties{
		SequenceNumber: &sn,
		EnqueuedTime:   &now,
		Offset:         &off,
		PartitionID:    &pID,
		PartitionKey:   nil,
	}
	single := "{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"
	msg := fmt.Sprintf("{\"records\":[%s]}", single)
	ev := eventhub.Event{
		Data:             []byte(msg),
		SystemProperties: &properties,
	}
	input.processEvents(&ev)

	assert.Equal(t, len(fakePipelineClient.publishedEvents), 1)
	message, err := fakePipelineClient.publishedEvents[0].Fields.GetValue("message")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, message, single)
}

// func TestNewInputDone(t *testing.T) {
//	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))
//	config := mapstr.M{
//		"connection_string":   "Endpoint=sb://something",
//		"eventhub":            "insights-operational-logs",
//		"storage_account":     "someaccount",
//		"storage_account_key": "secret",
//	}
//	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
// }

// ackClient is a fake beat.Client that ACKs the published messages.
type fakeClient struct {
	sync.Mutex
	publishedEvents []beat.Event
}

func (c *fakeClient) Close() error { return nil }

func (c *fakeClient) Publish(event beat.Event) {
	c.Lock()
	defer c.Unlock()
	c.publishedEvents = append(c.publishedEvents, event)
}

func (c *fakeClient) PublishAll(event []beat.Event) {
	for _, e := range event {
		c.Publish(e)
	}
}

// conversionClient is a fake beat.Client that mimics the otel output in the
// beat receiver runtime: it keeps converting the first published event on a
// separate goroutine while the input goes on publishing the next ones.
//
// If the input mutates a map shared with an already published event, the
// conversion triggers "concurrent map iteration and map write" (or a data
// race report when running with -race).
type conversionClient struct {
	stop      chan struct{}
	done      chan error
	published int
}

func newConversionClient() *conversionClient {
	return &conversionClient{
		stop: make(chan struct{}),
		done: make(chan error, 1),
	}
}

func (c *conversionClient) Close() error { return nil }

func (c *conversionClient) Publish(event beat.Event) {
	c.published++
	if c.published != 1 {
		return
	}
	ready := make(chan struct{})
	go func() {
		close(ready)
		for {
			if err := otelmap.FromMapstr(pcommon.NewMap(), event.Fields); err != nil {
				c.done <- err
				return
			}
			select {
			case <-c.stop:
				c.done <- nil
				return
			default:
			}
		}
	}()
	<-ready
}

func (c *conversionClient) PublishAll(events []beat.Event) {
	for _, event := range events {
		c.Publish(event)
	}
}

// wait stops the conversion goroutine and returns its result.
func (c *conversionClient) wait() error {
	close(c.stop)
	return <-c.done
}

// assertIndependentMetadata checks that the events published from the same
// message do not share their `azure` map: changing it in one event must not
// affect the others.
func assertIndependentMetadata(t *testing.T, events []beat.Event) {
	t.Helper()
	require.Len(t, events, 2, "the message should expand into two events")

	first, ok := events[0].Fields["azure"].(mapstr.M)
	require.True(t, ok, "the first event should have an azure map")
	second, ok := events[1].Fields["azure"].(mapstr.M)
	require.True(t, ok, "the second event should have an azure map")

	want := second.Clone()
	_, _ = first.Put("offset", "changed")
	_, _ = first.Put("added", true)

	assert.Equal(t, want, second, "changing the azure map of one event should not affect the other events")
}

// recordsPayload returns an Event Hub message body with n records.
func recordsPayload(n int) []byte {
	return []byte(`{"records":[` + strings.Repeat(`{"test":"record"},`, n-1) + `{"test":"record"}]}`)
}
