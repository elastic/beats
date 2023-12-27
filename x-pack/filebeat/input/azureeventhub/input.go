// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	eventHubConnector        = ";EntityPath="
	expandEventListFromField = "records"
	inputName                = "azure-eventhub"
)

func init() {
	err := input.Register(inputName, NewInput)
	if err != nil {
		panic(fmt.Errorf("failed to register %v input: %w", inputName, err))
	}
}

// configID computes a unique ID for the input configuration.
//
// It is used to identify the input in the registry and to detect
// changes in the configuration.
//
// We will remove this function as we upgrade the input to the
// v2 API (there is an ID in the v2 context).
func configID(config *conf.C) (string, error) {
	var tmp struct {
		ID string `config:"id"`
	}
	if err := config.Unpack(&tmp); err != nil {
		return "", fmt.Errorf("error extracting ID: %w", err)
	}
	if tmp.ID != "" {
		return tmp.ID, nil
	}

	var h map[string]interface{}
	_ = config.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute ID from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}

// azureInput struct for the azure-eventhub input
type azureInput struct {
	config       azureInputConfig // azure-eventhub configuration
	context      input.Context
	outlet       channel.Outleter
	log          *logp.Logger            // logging info and error messages
	workerCtx    context.Context         // worker goroutine context. It's cancelled when the input stops or the worker exits.
	workerCancel context.CancelFunc      // used to signal that the worker should stop.
	workerOnce   sync.Once               // guarantees that the worker goroutine is only started once.
	processor    *eph.EventProcessorHost // eph will be assigned if users have enabled the option
	id           string                  // ID of the input; used to identify the input in the input metrics registry only, and will be removed once the input is migrated to v2.
	metrics      *inputMetrics           // Metrics for the input.
}

// NewInput creates a new azure-eventhub input
func NewInput(
	cfg *conf.C,
	connector channel.Connector,
	inputContext input.Context,
) (input.Input, error) {
	var config azureInputConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("reading %s input config: %w", inputName, err)
	}

	// Since this is a v1 input, we need to set the ID manually.
	//
	// We need an ID to identify the input in the input metrics
	// registry.
	//
	// This is a temporary workaround until we migrate the input to v2.
	inputId, err := configID(cfg)
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

	in := azureInput{
		id:           inputId,
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

	return &in, nil
}

// Run starts the `azure-eventhub` input and then returns.
//
// The first invocation will start an input worker. All subsequent
// invocations will be no-ops.
//
// The input worker will continue fetching data from the event hub until
// the input Runner calls the `Stop()` method.
func (a *azureInput) Run() {
	// `Run` is invoked periodically by the input Runner. The `sync.Once`
	// guarantees that we only start the worker once during the first
	// invocation.
	a.workerOnce.Do(func() {
		a.log.Infof("%s input worker is starting.", inputName)

		// We set up the metrics in the `Run()` method and tear them down
		// in the `Stop()` method.
		//
		// The factory method `NewInput` is not a viable solution because
		// the Runner invokes it during the configuration check without
		// calling the `Stop()` function; this causes panics
		// due to multiple metrics registrations.
		a.metrics = newInputMetrics(a.id, nil)

		err := a.runWithEPH()
		if err != nil {
			a.log.Errorw("error starting the input worker", "error", err)
			return
		}
		a.log.Infof("%s input worker has started.", inputName)
	})
}

// Stop stops `azure-eventhub` input.
func (a *azureInput) Stop() {
	a.log.Infof("%s input worker is stopping.", inputName)
	if a.processor != nil {
		// Tells the processor to stop processing events and release all
		// resources (like scheduler, leaser, checkpointer, and client).
		err := a.processor.Close(context.Background())
		if err != nil {
			a.log.Errorw("error while closing eventhostprocessor", "error", err)
		}
	}

	if a.metrics != nil {
		a.metrics.Close()
	}

	a.workerCancel()
	a.log.Infof("%s input worker has stopped.", inputName)
}

// Wait stop the current server
func (a *azureInput) Wait() {
	a.Stop()
}

func (a *azureInput) processEvents(event *eventhub.Event, partitionID string) bool {
	processingStartTime := time.Now()
	azure := mapstr.M{
		// partitionID is only mapped in the non-eph option which is not available yet, this field will be temporary unavailable
		//"partition_id":   partitionID,
		"eventhub":       a.config.EventHubName,
		"consumer_group": a.config.ConsumerGroup,
	}

	// update the input metrics
	a.metrics.receivedMessages.Inc()
	a.metrics.receivedBytes.Add(uint64(len(event.Data)))

	records := a.parseMultipleRecords(event.Data)

	for _, record := range records {
		_, _ = azure.Put("offset", event.SystemProperties.Offset)
		_, _ = azure.Put("sequence_number", event.SystemProperties.SequenceNumber)
		_, _ = azure.Put("enqueued_time", event.SystemProperties.EnqueuedTime)
		ok := a.outlet.OnEvent(beat.Event{
			// this is the default value for the @timestamp field; usually the ingest
			// pipeline replaces it with a value in the payload.
			Timestamp: processingStartTime,
			Fields: mapstr.M{
				"message": record,
				"azure":   azure,
			},
			Private: event.Data,
		})
		if !ok {
			a.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())
			return ok
		}

		a.metrics.sentEvents.Inc()
	}

	a.metrics.processedMessages.Inc()
	a.metrics.processingTime.Update(time.Since(processingStartTime).Nanoseconds())

	return true
}

// parseMultipleRecords will try to split the message into multiple ones based on the group field provided by the configuration
func (a *azureInput) parseMultipleRecords(bMessage []byte) []string {
	var mapObject map[string][]interface{}
	var messages []string

	// Clean up the message for known issues [1] where Azure services produce malformed JSON documents.
	// Sanitization occurs if options are available and the message contains an invalid JSON.
	//
	// [1]: https://learn.microsoft.com/en-us/answers/questions/1001797/invalid-json-logs-produced-for-function-apps
	if len(a.config.SanitizeOptions) != 0 && !json.Valid(bMessage) {
		bMessage = sanitize(bMessage, a.config.SanitizeOptions...)
		a.metrics.sanitizedMessages.Inc()
	}

	// check if the message is a "records" object containing a list of events
	err := json.Unmarshal(bMessage, &mapObject)
	if err == nil {
		if len(mapObject[expandEventListFromField]) > 0 {
			for _, ms := range mapObject[expandEventListFromField] {
				js, err := json.Marshal(ms)
				if err == nil {
					messages = append(messages, string(js))
					a.metrics.receivedEvents.Inc()
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
			a.metrics.decodeErrors.Inc()
			return []string{string(bMessage)}
		}

		for _, ms := range arrayObject {
			js, err := json.Marshal(ms)
			if err == nil {
				messages = append(messages, string(js))
				a.metrics.receivedEvents.Inc()
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
