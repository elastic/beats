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

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// startPositionEarliest lets the processor start from the earliest
	// available event from the event hub retention period.
	startPositionEarliest = "earliest"
	// startPositionEarliest lets the processor start from the latest
	// available event from the event hub retention period.
	startPositionLatest = "latest"
	// processorRestartBackoff is the initial backoff time before
	// restarting the processor.
	processorRestartBackoff = 10 * time.Second
	// processorRestartMaxBackoff is the maximum backoff time before
	// restarting the processor.
	processorRestartMaxBackoff = 120 * time.Second
)

// azureInputConfig the Azure Event Hub input v2,
// that uses the modern Azure Event Hub SDK for Go.
type eventHubInputV2 struct {
	config             azureInputConfig
	log                *logp.Logger
	metrics            *inputMetrics
	checkpointStore    *checkpoints.BlobStore
	consumerClient     *azeventhubs.ConsumerClient
	pipeline           beat.Pipeline
	messageDecoder     messageDecoder
	migrationAssistant *migrationAssistant
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

	// Initialize the components needed to process events.
	err = in.setup(ctx)
	if err != nil {
		return err
	}
	defer in.teardown(ctx)

	// Store a reference to the pipeline, so we
	// can create a new pipeline client for each
	// partition.
	in.pipeline = pipeline

	// Start the main run loop
	in.run(ctx)

	return nil
}

// setup initializes the components needed to process events.
func (in *eventHubInputV2) setup(ctx context.Context) error {

	// Decode the messages from event hub into
	// a `[]string`.
	in.messageDecoder = messageDecoder{
		config:  in.config,
		log:     in.log,
		metrics: in.metrics,
	}

	containerClient, err := container.NewClientFromConnectionString(
		in.config.SAConnectionString,
		in.config.SAContainer,
		&container.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				// FIXME: check more pipelineClient creation options.
				Cloud: cloud.AzurePublic,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create blob container client: %w", err)
	}

	// The modern event hub SDK does not create the container
	// automatically like the old SDK.
	//
	// The new `BlobStore` explicitly says:
	//   "the container must exist before the checkpoint store can be used."
	//
	// We need to ensure it exists before we can use it.
	err = in.ensureContainerExists(ctx, containerClient)
	if err != nil {
		return fmt.Errorf("failed to ensure blob container exists: %w", err)
	}

	// Create the checkpoint store.
	//
	// The processor uses the checkpoint store to persist
	// checkpoint information in the container using blobs (one blob
	// for each event hub partition).
	checkpointStore, err := checkpoints.NewBlobStore(containerClient, nil)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint store: %w", err)
	}
	in.checkpointStore = checkpointStore

	// Create the event hub consumerClient to receive events.
	consumerClient, err := azeventhubs.NewConsumerClientFromConnectionString(
		in.config.ConnectionString,
		in.config.EventHubName,
		in.config.ConsumerGroup,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create consumer client: %w", err)
	}
	in.consumerClient = consumerClient

	// Manage the migration of the checkpoint information
	// from the old Event Hub SDK to the new Event Hub SDK.
	in.migrationAssistant = newMigrationAssistant(
		in.log,
		consumerClient,
		containerClient,
		checkpointStore,
	)

	return nil
}

// teardown releases the resources used by the input.
func (in *eventHubInputV2) teardown(ctx context.Context) {
	if in.consumerClient == nil {
		return
	}

	err := in.consumerClient.Close(ctx)
	if err != nil {
		in.log.Errorw(
			"error closing consumer client",
			"error", err,
		)
	}
}

// run starts the main loop for processing events.
func (in *eventHubInputV2) run(ctx context.Context) {
	if in.config.MigrateCheckpoint {
		in.log.Infow("checkpoint migration is enabled")
		// Check if we need to migrate the checkpoint store.
		err := in.migrationAssistant.checkAndMigrate(
			ctx,
			in.config.ConnectionString,
			in.config.EventHubName,
			in.config.ConsumerGroup,
		)
		if err != nil {
			in.log.Errorw("error migrating checkpoint store", "error", err)
			// FIXME: should we return here?
		}
	}

	// Handle the case when the processor stops due to
	// transient errors (network failures) and we need to
	// restart it.
	processorRunBackoff := backoff.NewEqualJitterBackoff(
		ctx.Done(),
		processorRestartBackoff,    // initial backoff
		processorRestartMaxBackoff, // max backoff
	)

	// Create the processor options using the input
	// configuration.
	processorOptions := createProcessorOptions(in.config)

	for ctx.Err() == nil {
		// Create a new processor for each run.
		//
		// The docs explicitly say that the processor
		// is not reusable.
		processor, err := azeventhubs.NewProcessor(
			in.consumerClient,
			in.checkpointStore,
			processorOptions,
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
		// This is a blocking call.
		//
		// It will return when the processor stops due to:
		//  - an error
		//	- when the context is cancelled.
		//
		// On cancellation, it will return a nil error.
		if err := processor.Run(ctx); err != nil {
			in.log.Errorw("processor exited with a non-nil error", "error", err)

			in.log.Infow("waiting before retrying starting the processor")

			// `Run()` returns an error when the processor thinks it's
			// unrecoverable.
			//
			// We wait before retrying to start the processor.
			processorRunBackoff.Wait()

			// Update input metrics.
			in.metrics.processorRestarts.Inc()
		}

		in.log.Infow(
			"run completed; restarting the processor if context error is nil",
			"context_error", ctx.Err(),
		)
	}
}

// createProcessorOptions creates the processor options using the input configuration.
func createProcessorOptions(config azureInputConfig) *azeventhubs.ProcessorOptions {

	// Start position offers multiple options:
	//
	// - Offset
	// - SequenceNumber
	// - EnqueuedTime
	//
	// As of now, we only support Earliest and Latest.
	//
	// The processor uses the default start position for
	// all partitions if there is no checkpoint information
	// available from the storage account container.
	defaultStartPosition := azeventhubs.StartPosition{}

	switch config.ProcessorStartPosition {
	case startPositionEarliest:
		defaultStartPosition.Earliest = to.Ptr(true)
	case startPositionLatest:
		defaultStartPosition.Latest = to.Ptr(true)
	}

	return &azeventhubs.ProcessorOptions{
		//
		// The `LoadBalancingStrategy` controls how the
		// processor distributes the partitions across the
		// consumers.
		//
		// LoadBalancingStrategy offers multiple options:
		//
		// - Balanced
		// - Greedy
		//
		// As of now, we only support the "balanced" load
		// balancing strategy for retro compatibility with
		// the old SDK.
		//
		LoadBalancingStrategy: azeventhubs.ProcessorStrategyBalanced,
		UpdateInterval:        config.ProcessorUpdateInterval,
		StartPositions: azeventhubs.StartPositions{
			Default: defaultStartPosition,
		},
	}
}

// ensureContainerExists ensures the blob container exists.
func (in *eventHubInputV2) ensureContainerExists(ctx context.Context, blobContainerClient *container.Client) error {
	exists, err := in.containerExists(ctx, blobContainerClient)
	if err != nil {
		return fmt.Errorf("failed to check if blob container exists: %w", err)
	}
	if exists {
		return nil
	}

	// Since the container does not exist, we create it.
	r, err := blobContainerClient.Create(ctx, nil)
	if err != nil {
		// If the container already exists, we ignore the error.
		var responseError *azcore.ResponseError
		if !errors.As(err, &responseError) || responseError.ErrorCode != string(bloberror.ContainerAlreadyExists) {
			return fmt.Errorf("failed to create blob container: %w", err)
		}

		// The container already exists, we can ignore the error.
		//
		// This is a possible scenario when another process created
		// the container after we checked if it exists.
		in.log.Debugw(
			"blob container already exists, no need to create a new one",
			"container", in.config.SAContainer,
		)
	}

	in.log.Infow("blob container created successfully", "response", r)

	return nil
}

// containerExists checks if the blob container exists.
func (in *eventHubInputV2) containerExists(ctx context.Context, blobContainerClient *container.Client) (bool, error) {
	// Try to access the container to see if it exists.
	_, err := blobContainerClient.GetProperties(ctx, &container.GetPropertiesOptions{})
	if err == nil {
		in.log.Debugw("blob container already exists, no need to create a new one", "container", in.config.SAContainer)
		return true, nil
	}

	var responseError *azcore.ResponseError
	if errors.As(err, &responseError) && responseError.ErrorCode == string(bloberror.ContainerNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check if blob container exists: %w", err)
}

// workersLoop starts a goroutine for each partition to process events.
func (in *eventHubInputV2) workersLoop(ctx context.Context, processor *azeventhubs.Processor) {
	for {
		// The call blocks until an owned partition is available or the
		// context is cancelled.
		processorPartitionClient := processor.NextPartitionClient(ctx)
		if processorPartitionClient == nil {
			// We break out from the for loop when `NextPartitionClient`
			// return `nil` (signals the processor has stopped).
			break
		}

		partitionID := processorPartitionClient.PartitionID()

		// Start a goroutine to process events for the partition.
		go func() {
			in.log.Infow(
				"starting a partition worker",
				"partition", partitionID,
			)

			if err := in.processEventsForPartition(ctx, processorPartitionClient); err != nil {
				// It seems we always get an error,
				// even when the processor is stopped.
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

	partitionID := partitionClient.PartitionID()

	// 1/3 [BEGIN] Initialize any partition specific resources for your application.
	pipelineClient, err := initializePartitionResources(ctx, partitionClient, in.pipeline, in.log)
	if err != nil {
		return err
	}

	defer func() {
		// 3/3 [END] Do cleanup here, like shutting down database clients
		// or other resources used for processing this partition.
		shutdownPartitionResources(ctx, partitionClient, pipelineClient)
		in.log.Debugw("partition resources cleaned up", "partition", partitionID)
	}()

	// 2/3 [CONTINUOUS] Receive events, checkpointing as needed using UpdateCheckpoint.
	for {
		// Wait up to `in.config.PartitionReceiveTimeout` for `in.config.PartitionReceiveCount` events,
		// otherwise returns whatever we collected during that time.
		receiveCtx, cancelReceive := context.WithTimeout(ctx, in.config.PartitionReceiveTimeout)
		events, err := partitionClient.ReceiveEvents(receiveCtx, in.config.PartitionReceiveCount, nil)
		cancelReceive()

		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			var eventHubError *azeventhubs.Error
			if errors.As(err, &eventHubError) && eventHubError.Code == azeventhubs.ErrorCodeOwnershipLost {
				in.log.Infow(
					"ownership lost for partition, stopping processing",
					"partition", partitionID,
				)

				return nil
			}

			return err
		}

		if len(events) == 0 {
			continue
		}

		err = in.processReceivedEvents(events, partitionID, pipelineClient)
		if err != nil {
			return fmt.Errorf("error processing received events: %w", err)
		}
	}
}

// processReceivedEvents
func (in *eventHubInputV2) processReceivedEvents(receivedEvents []*azeventhubs.ReceivedEventData, partitionID string, pipelineClient beat.Client) error {
	processingStartTime := time.Now()
	eventHubMetadata := mapstr.M{
		"partition_id":   partitionID,
		"eventhub":       in.config.EventHubName,
		"consumer_group": in.config.ConsumerGroup,
	}

	for _, receivedEventData := range receivedEvents {
		// Update input metrics.
		in.metrics.receivedMessages.Inc()
		in.metrics.receivedBytes.Add(uint64(len(receivedEventData.Body)))

		// A single event can contain multiple records.
		// We create a new event for each record.
		records := in.messageDecoder.Decode(receivedEventData.Body)

		for _, record := range records {
			_, _ = eventHubMetadata.Put("offset", receivedEventData.Offset)
			_, _ = eventHubMetadata.Put("sequence_number", receivedEventData.SequenceNumber)
			_, _ = eventHubMetadata.Put("enqueued_time", receivedEventData.EnqueuedTime)

			// The partition key is optional.
			if receivedEventData.PartitionKey != nil {
				_, _ = eventHubMetadata.Put("partition_key", *receivedEventData.PartitionKey)
			}

			event := beat.Event{
				// this is the default value for the @timestamp field; usually the ingest
				// pipeline replaces it with a value in the payload.
				Timestamp: processingStartTime,
				Fields: mapstr.M{
					"message": record,
					"azure":   eventHubMetadata,
				},
				Private: receivedEventData,
			}

			// Publish the event to the Beats pipeline.
			pipelineClient.Publish(event)

			// Update input metrics.
			in.metrics.sentEvents.Inc()
		}

		// Update input metrics.
		in.metrics.processedMessages.Inc()
		in.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())
	}

	return nil
}

// initializePartitionResources initializes any partition specific resources for your application.
//
// Sets up a pipelineClient for publishing events and receive notification of their ACKs.
func initializePartitionResources(ctx context.Context, partitionClient *azeventhubs.ProcessorPartitionClient, pipeline beat.Pipeline, log *logp.Logger) (beat.Client, error) {
	// initialize things that might be partition specific, like a
	// database connection.
	return pipeline.ConnectWith(beat.ClientConfig{
		EventListener: acker.LastEventPrivateReporter(func(acked int, data any) {
			// Update the checkpoint for the given partition.
			//
			// The pipeline calls this function when events have been
			// successfully published and acknowledged by the output.
			//
			// The data parameter contains the private data of the last
			// acknowledged event.
			receivedEventData, ok := data.(*azeventhubs.ReceivedEventData)
			if !ok {
				log.Errorw(
					"error updating checkpoint",
					"partition", partitionClient.PartitionID(),
					"acked", acked,
					"error", "invalid data type",
					"type", fmt.Sprintf("%T", data),
				)
				return
			}

			err := partitionClient.UpdateCheckpoint(ctx, receivedEventData, nil)
			if err != nil {
				log.Errorw(
					"error updating checkpoint",
					"error", err,
				)
			}

			log.Debugw(
				"checkpoint updated",
				"partition", partitionClient.PartitionID(),
				"acked", acked,
				"sequence_number", receivedEventData.SequenceNumber,
				"offset", receivedEventData.Offset,
				"enqueued_time", receivedEventData.EnqueuedTime,
			)
		}),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: to.Ptr(false),
		},
	})
}

func shutdownPartitionResources(ctx context.Context, partitionClient *azeventhubs.ProcessorPartitionClient, pipelineClient beat.Client) {
	// Each PartitionClient holds onto an external resource and should be closed if you're
	// not processing them anymore.
	defer partitionClient.Close(ctx)

	// Closing the pipeline since we're done
	// processing events for this partition.
	defer pipelineClient.Close()
}
