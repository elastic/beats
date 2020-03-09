// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"

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
	cfgwarn.Beta("The %s input is beta", inputName)
	var config azureInputConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "reading %s input config", inputName)
	}
	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: inputContext.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
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

	input := &azureInput{
		config:       config,
		log:          logp.NewLogger(fmt.Sprintf("%s input", inputName)).With("connection string", config.ConnectionString),
		outlet:       out,
		context:      inputContext,
		workerCtx:    workerCtx,
		workerCancel: workerCancel,
	}

	input.log.Infof("Initialized %s input.", inputName)
	return input, nil
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

func (a *azureInput) processEvents(event *eventhub.Event, partitionID string) error {
	timestamp := time.Now()
	azure := common.MapStr{
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
			Fields: common.MapStr{
				"message": msg,
				"azure":   azure,
			},
		})
		if !ok {
			return errors.New("event has not been sent")
		}
	}
	return nil
}

// parseMultipleMessages will try to split the message into multiple ones based on the group field provided by the configuration
func (a *azureInput) parseMultipleMessages(bMessage []byte) []string {
	var obj map[string][]interface{}
	err := json.Unmarshal(bMessage, &obj)
	if err != nil {
		a.log.Errorw(fmt.Sprintf("deserializing multiple messages using the group object `records`"), "error", err)
		return []string{string(bMessage)}
	}
	var messages []string
	if len(obj[expandEventListFromField]) > 0 {
		for _, ms := range obj[expandEventListFromField] {
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
