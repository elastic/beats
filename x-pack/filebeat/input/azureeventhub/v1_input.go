// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/go-autorest/autorest/azure"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
	"github.com/Azure/azure-event-hubs-go/v3/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// eventHubInputV1 is the Azure Event Hub input V1.
//
// This input uses the Azure Event Hub SDK v3 (legacy).
type eventHubInputV1 struct {
	config         azureInputConfig
	log            *logp.Logger
	metrics        *inputMetrics
	processor      *eph.EventProcessorHost
	pipelineClient beat.Client
	messageDecoder messageDecoder
}

// newEventHubInputV1 creates a new instance of the Azure Event Hub input V1.
// This input uses the Azure Event Hub SDK v3 (legacy).
func newEventHubInputV1(config azureInputConfig, logger *logp.Logger) (v2.Input, error) {
	log := logger.
		Named(inputName).
		With(
			"connection string", stripConnectionString(config.ConnectionString),
		)

	return &eventHubInputV1{
		config: config,
		log:    log,
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

	// Update the status to starting
	inputContext.UpdateStatus(status.Starting, "")

	// Create pipelineClient for publishing events.
	in.pipelineClient, err = createPipelineClient(pipeline)
	if err != nil {
		return fmt.Errorf("failed to create pipeline pipelineClient: %w", err)
	}
	defer in.pipelineClient.Close()

	// Setup input metrics
	in.metrics = newInputMetrics(inputContext.ID, nil)
	defer in.metrics.Close()

	// Set up new and legacy sanitizers, if any.
	sanitizers, err := newSanitizers(in.config.Sanitizers, in.config.LegacySanitizeOptions)
	if err != nil {
		return fmt.Errorf("failed to create sanitizers: %w", err)
	}

	in.messageDecoder = messageDecoder{
		config:     in.config,
		log:        in.log,
		metrics:    in.metrics,
		sanitizers: sanitizers,
	}

	ctx := v2.GoContextFromCanceler(inputContext.Cancelation)

	// Initialize the input components
	// in preparation for the main run loop.
	err = in.setup(ctx)
	if err != nil {
		return err
	}

	// Start the main run loop
	err = in.run(ctx)
	if err != nil {
		in.log.Errorw("error running input", "error", err)
		inputContext.UpdateStatus(status.Failed, err.Error())
		return err
	}

	inputContext.UpdateStatus(status.Stopped, "")
	return nil
}

// setup initializes the input components.
//
// The main components are:
// 1. Azure Storage Leaser / Checkpointer
// 2. Event Processor Host
// 3. Message handler
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

	in.log.Infof("storage leaser checkpointer created for container %q", in.config.SAContainer)

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

	in.log.Infof("event processor host created for event hub %q", in.config.EventHubName)

	// ------------------------------------------------
	// 3 — Register a message handler
	// ------------------------------------------------

	// register a message handler -- many can be registered
	handlerID, err := in.processor.RegisterHandler(ctx, func(c context.Context, e *eventhub.Event) error {

		// Take the event message from the event hub,
		// creates and publishes one (or more) events
		// to the beats pipeline.
		in.processEvents(e)

		// Why is this function always returning no error?
		//
		// The legacy SDK does not offer hooks to control
		// checkpointing (it internally updates the checkpoint
		// info after a successful handler execution).
		//
		// So we are keeping the existing behaviour (do not
		// handle publish acks).
		//
		// On shutdown, Filebeat stops the input, waits for
		// the output to process all the events in the queue.
		return nil
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

	// The `Start()` function is not an option because
	// it waits for an `os.Interrupt` signal to stop
	// the processor.
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

	// wait for the context to be done
	<-ctx.Done()

	return ctx.Err()
}

func (in *eventHubInputV1) processEvents(event *eventhub.Event) {
	processingStartTime := time.Now()
	eventHubMetadata := mapstr.M{
		// The `partition_id` is not available in the
		// legacy version of the SDK.
		"eventhub":       in.config.EventHubName,
		"consumer_group": in.config.ConsumerGroup,
	}

	// update the input metrics
	in.metrics.receivedMessages.Inc()
	in.metrics.receivedBytes.Add(uint64(len(event.Data)))

	records := in.messageDecoder.Decode(event.Data)

	for _, record := range records {
		_, _ = eventHubMetadata.Put("offset", event.SystemProperties.Offset)
		_, _ = eventHubMetadata.Put("sequence_number", event.SystemProperties.SequenceNumber)
		_, _ = eventHubMetadata.Put("enqueued_time", event.SystemProperties.EnqueuedTime)

		event := beat.Event{
			// We set the timestamp to the processing
			// start time as default value.
			//
			// Usually, the ingest pipeline replaces it
			// with a value in the payload.
			Timestamp: processingStartTime,
			Fields: mapstr.M{
				"message": record,
				"azure":   eventHubMetadata,
			},
			Private: event.Data,
		}

		in.pipelineClient.Publish(event)

		in.metrics.sentEvents.Inc()
	}

	in.metrics.processedMessages.Inc()
	in.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())
}

func createPipelineClient(pipeline beat.Pipeline) (beat.Client, error) {
	return pipeline.ConnectWith(beat.ClientConfig{
		EventListener: acker.LastEventPrivateReporter(func(acked int, data interface{}) {
			// fmt.Println(acked, data)
		}),
		Processing: beat.ProcessingConfig{
			// This input only produces events with basic types so normalization
			// is not required.
			EventNormalization: to.Ptr(false),
		},
	})
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

func getAzureEnvironment(overrideResManager string) (azure.Environment, error) {
	// if no override is set then the azure public cloud is used
	if overrideResManager == "" || overrideResManager == "<no value>" {
		return azure.PublicCloud, nil
	}
	if env, ok := environments[overrideResManager]; ok {
		return env, nil
	}
	// can retrieve hybrid env from the resource manager endpoint
	return azure.EnvironmentFromURL(overrideResManager)
}
