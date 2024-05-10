// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
	"github.com/Azure/azure-event-hubs-go/v3/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type eventHubInputV1 struct {
	config    azureInputConfig
	log       *logp.Logger
	metrics   *inputMetrics
	processor *eph.EventProcessorHost
	client    beat.Client
}

func newEventHubInputV1(config azureInputConfig, log *logp.Logger) (v2.Input, error) {
	//log := logp.NewLogger(fmt.Sprintf("%s input", inputName)).With("connection string", stripConnectionString(config.ConnectionString))
	return &eventHubInputV1{
		config: config,
		log:    log.Named(inputName),
	}, nil
}

func (in *eventHubInputV1) Name() string {
	return inputName
}

func (in *eventHubInputV1) Test(v2.TestContext) error {
	return nil
}

func (in *eventHubInputV1) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	var err error

	// Create client for publishing events and receive notification of their ACKs.
	in.client, err = createPipelineClient(pipeline)
	if err != nil {
		return fmt.Errorf("failed to create pipeline client: %w", err)
	}
	defer in.client.Close()

	// Setup input metrics
	inputMetrics := newInputMetrics(inputContext.ID, nil)
	if err != nil {
		return fmt.Errorf("failed to create input metrics: %w", err)
	}
	defer inputMetrics.Close()
	in.metrics = inputMetrics

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

	// Initialize everything for this run
	err = in.setup(ctx)
	if err != nil {
		return err
	}

	// Start the main run loop
	err = in.run(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (in *eventHubInputV1) setup(ctx context.Context) error {

	// ----------------------------------------------------
	// 1 — Create a new Azure Storage Leaser / Checkpointer
	// ----------------------------------------------------

	cred, err := azblob.NewSharedKeyCredential(in.config.SAName, in.config.SAKey)
	if err != nil {
		return err
	}

	env, err := getAzureEnvironment(in.config.OverrideEnvironment)
	if err != nil {
		return err
	}

	leaserCheckpointer, err := storage.NewStorageLeaserCheckpointer(cred, in.config.SAName, in.config.SAContainer, env)
	if err != nil {
		in.log.Errorw("error creating storage leaser checkpointer", "error", err)
		return err
	}

	// ------------------------------------------------
	// 2 — Create a new event processor host
	// ------------------------------------------------

	// adding a nil EventProcessorHostOption will break the code,
	// this is why a condition is added and a.processor is assigned.
	if in.config.ConsumerGroup != "" {
		in.processor, err = eph.NewFromConnectionString(
			ctx,
			fmt.Sprintf("%s%s%s", in.config.ConnectionString, eventHubConnector, in.config.EventHubName),
			leaserCheckpointer,
			leaserCheckpointer,
			eph.WithConsumerGroup(in.config.ConsumerGroup),
			eph.WithNoBanner())
	} else {
		in.processor, err = eph.NewFromConnectionString(
			ctx,
			fmt.Sprintf("%s%s%s", in.config.ConnectionString, eventHubConnector, in.config.EventHubName),
			leaserCheckpointer,
			leaserCheckpointer,
			eph.WithNoBanner())
	}
	if err != nil {
		in.log.Errorw("error creating processor", "error", err)
		return err
	}

	// ------------------------------------------------
	// 3 — Register a message handler
	// ------------------------------------------------

	// register a message handler -- many can be registered
	handlerID, err := in.processor.RegisterHandler(ctx,
		func(c context.Context, e *eventhub.Event) error {
			var onEventErr error
			// partitionID is not yet mapped in the azure-eventhub sdk
			ok := in.processEvents(e, "")
			if !ok {
				onEventErr = errors.New("OnEvent function returned false. Stopping input worker")
				in.log.Error(onEventErr.Error())

				// FIXME: should we stop the processor here?
				// in.Stop()
			}

			return onEventErr
		})
	if err != nil {
		in.log.Errorw("error registering handler", "error", err)
		return err
	}

	in.log.Infof("handler id: %q is registered\n", handlerID)

	return nil
}

func (in *eventHubInputV1) run(ctx context.Context) error {
	// Start handling messages from all the partitions balancing across
	// multiple consumers.
	// The processor can be stopped by calling `Close()` on the processor.
	err := in.processor.StartNonBlocking(ctx)
	if err != nil {
		in.log.Errorw("error starting the processor", "error", err)
		return err
	}
	defer func() {
		in.log.Infof("%s input worker is stopping.", inputName)
		err := in.processor.Close(context.Background())
		if err != nil {
			in.log.Errorw("error while closing eventhostprocessor", "error", err)
		}
		in.log.Infof("%s input worker has stopped.", inputName)
	}()

	in.log.Infof("%s input worker has started.", inputName)

	for ctx.Err() == nil {
		select {
		case <-ctx.Done():
			return nil
		}
	}

	in.log.Errorw("error during processing", "error", ctx.Err())

	return ctx.Err()
}

func (in *eventHubInputV1) processEvents(event *eventhub.Event, partitionID string) bool {
	processingStartTime := time.Now()
	azure := mapstr.M{
		// partitionID is only mapped in the non-eph option which is not available yet, this field will be temporary unavailable
		//"partition_id":   partitionID,
		"eventhub":       in.config.EventHubName,
		"consumer_group": in.config.ConsumerGroup,
	}

	// update the input metrics
	in.metrics.receivedMessages.Inc()
	in.metrics.receivedBytes.Add(uint64(len(event.Data)))

	records := in.parseMultipleRecords(event.Data)

	for _, record := range records {
		_, _ = azure.Put("offset", event.SystemProperties.Offset)
		_, _ = azure.Put("sequence_number", event.SystemProperties.SequenceNumber)
		_, _ = azure.Put("enqueued_time", event.SystemProperties.EnqueuedTime)

		//ok := in.outlet.OnEvent(beat.Event{
		//	// this is the default value for the @timestamp field; usually the ingest
		//	// pipeline replaces it with a value in the payload.
		//	Timestamp: processingStartTime,
		//	Fields: mapstr.M{
		//		"message": record,
		//		"azure":   azure,
		//	},
		//	Private: event.Data,
		//})
		//if !ok {
		//	in.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())
		//	return ok
		//}

		event := beat.Event{
			// this is the default value for the @timestamp field; usually the ingest
			// pipeline replaces it with a value in the payload.
			Timestamp: processingStartTime,
			Fields: mapstr.M{
				"message": record,
				"azure":   azure,
			},
			Private: event.Data,
		}

		in.client.Publish(event)

		in.metrics.sentEvents.Inc()
	}

	in.metrics.processedMessages.Inc()
	in.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())

	return true
}

// parseMultipleRecords will try to split the message into multiple ones based on the group field provided by the configuration
func (in *eventHubInputV1) parseMultipleRecords(bMessage []byte) []string {
	var mapObject map[string][]interface{}
	var messages []string

	// Clean up the message for known issues [1] where Azure services produce malformed JSON documents.
	// Sanitization occurs if options are available and the message contains an invalid JSON.
	//
	// [1]: https://learn.microsoft.com/en-us/answers/questions/1001797/invalid-json-logs-produced-for-function-apps
	if len(in.config.SanitizeOptions) != 0 && !json.Valid(bMessage) {
		bMessage = sanitize(bMessage, in.config.SanitizeOptions...)
		in.metrics.sanitizedMessages.Inc()
	}

	// check if the message is a "records" object containing a list of events
	err := json.Unmarshal(bMessage, &mapObject)
	if err == nil {
		if len(mapObject[expandEventListFromField]) > 0 {
			for _, ms := range mapObject[expandEventListFromField] {
				js, err := json.Marshal(ms)
				if err == nil {
					messages = append(messages, string(js))
					in.metrics.receivedEvents.Inc()
				} else {
					in.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
				}
			}
		}
	} else {
		in.log.Debugf("deserializing multiple messages to a `records` object returning error: %s", err)
		// in some cases the message is an array
		var arrayObject []interface{}
		err = json.Unmarshal(bMessage, &arrayObject)
		if err != nil {
			// return entire message
			in.log.Debugf("deserializing multiple messages to an array returning error: %s", err)
			in.metrics.decodeErrors.Inc()
			return []string{string(bMessage)}
		}

		for _, ms := range arrayObject {
			js, err := json.Marshal(ms)
			if err == nil {
				messages = append(messages, string(js))
				in.metrics.receivedEvents.Inc()
			} else {
				in.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}

	return messages
}
