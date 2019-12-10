package azureeventhub

import (
	"context"
	"github.com/Azure/azure-event-hubs-go/v2"
	"github.com/Azure/azure-event-hubs-go/v2/eph"
	"github.com/Azure/azure-event-hubs-go/v2/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest/azure"
)

// runWithEPH will consume ingested events using the Event Processor Host (EPH) https://github.com/Azure/azure-event-hubs-go#event-processor-host, https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host
func (a azureInput) runWithEPH() error {
	// create a new Azure Storage Leaser / Checkpointer
	cred, err := azblob.NewSharedKeyCredential(a.config.SAName, a.config.SAKey)
	if err != nil {
		return err
	}
	leaserCheckpointer, err := storage.NewStorageLeaserCheckpointer(cred, a.config.SAName, a.config.SAContainer, azure.PublicCloud)
	if err != nil {
		return err
	}
	var eventProcessorHost *eph.EventProcessorHost
	ctx, cancel := context.WithCancel(a.workerCtx)
	defer cancel()
	if a.config.ConsumerGroup != "" {
		eventProcessorHost, err = eph.NewFromConnectionString(
			ctx,
			a.config.ConnectionString+eventHubConnector+a.config.EventHubName,
			leaserCheckpointer,
			leaserCheckpointer,
			eph.WithConsumerGroup(a.config.ConsumerGroup))
	} else {
		eventProcessorHost, err = eph.NewFromConnectionString(
			ctx,
			a.config.ConnectionString+eventHubConnector+a.config.EventHubName,
			leaserCheckpointer,
			leaserCheckpointer)
	}
	if err != nil {
		return err
	}

	// register a message handler -- many can be registered
	handlerID, err := eventProcessorHost.RegisterHandler(ctx,
		func(c context.Context, e *eventhub.Event) error {
			return a.processEvents(e.Data)
		})
	if err != nil {
		return err
	}

	a.log.Info("handler id: %q is running\n", handlerID)

	// unregister a handler to stop that handler from receiving events
	// processor.UnregisterHandler(ctx, handleID)

	// start handling messages from all of the partitions balancing across multiple consumers
	err = eventProcessorHost.StartNonBlocking(ctx)
	if err != nil {
		return err
	}

	// Wait for a signal to quit:
	//signalChan := make(chan os.Signal, 1)
	//signal.Notify(signalChan, os.Interrupt, os.Kill)
	//<-signalChan

	//err = processor.Close(ctx)
	//if err != nil {
	//	return err
	//}
	return nil
}
