// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureeventhub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
)

const (
	eventHubConnector        = ";EntityPath="
	expandEventListFromField = "records"
)

// azureInput struct for the azure-eventhub input
type azureInput struct {
	config       azureInputConfig // azure-eventhub configuration
	context      input.Context
	outlet       channel.Outleter
	log          *logp.Logger            // logging info and error messages
	workerCtx    context.Context         // worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc      // used to signal that the worker should stop.
	workerOnce   sync.Once               // guarantees that the worker goroutine is only started once.
	workerWg     sync.WaitGroup          // waits on worker goroutine.
	processor    *eph.EventProcessorHost // eph will be assigned if users have enabled the option
	hub          *eventhub.Hub           // hub will be assigned
}

const (
	inputName = "azure-eventhub"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(errors.Wrapf(err, "failed to register %v input", inputName))
	}
}

// NewInput creates a new azure-eventhub input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	var config azureInputConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "reading %s input config", inputName)
	}

	inputCtx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputContext.Done:
		case <-inputCtx.Done():
		}
	}()

	// If the input ever needs to be made restartable, then context would need
	// to be recreated with each restart.
	workerCtx, workerCancel := context.WithCancel(inputCtx)

	in := &azureInput{
		config:       config,
		log:          logp.NewLogger(fmt.Sprintf("%s input", inputName)).With("connection string", stripConnectionString(config.ConnectionString)),
		context:      inputContext,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}
	out, err := connector.Connect(cfg)
	if err != nil {
		return nil, err
	}
	in.outlet = out
	in.log.Infof("Initialized %s input.", inputName)
	return in, nil
}

// Run starts the input worker then returns. Only the first invocation
// will ever start the worker.
func (a *azureInput) Run() {
	a.workerOnce.Do(func() {
		a.workerWg.Add(1)
		go func() {
			a.log.Infof("%s input worker has started.", inputName)
			defer a.log.Infof("%s input worker has stopped.", inputName)
			defer a.workerWg.Done()
			defer a.workerCancel()
			err := a.runWithEPH()
			if err != nil {
				a.log.Error(err)
				return
			}
		}()
	})
}

// run will run the input with the non-eph version, this option will be available once a more reliable storage is in place, it is curently using an in-memory storage
//func (a *azureInput) run() error {
//	var err error
//	a.hub, err = eventhub.NewHubFromConnectionString(fmt.Sprintf("%s%s%s", a.config.ConnectionString, eventHubConnector, a.config.EventHubName))
//	if err != nil {
//		return err
//	}
//	// listen to each partition of the Event Hub
//	runtimeInfo, err := a.hub.GetRuntimeInformation(a.workerCtx)
//	if err != nil {
//		return err
//	}
//
//	for _, partitionID := range runtimeInfo.PartitionIDs {
//		// Start receiving messages
//		handler := func(c context.Context, event *eventhub.Event) error {
//			a.log.Info(string(event.Data))
//			return a.processEvents(event, partitionID)
//		}
//		var err error
//		// sending a nill ReceiveOption will throw an exception
//		if a.config.ConsumerGroup != "" {
//			_, err = a.hub.Receive(a.workerCtx, partitionID, handler, eventhub.ReceiveWithConsumerGroup(a.config.ConsumerGroup))
//		} else {
//			_, err = a.hub.Receive(a.workerCtx, partitionID, handler)
//		}
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

// Stop stops TCP server
func (a *azureInput) Stop() {
	if a.hub != nil {
		err := a.hub.Close(a.workerCtx)
		if err != nil {
			a.log.Errorw(fmt.Sprintf("error while closing eventhub"), "error", err)
		}
	}
	if a.processor != nil {
		err := a.processor.Close(a.workerCtx)
		if err != nil {
			a.log.Errorw(fmt.Sprintf("error while closing eventhostprocessor"), "error", err)
		}
	}
	a.workerCancel()
	a.workerWg.Wait()
}

// Wait stop the current server
func (a *azureInput) Wait() {
	a.Stop()
}

func (a *azureInput) processEvents(event *eventhub.Event, partitionID string) bool {
	timestamp := time.Now()
	azure := mapstr.M{
		// partitionID is only mapped in the non-eph option which is not available yet, this field will be temporary unavailable
		//"partition_id":   partitionID,
		"eventhub":       a.config.EventHubName,
		"consumer_group": a.config.ConsumerGroup,
	}
	messages := a.parseMultipleMessages(event.Data)
	for _, msg := range messages {
		azure.Put("offset", event.SystemProperties.Offset)
		azure.Put("sequence_number", event.SystemProperties.SequenceNumber)
		azure.Put("enqueued_time", event.SystemProperties.EnqueuedTime)
		ok := a.outlet.OnEvent(beat.Event{
			Timestamp: timestamp,
			Fields: mapstr.M{
				"message": msg,
				"azure":   azure,
			},
			Private: event.Data,
		})
		if !ok {
			return ok
		}
	}
	return true
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (a *azureInput) parseMultipleMessages(bMessage []byte) []string {
	var mapObject map[string][]interface{}
	var messages []string
	// check if the message is a "records" object containing a list of events
	err := json.Unmarshal(bMessage, &mapObject)
	if err == nil {
		if len(mapObject[expandEventListFromField]) > 0 {
			for _, ms := range mapObject[expandEventListFromField] {
				js, err := json.Marshal(ms)
				if err == nil {
					messages = append(messages, string(js))
				} else {
					a.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
				}
			}
		}
	} else {
		a.log.Debugf("deserializing multiple messages to a `records` object returning error: %s", err)
		// in some cases the message is an array
		var arrayObject []interface{}
		err = json.Unmarshal(bMessage, &arrayObject)
		if err != nil {
			// return entire message
			a.log.Debugf("deserializing multiple messages to an array returning error: %s", err)
			return []string{string(bMessage)}
		}
		for _, ms := range arrayObject {
			js, err := json.Marshal(ms)
			if err == nil {
				messages = append(messages, string(js))
			} else {
				a.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}
	return messages
}

// Strip connection string to remove sensitive information
// A connection string should look like this:
// Endpoint=sb://dummynamespace.servicebus.windows.net/;SharedAccessKeyName=DummyAccessKeyName;SharedAccessKey=5dOntTRytoC24opYThisAsit3is2B+OGY1US/fuL3ly=
// This code will remove everything after ';' so key information is stripped
func stripConnectionString(c string) string {
	if parts := strings.SplitN(c, ";", 2); len(parts) == 2 {
		return parts[0]
	}

	// We actually expect the string to have the documented format
	// if we reach here something is wrong, so let's stay on the safe side
	return "(redacted)"
}
