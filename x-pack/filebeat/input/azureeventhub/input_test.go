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
