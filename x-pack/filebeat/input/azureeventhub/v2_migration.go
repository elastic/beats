// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix

package azureeventhub

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"

	"github.com/elastic/elastic-agent-libs/logp"
)

type consumerClient interface {
	GetEventHubProperties(ctx context.Context, options *azeventhubs.GetEventHubPropertiesOptions) (azeventhubs.EventHubProperties, error)
}

type containerClient interface {
	NewBlobClient(blobName string) *blob.Client
	NewListBlobsFlatPager(o *container.ListBlobsFlatOptions) *runtime.Pager[container.ListBlobsFlatResponse]
}

type checkpointer interface {
	SetCheckpoint(ctx context.Context, checkpoint azeventhubs.Checkpoint, options *azeventhubs.SetCheckpointOptions) error
}

// migrationAssistant assists the input in migrating
// v1 checkpoint information to v2.
type migrationAssistant struct {
	config              azureInputConfig
	log                 *logp.Logger
	consumerClient      consumerClient
	blobContainerClient containerClient
	checkpointStore     checkpointer
}

// newMigrationAssistant creates a new migration assistant.
func newMigrationAssistant(config azureInputConfig, log *logp.Logger, consumerClient consumerClient, blobContainerClient containerClient, checkpointStore checkpointer) *migrationAssistant {
	return &migrationAssistant{
		config:              config,
		log:                 log,
		consumerClient:      consumerClient,
		blobContainerClient: blobContainerClient,
		checkpointStore:     checkpointStore,
	}
}

// checkAndMigrate checks if the v1 checkpoint information for the partitions
// exists and migrates it to v2 if it does.
func (m *migrationAssistant) checkAndMigrate(ctx context.Context, eventHubConnectionString, consumerGroup string) error {
	// Fetching event hub information
	eventHubProperties, err := m.consumerClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get event hub properties: %w", err)
	}

	m.log.Infow(
		"event hub information",
		"name", eventHubProperties.Name,
		"created_on", eventHubProperties.CreatedOn,
		"partition_ids", eventHubProperties.PartitionIDs,
	)

	// The input v1 stores the checkpoint information at the
	// root of the container.
	blobs, err := m.listBlobs(ctx)
	if err != nil {
		return err
	}

	for _, partitionID := range eventHubProperties.PartitionIDs {
		err = m.checkAndMigratePartition(
			ctx,
			blobs,
			partitionID,
			m.config.ConnectionStringProperties.FullyQualifiedNamespace,
			eventHubProperties.Name,
			consumerGroup,
		)
		if err != nil {
			return fmt.Errorf("failed to check and migrate partition: %w", err)
		}
	}

	return nil
}

// checkAndMigratePartition checks if the v1 checkpoint information for the `partitionID` exists
// `partitionID` partition.
func (m *migrationAssistant) checkAndMigratePartition(
	ctx context.Context,
	blobs map[string]bool,
	partitionID,
	fullyQualifiedNamespace,
	eventHubName,
	consumerGroup string) error {

	// Build the blob path (in the v2 checkpoint format) for the partition `partitionID`
	// using the fully qualified namespace, event hub name, consumer group, and partition ID.
	//
	// The blob path is in the format:
	//     {fullyQualifiedNamespace}/{eventHubName}/{consumerGroup}/checkpoint/{partitionID}
	//
	// Here is an example of the blob path:
	//     mbranca-general.servicebus.windows.net/mbrancalogs/$Default/checkpoint/0
	//
	blob := fmt.Sprintf("%s/%s/%s/checkpoint/%s", fullyQualifiedNamespace, eventHubName, consumerGroup, partitionID)

	// Check if v2 checkpoint information exists
	if _, ok := blobs[blob]; ok {
		m.log.Infow(
			"checkpoint v2 information for partition already exists, no migration needed",
			"partitionID", partitionID,
		)

		return nil
	}

	// Check if v1 checkpoint information exists
	if _, ok := blobs[partitionID]; !ok {
		m.log.Infow(
			"checkpoint v1 information for partition doesn't exist, no migration needed",
			"partitionID", partitionID,
		)

		return nil
	}

	// Try downloading the checkpoint v1 information for the partition
	cln := m.blobContainerClient.NewBlobClient(partitionID)

	// 4KB buffer should be enough to read
	// the checkpoint v1 information.
	buff := [4096]byte{}

	size, err := cln.DownloadBuffer(ctx, buff[:], nil)
	if err != nil {
		return fmt.Errorf("failed to download checkpoint v1 information for partition %s: %w", partitionID, err)
	}

	m.log.Infow(
		"downloaded checkpoint v1 information for partition",
		"partitionID", partitionID,
		"size", size,
	)

	// Unmarshal the checkpoint v1 information
	var checkpointV1 *LegacyCheckpoint

	if err := json.Unmarshal(buff[0:size], &checkpointV1); err != nil {
		return fmt.Errorf("failed to unmarshal checkpoint v1 information for partition %s: %w", partitionID, err)
	}

	// migrate the checkpoint v1 information to v2
	m.log.Infow("migrating checkpoint v1 information to v2", "partitionID", partitionID)

	// Common checkpoint information
	checkpointV2 := azeventhubs.Checkpoint{
		ConsumerGroup:           consumerGroup,
		EventHubName:            eventHubName,
		FullyQualifiedNamespace: fullyQualifiedNamespace,
		PartitionID:             partitionID,
	}

	offset, err := strconv.ParseInt(checkpointV1.Checkpoint.Offset, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse offset: %w", err)
	}

	checkpointV2.Offset = &offset
	checkpointV2.SequenceNumber = &checkpointV1.Checkpoint.SequenceNumber

	// Stores the checkpoint v2 information for the partition
	if err := m.checkpointStore.SetCheckpoint(ctx, checkpointV2, nil); err != nil {
		return fmt.Errorf("failed to update checkpoint v2 information for partition %s: %w", partitionID, err)
	}

	m.log.Infow("migrated checkpoint v1 information to v2", "partitionID", partitionID)

	return nil
}

// listBlobs lists all the blobs in the container.
func (m *migrationAssistant) listBlobs(ctx context.Context) (map[string]bool, error) {
	blobs := map[string]bool{}

	c := m.blobContainerClient.NewListBlobsFlatPager(nil)
	for c.More() {
		page, err := c.NextPage(ctx)
		if err != nil {
			return map[string]bool{}, fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			blobs[*blob.Name] = true
		}
	}
	return blobs, nil
}

type LegacyCheckpoint struct {
	PartitionID string `json:"partitionID"`
	Epoch       int    `json:"epoch"`
	Owner       string `json:"owner"`
	Checkpoint  struct {
		Offset         string `json:"offset"`
		SequenceNumber int64  `json:"sequenceNumber"`
		EnqueueTime    string `json:"enqueueTime"` // ": "0001-01-01T00:00:00Z"
	} `json:"checkpoint"`
}
