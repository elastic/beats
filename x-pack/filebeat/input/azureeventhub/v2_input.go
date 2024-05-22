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

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type eventHubInputV2 struct {
	config          azureInputConfig
	log             *logp.Logger
	metrics         *inputMetrics
	checkpointStore *checkpoints.BlobStore
	consumerClient  *azeventhubs.ConsumerClient
	pipelineClient  beat.Client
	messageDecoder  messageDecoder
}

func newEventHubInputV2(config azureInputConfig, log *logp.Logger) (v2.Input, error) {
	return &eventHubInputV2{
		config: config,
		log:    log.Named(inputName),
	}, nil
}

func (in *eventHubInputV2) Name() string {
	return inputName
}

func (in *eventHubInputV2) Test(v2.TestContext) error {
	return nil
}

func (in *eventHubInputV2) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	var err error

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

	// Create pipelineClient for publishing events and receive notification of their ACKs.
	in.pipelineClient, err = createPipelineClient(pipeline)
	if err != nil {
		return fmt.Errorf("failed to create pipeline pipelineClient: %w", err)
	}
	defer in.pipelineClient.Close()

	// Setup input metrics
	inputMetrics := newInputMetrics(inputContext.ID, nil)
	defer inputMetrics.Close()
	in.metrics = inputMetrics

	// Decode the messages from event hub into
	// a `[]string`.
	in.messageDecoder = messageDecoder{
		config:  &in.config,
		log:     in.log,
		metrics: in.metrics,
	}

	// Initialize the components needed to process events, in particular
	// the consumerClient.
	err = in.setup(ctx)
	if err != nil {
		return err
	}
	defer in.consumerClient.Close(context.Background())

	// Start the main run loop
	in.run(ctx)

	return nil
}

func (in *eventHubInputV2) setup(ctx context.Context) error {
	// FIXME: check more pipelineClient creation options.
	blobContainerClient, err := container.NewClientFromConnectionString(
		in.config.SAConnectionString,
		in.config.SAContainer,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create blob container pipelineClient: %w", err)
	}

	checkpointStore, err := checkpoints.NewBlobStore(blobContainerClient, nil)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint store: %w", err)
	}
	in.checkpointStore = checkpointStore

	consumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(
		in.config.ConnectionString,
		in.config.EventHubName,
		in.config.ConsumerGroup,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create consumer pipelineClient: %w", err)
	}
	in.consumerClient = consumerClient

	return nil
}

func (in *eventHubInputV2) run(ctx context.Context) {
	processor, err := azeventhubs.NewProcessor(
		in.consumerClient,
		in.checkpointStore,
		nil,
	)
	if err != nil {
		in.log.Errorw("error creating processor", "error", err)
		return
	}

	// Run in the background, launching goroutines to process each partition
	go in.workersLoop(processor)

	if err := processor.Run(ctx); err != nil {
		// FIXME: `Run()` returns an error when the processor thinks it's unrecoverable.
		// We should check the error and decide if we want to retry or not. Should
		// we add an exponential backoff and retry mechanism?
		in.log.Errorw("error running processor", "error", err)
	}
}

func (in *eventHubInputV2) workersLoop(processor *azeventhubs.Processor) {
	for {
		processorPartitionClient := processor.NextPartitionClient(context.TODO())
		if processorPartitionClient == nil {
			// Processor has stopped
			break
		}

		go func() {
			in.log.Infow("starting a partition worker", "partition", processorPartitionClient.PartitionID())

			if err := in.processEventsForPartition(processorPartitionClient); err != nil {
				// FIXME: it seems we always get an error, even when the processor is stopped.
				in.log.Infow(
					"stopping processing events for partition",
					"reason", err,
					"partition", processorPartitionClient.PartitionID(),
				)
			}

			in.log.Infow(
				"partition worker exited",
				"partition", processorPartitionClient.PartitionID(),
			)
		}()
	}
}

// processEventsForPartition shows the typical pattern for processing a partition.
func (in *eventHubInputV2) processEventsForPartition(partitionClient *azeventhubs.ProcessorPartitionClient) error {
	// 1. [BEGIN] Initialize any partition specific resources for your application.
	// 2. [CONTINUOUS] Loop, calling ReceiveEvents() and UpdateCheckpoint().
	// 3. [END] Cleanup any resources.
	partitionID := partitionClient.PartitionID()

	defer func() {
		// 3/3 [END] Do cleanup here, like shutting down database clients
		// or other resources used for processing this partition.
		shutdownPartitionResources(partitionClient)
	}()

	// 1/3 [BEGIN] Initialize any partition specific resources for your application.
	if err := initializePartitionResources(partitionID); err != nil {
		return err
	}

	// 2/3 [CONTINUOUS] Receive events, checkpointing as needed using UpdateCheckpoint.
	for {
		// Wait up to a minute for 100 events, otherwise returns whatever we collected during that time.
		receiveCtx, cancelReceive := context.WithTimeout(context.TODO(), 10*time.Second)
		events, err := partitionClient.ReceiveEvents(receiveCtx, 100, nil)
		cancelReceive()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			var eventHubError *azeventhubs.Error

			if errors.As(err, &eventHubError) && eventHubError.Code == azeventhubs.ErrorCodeOwnershipLost {
				return nil
			}

			return err
		}

		if len(events) == 0 {
			continue
		}

		in.log.Debugw("received events", "partition", partitionID)

		err = in.processReceivedEvents(events)
		if err != nil {
			return fmt.Errorf("error processing received events: %w", err)
		}

		in.log.Debugw("updating checkpoint information", "partition", partitionID)

		// Updates the checkpoint with the latest event received. If processing needs to restart
		// it will restart from this point, automatically.
		if err := partitionClient.UpdateCheckpoint(context.TODO(), events[len(events)-1], nil); err != nil {
			in.log.Errorw("error updating checkpoint", "error", err)
			return err
		}

		in.log.Debugw("checkpoint updated", "partition", partitionID)
	}
}

// processReceivedEvents
func (in *eventHubInputV2) processReceivedEvents(receivedEvents []*azeventhubs.ReceivedEventData) error {
	processingStartTime := time.Now()
	azure := mapstr.M{
		// The partition ID is not available.
		// "partition_id":   partitionID,
		"eventhub":       in.config.EventHubName,
		"consumer_group": in.config.ConsumerGroup,
	}

	for _, receivedEventData := range receivedEvents {
		// A single event can contain multiple records. We create a new event for each record.
		//records := in.unpackRecords(receivedEventData.Body)
		records := in.messageDecoder.Decode(receivedEventData.Body)

		for record := range records {
			_, _ = azure.Put("offset", receivedEventData.Offset)
			_, _ = azure.Put("sequence_number", receivedEventData.SequenceNumber)
			_, _ = azure.Put("enqueued_time", receivedEventData.EnqueuedTime)

			event := beat.Event{
				// this is the default value for the @timestamp field; usually the ingest
				// pipeline replaces it with a value in the payload.
				Timestamp: processingStartTime,
				Fields: mapstr.M{
					"message": record,
					"azure":   azure,
				},
				Private: receivedEventData.Body,
			}

			in.pipelineClient.Publish(event)
		}
	}

	return nil
}

func (in *eventHubInputV2) unpackRecords(bMessage []byte) []string {
	var mapObject map[string][]interface{}
	var records []string

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
					records = append(records, string(js))
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
				records = append(records, string(js))
				in.metrics.receivedEvents.Inc()
			} else {
				in.log.Errorw(fmt.Sprintf("serializing message %s", ms), "error", err)
			}
		}
	}

	return records
}

func initializePartitionResources(partitionID string) error {
	// initialize things that might be partition specific, like a
	// database connection.
	return nil
}

func shutdownPartitionResources(partitionClient *azeventhubs.ProcessorPartitionClient) {
	// Each PartitionClient holds onto an external resource and should be closed if you're
	// not processing them anymore.
	defer partitionClient.Close(context.TODO())
}
