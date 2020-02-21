// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureeventhub

import (
	"context"
	"errors"
	"fmt"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/Azure/azure-event-hubs-go/v3/eph"
	"github.com/Azure/azure-event-hubs-go/v3/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
)

// runWithEPH will consume ingested events using the Event Processor Host (EPH) https://github.com/Azure/azure-event-hubs-go#event-processor-host, https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host
func (a *azureInput) runWithEPH() error {
	// create a new Azure Storage Leaser / Checkpointer
	cred, err := azblob.NewSharedKeyCredential(a.config.SAName, a.config.SAKey)
	if err != nil {
		return err
	}
	leaserCheckpointer, err := storage.NewStorageLeaserCheckpointer(cred, a.config.SAName, a.config.SAContainer, azure.PublicCloud)
	if err != nil {
		return err
	}
	// adding a nil EventProcessorHostOption will break the code, this is why a condition is added and a.processor is assigned
	if a.config.ConsumerGroup != "" {
		a.processor, err = eph.NewFromConnectionString(
			a.workerCtx,
			fmt.Sprintf("%s%s%s", a.config.ConnectionString, eventHubConnector, a.config.EventHubName),
			leaserCheckpointer,
			leaserCheckpointer,
			eph.WithConsumerGroup(a.config.ConsumerGroup))
	} else {
		a.processor, err = eph.NewFromConnectionString(
			a.workerCtx,
			fmt.Sprintf("%s%s%s", a.config.ConnectionString, eventHubConnector, a.config.EventHubName),
			leaserCheckpointer,
			leaserCheckpointer)
	}
	if err != nil {
		return err
	}

	// register a message handler -- many can be registered
	handlerID, err := a.processor.RegisterHandler(a.workerCtx,
		func(c context.Context, e *eventhub.Event) error {
			var onEventErr error
			// partitionID is not yet mapped in the azure-eventhub sdk
			ok := a.processEvents(e, "")
			if !ok {
				onEventErr = errors.New("OnEvent function returned false. Stopping input worker")
				a.log.Debug(onEventErr.Error())
				a.Stop()
			}
			x := <-a.ackChannel
			_= x
			return onEventErr
		})
	if err != nil {
		return err
	}
	a.log.Infof("handler id: %q is running\n", handlerID)

	// unregister a handler to stop that handler from receiving events
	// processor.UnregisterHandler(ctx, handleID)

	// start handling messages from all of the partitions balancing across multiple consumers
	err = a.processor.Start(a.workerCtx)
	if err != nil {
		return err
	}
	return nil
}
