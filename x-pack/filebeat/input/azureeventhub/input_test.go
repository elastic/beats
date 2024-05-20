// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/elastic/elastic-agent-libs/monitoring"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
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

	reg := monitoring.NewRegistry()
	metrics := newInputMetrics("test", reg)
	defer metrics.Close()

	fakePipelineClient := fakeClient{}

	input := eventHubInputV1{
		config:         defaultTestConfig,
		log:            log,
		metrics:        metrics,
		pipelineClient: &fakePipelineClient,
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
	ok := input.processEvents(&ev, "0")
	if !ok {
		t.Fatal("OnEvent function returned false")
	}

	assert.Equal(t, len(fakePipelineClient.publishedEvents), 1)
	message, err := fakePipelineClient.publishedEvents[0].Fields.GetValue("message")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, message, single)
}

func TestParseMultipleRecords(t *testing.T) {
	// records object
	msg := "{\"records\":[{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]}"
	msgs := []string{
		"{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
		"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}",
	}

	reg := monitoring.NewRegistry()
	metrics := newInputMetrics("test", reg)
	defer metrics.Close()

	fakePipelineClient := fakeClient{}

	input := eventHubInputV1{
		config:         azureInputConfig{},
		log:            logp.NewLogger(fmt.Sprintf("%s test for input", inputName)),
		metrics:        metrics,
		pipelineClient: &fakePipelineClient,
	}

	messages := input.unpackRecords([]byte(msg))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}

	// array of events
	msg1 := "[{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is 2nd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}," +
		"{\"test\":\"this is 3rd message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}]"
	messages = input.unpackRecords([]byte(msg1))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 3)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}

	// one event only
	msg2 := "{\"test\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"
	messages = input.unpackRecords([]byte(msg2))
	assert.NotNil(t, messages)
	assert.Equal(t, len(messages), 1)
	for _, ms := range messages {
		assert.Contains(t, msgs, ms)
	}
}

//func TestNewInputDone(t *testing.T) {
//	log := logp.NewLogger(fmt.Sprintf("%s test for input", inputName))
//	config := mapstr.M{
//		"connection_string":   "Endpoint=sb://something",
//		"eventhub":            "insights-operational-logs",
//		"storage_account":     "someaccount",
//		"storage_account_key": "secret",
//	}
//	inputtest.AssertNotStartedInputCanBeDone(t, NewInput, &config)
//}

func TestStripConnectionString(t *testing.T) {
	tests := []struct {
		connectionString, expected string
	}{
		{
			"Endpoint=sb://something",
			"(redacted)",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
		{
			"Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=",
			"Endpoint=sb://dummynamespace.servicebus.windows.net/",
		},
	}

	for _, tt := range tests {
		res := stripConnectionString(tt.connectionString)
		assert.Equal(t, res, tt.expected)
	}
}

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
