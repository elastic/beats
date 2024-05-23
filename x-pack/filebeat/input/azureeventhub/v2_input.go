// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// azureInputConfig the Azure Event Hub input v2,
// that uses the modern Azure Event Hub SDK for Go.
type eventHubInputV2 struct {
	config          azureInputConfig
	log             *logp.Logger
	metrics         *inputMetrics
	checkpointStore *checkpoints.BlobStore
	consumerClient  *azeventhubs.ConsumerClient
	pipelineClient  beat.Client
	messageDecoder  messageDecoder
}

// newEventHubInputV2 creates a new instance of the Azure Event Hub input v2,
// that uses the modern Azure Event Hub SDK for Go.
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

// Run starts the Azure Event Hub input v2.
func (in *eventHubInputV2) Run(
	inputContext v2.Context,
	pipeline beat.Pipeline,
) error {
	var err error

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

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

	// Initialize the components needed to process events,
	// in particular the consumerClient.
	err = in.setup(ctx)
	if err != nil {
		return err
	}
	defer in.consumerClient.Close(context.Background())

	// Create pipelineClient for publishing events and receive
	// notification of their ACKs.
	in.pipelineClient, err = createPipelineClient(pipeline)
	if err != nil {
		return fmt.Errorf("failed to create pipeline pipelineClient: %w", err)
	}
	defer in.pipelineClient.Close()

	// Start the main run loop
	in.run(ctx)

	return nil
}

// setup initializes the components needed to process events.
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

// run starts the main loop for processing events.
func (in *eventHubInputV2) run(ctx context.Context) {

	// Handle the case when the processor stops due to
	// transient errors (network failures) and we need to
	// restart.
	processorRunBackoff := backoff.NewEqualJitterBackoff(
		ctx.Done(),
		10*time.Second,  // initial backoff
		120*time.Second, // max backoff
	)

	for ctx.Err() == nil {
		// Create a new processor for each run.
		//
		// The docs explicitly say that the processor
		// is not reusable.
		processor, err := azeventhubs.NewProcessor(
			in.consumerClient,
			in.checkpointStore,
			nil, // default options
		)
		if err != nil {
			in.log.Errorw("error creating processor", "error", err)
			return
		}

		// Launch one goroutines for each partition
		// to process events.
		go in.workersLoop(ctx, processor)

		// Run the processor to start processing events.
		//
		// This is a blocking call. It will return when the processor
		// stops due to an error or when the context is cancelled.
		if err := processor.Run(ctx); err != nil {
			in.log.Errorw("processor exited with a non-nil error", "error", err)

			// FIXME: `time.Sleep()` is not the best way to handle this.
			// Using it for testing purposes.
			// time.Sleep(30 * time.Second)
			in.log.Infow("waiting before retrying starting the processor")

			// FIXME: `Run()` returns an error when the processor thinks it's unrecoverable.
			// We should check the error and decide if we want to retry or not. Should
			// we add an and retry mechanism with exponential backoff?
			processorRunBackoff.Wait()

			in.log.Infow("ready to try to start the processor again")
		}

		in.log.Infow(
			"run completed; continue if context error is nil",
			"context_error", ctx.Err(),
		)
	}
}

// workersLoop starts a goroutine for each partition to process events.
func (in *eventHubInputV2) workersLoop(ctx context.Context, processor *azeventhubs.Processor) {
	for {
		processorPartitionClient := processor.NextPartitionClient(ctx)
		if processorPartitionClient == nil {
			// We break out from the for loop when `NextPartitionClient`
			// return `nil` (signals the processor has stopped).
			break
		}

		partitionID := processorPartitionClient.PartitionID()

		go func() {
			in.log.Infow(
				"starting a partition worker",
				"partition", partitionID,
			)

			if err := in.processEventsForPartition(ctx, processorPartitionClient); err != nil {
				// FIXME: it seems we always get an error, even when the processor is stopped.
				in.log.Infow(
					"stopping processing events for partition",
					"reason", err,
					"partition", partitionID,
				)
			}

			in.log.Infow(
				"partition worker exited",
				"partition", partitionID,
			)
		}()
	}
}

// processEventsForPartition receives events from a partition and processes them.
func (in *eventHubInputV2) processEventsForPartition(ctx context.Context, partitionClient *azeventhubs.ProcessorPartitionClient) error {
	// 1. [BEGIN] Initialize any partition specific resources for your application.
	// 2. [CONTINUOUS] Loop, calling ReceiveEvents() and UpdateCheckpoint().
	// 3. [END] Cleanup any resources.
	defer func() {
		// 3/3 [END] Do cleanup here, like shutting down database clients
		// or other resources used for processing this partition.
		shutdownPartitionResources(partitionClient)
	}()

	partitionID := partitionClient.PartitionID()

	// 1/3 [BEGIN] Initialize any partition specific resources for your application.
	if err := initializePartitionResources(partitionID); err != nil {
		return err
	}

	// 2/3 [CONTINUOUS] Receive events, checkpointing as needed using UpdateCheckpoint.
	for {
		// Wait up to a minute for 100 events, otherwise returns whatever we collected during that time.
		receiveCtx, cancelReceive := context.WithTimeout(ctx, 5*time.Second)
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

		// Updates the checkpoint with the latest event received.
		//
		// If processing needs to restart it will restart from this
		// point, automatically.
		if err := partitionClient.UpdateCheckpoint(ctx, events[len(events)-1], nil); err != nil {
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
		// Update input metrics.
		in.metrics.receivedMessages.Inc()
		in.metrics.receivedBytes.Add(uint64(len(receivedEventData.Body)))

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
				Private: receivedEventData,
			}

			// Publish the event to the Beats pipeline.
			in.pipelineClient.Publish(event)

			// Update input metrics.
			in.metrics.sentEvents.Inc()
		}

		// Update input metrics.
		in.metrics.processedMessages.Inc()
		in.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())
	}

	return nil
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
