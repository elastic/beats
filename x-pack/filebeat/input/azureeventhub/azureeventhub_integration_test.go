// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration && azure && !aix
// +build integration,azure,!aix

package azureeventhub

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/filebeat/channel"
	"github.com/elastic/beats/v8/filebeat/input"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
)

var (
	azureConfig = common.MustNewConfigFrom(common.MapStr{
		"storage_account_key":       lookupEnv("STORAGE_ACCOUNT_NAME"),
		"storage_account":           lookupEnv("STORAGE_ACCOUNT_KEY"),
		"storage_account_container": ephContainerName,
		"connection_string":         lookupEnv("EVENTHUB_CONNECTION_STRING"),
		"consumer_group":            lookupEnv("EVENTHUB_CONSUMERGROUP"),
		"eventhub":                  lookupEnv("EVENTHUB_NAME"),
	})

	message = "{\"records\":[{\"some_field\":\"this is some message\",\"time\":\"2019-12-17T13:43:44.4946995Z\"}"
)

func TestInput(t *testing.T) {
	err := addEventToHub(lookupEnv("EVENTHUB_CONNECTION_STRING"))
	if err != nil {
		t.Fatal(err)
	}
	context := input.Context{
		Done:     make(chan struct{}),
		BeatDone: make(chan struct{}),
	}

	o := &stubOutleter{}
	o.cond = sync.NewCond(o)
	defer o.Close()

	connector := channel.ConnectorFunc(func(_ *common.Config, _ beat.ClientConfig) (channel.Outleter, error) {
		return o, nil
	})
	input, err := NewInput(azureConfig, connector, context)
	if err != nil {
		t.Fatal(err)
	}

	// Run the input and wait for finalization
	input.Run()

	timeout := time.After(30 * time.Second)
	// Route input events through our capturer instead of sending through ES.
	events := make(chan beat.Event, 100)
	defer close(events)

	select {
	case event := <-events:
		text, err := event.Fields.GetValue("message")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, text, message)

	case <-timeout:
		t.Fatal("timeout waiting for incoming events")
	}

	// Close the done channel and make sure the beat shuts down in a reasonable
	// amount of time.
	close(context.Done)
	didClose := make(chan struct{})
	go func() {
		input.Wait()
		close(didClose)
	}()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for beat to shut down")
	case <-didClose:
	}
}

func lookupEnv(t *testing.T, varName string) string {
	value, ok := os.LookupEnv(varName)
	if !ok {
		t.Fatalf("Environment variable %s is not set", varName)
	}
	return value
}

func addEventToHub(connStr string) error {
	hub, err := eventhub.NewHubFromConnectionString(connStr)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	// send a single message into a random partition
	err = hub.Send(ctx, eventhub.NewEventFromString(message))
	if err != nil {
		return err
	}
	hub.Close(ctx)
	defer cancel()
	return nil
}
