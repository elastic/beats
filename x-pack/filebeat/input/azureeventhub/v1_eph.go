// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

// import (
//
//	"context"
//	"errors"
//	"fmt"
//
//	eventhub "github.com/Azure/azure-event-hubs-go/v3"
//	"github.com/Azure/azure-event-hubs-go/v3/eph"
//	"github.com/Azure/azure-event-hubs-go/v3/storage"
//	"github.com/Azure/azure-storage-blob-go/azblob"
//	"github.com/Azure/go-autorest/autorest/azure"
//
// )
//
// // users can select from one of the already defined azure cloud envs
//var environments = map[string]azure.Environment{
//	azure.ChinaCloud.ResourceManagerEndpoint:        azure.ChinaCloud,
//	azure.GermanCloud.ResourceManagerEndpoint:       azure.GermanCloud,
//	azure.PublicCloud.ResourceManagerEndpoint:       azure.PublicCloud,
//	azure.USGovernmentCloud.ResourceManagerEndpoint: azure.USGovernmentCloud,
//}

// // runWithEPH will consume ingested events using the Event Processor Host (EPH).
// //
// // To learn more, check the following resources:
// // - https://github.com/Azure/azure-event-hubs-go#event-processor-host
// // - https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-event-processor-host
//
//	func (in *eventHubInputV1) runWithEPH() error {
//		// create a new Azure Storage Leaser / Checkpointer
//		cred, err := azblob.NewSharedKeyCredential(in.config.SAName, in.config.SAKey)
//		if err != nil {
//			return err
//		}
//		env, err := getAzureEnvironment(in.config.OverrideEnvironment)
//		if err != nil {
//			return err
//		}
//		leaserCheckpointer, err := storage.NewStorageLeaserCheckpointer(cred, in.config.SAName, in.config.SAContainer, env)
//		if err != nil {
//			in.log.Errorw("error creating storage leaser checkpointer", "error", err)
//			return err
//		}
//
//		// adding a nil EventProcessorHostOption will break the code,
//		// this is why a condition is added and a.processor is assigned.
//		if in.config.ConsumerGroup != "" {
//			in.processor, err = eph.NewFromConnectionString(
//				in.workerCtx,
//				fmt.Sprintf("%s%s%s", in.config.ConnectionString, eventHubConnector, in.config.EventHubName),
//				leaserCheckpointer,
//				leaserCheckpointer,
//				eph.WithConsumerGroup(in.config.ConsumerGroup),
//				eph.WithNoBanner())
//		} else {
//			in.processor, err = eph.NewFromConnectionString(
//				in.workerCtx,
//				fmt.Sprintf("%s%s%s", in.config.ConnectionString, eventHubConnector, in.config.EventHubName),
//				leaserCheckpointer,
//				leaserCheckpointer,
//				eph.WithNoBanner())
//		}
//		if err != nil {
//			in.log.Errorw("error creating processor", "error", err)
//			return err
//		}
//
//		// register a message handler -- many can be registered
//		handlerID, err := in.processor.RegisterHandler(in.workerCtx,
//			func(c context.Context, e *eventhub.Event) error {
//				var onEventErr error
//				// partitionID is not yet mapped in the azure-eventhub sdk
//				ok := in.processEvents(e, "")
//				if !ok {
//					onEventErr = errors.New("OnEvent function returned false. Stopping input worker")
//					in.log.Error(onEventErr.Error())
//					in.Stop()
//				}
//				return onEventErr
//			})
//		if err != nil {
//			in.log.Errorw("error registering handler", "error", err)
//			return err
//		}
//		in.log.Infof("handler id: %q is registered\n", handlerID)
//
//		// Start handling messages from all of the partitions balancing across
//		// multiple consumers.
//		// The processor can be stopped by calling `Close()` on the processor.
//		err = in.processor.StartNonBlocking(in.workerCtx)
//		if err != nil {
//			in.log.Errorw("error starting the processor", "error", err)
//			return err
//		}
//
//		return nil
//	}
//func getAzureEnvironment(overrideResManager string) (azure.Environment, error) {
//	// if no override is set then the azure public cloud is used
//	if overrideResManager == "" || overrideResManager == "<no value>" {
//		return azure.PublicCloud, nil
//	}
//	if env, ok := environments[overrideResManager]; ok {
//		return env, nil
//	}
//	// can retrieve hybrid env from the resource manager endpoint
//	return azure.EnvironmentFromURL(overrideResManager)
//}
