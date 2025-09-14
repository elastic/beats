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
	"net/url"
	"strconv"
	"strings"

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
	log                 *logp.Logger
	consumerClient      consumerClient
	blobContainerClient containerClient
	checkpointStore     checkpointer
}

// newMigrationAssistant creates a new migration assistant.
func newMigrationAssistant(log *logp.Logger, consumerClient consumerClient, blobContainerClient containerClient, checkpointStore checkpointer) *migrationAssistant {
	return &migrationAssistant{
		log:                 log,
		consumerClient:      consumerClient,
		blobContainerClient: blobContainerClient,
		checkpointStore:     checkpointStore,
	}
}

// checkAndMigrate checks if the v1 checkpoint information for the partitions
// exists and migrates it to v2 if it does.
func (m *migrationAssistant) checkAndMigrate(ctx context.Context, eventHubConnectionString, eventHubName, consumerGroup string) error {
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

	// Parse the connection string to get FQDN.
	connectionStringInfo, err := parseConnectionString(eventHubConnectionString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	blobs, err := m.listBlobs(ctx)
	if err != nil {
		return err
	}

	for _, partitionID := range eventHubProperties.PartitionIDs {
		err = m.checkAndMigratePartition(ctx, blobs, partitionID, connectionStringInfo.FullyQualifiedNamespace, eventHubName, consumerGroup)
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

// ConnectionStringProperties are the properties of a connection string
// as returned by [ParseConnectionString].
type ConnectionStringProperties struct {
	// Endpoint is the Endpoint value in the connection string.
	// Ex: sb://example.servicebus.windows.net
	Endpoint string

	// EntityPath is EntityPath value in the connection string.
	EntityPath *string

	// FullyQualifiedNamespace is the Endpoint value without the protocol scheme.
	// Ex: example.servicebus.windows.net
	FullyQualifiedNamespace string

	// SharedAccessKey is the SharedAccessKey value in the connection string.
	SharedAccessKey *string

	// SharedAccessKeyName is the SharedAccessKeyName value in the connection string.
	SharedAccessKeyName *string

	// SharedAccessSignature is the SharedAccessSignature value in the connection string.
	SharedAccessSignature *string

	// Emulator indicates that the connection string is for an emulator:
	// ex: Endpoint=localhost:6765;SharedAccessKeyName=<< REDACTED >>;SharedAccessKey=<< REDACTED >>;UseDevelopmentEmulator=true
	Emulator bool
}

// ParseConnectionString takes a connection string from the Azure portal and returns the
// parsed representation.
//
// There are two supported formats:
//
//  1. Connection strings generated from the portal (or elsewhere) that contain an embedded key and keyname.
//
//  2. A connection string with an embedded SharedAccessSignature:
//     Endpoint=sb://<sb>.servicebus.windows.net;SharedAccessSignature=SharedAccessSignature sr=<sb>.servicebus.windows.net&sig=<base64-sig>&se=<expiry>&skn=<keyname>"
func parseConnectionString(connStr string) (ConnectionStringProperties, error) {
	const (
		endpointKey              = "Endpoint"
		sharedAccessKeyNameKey   = "SharedAccessKeyName"
		sharedAccessKeyKey       = "SharedAccessKey"
		entityPathKey            = "EntityPath"
		sharedAccessSignatureKey = "SharedAccessSignature"
		useEmulator              = "UseDevelopmentEmulator"
	)

	csp := ConnectionStringProperties{}

	splits := strings.Split(connStr, ";")

	for _, split := range splits {
		if split == "" {
			continue
		}

		keyAndValue := strings.SplitN(split, "=", 2)
		if len(keyAndValue) < 2 {
			return ConnectionStringProperties{}, errors.New("failed parsing connection string due to unmatched key value separated by '='")
		}

		// if a key value pair has `=` in the value, recombine them
		key := keyAndValue[0]
		value := strings.Join(keyAndValue[1:], "=")
		switch {
		case strings.EqualFold(endpointKey, key):
			u, err := url.Parse(value)
			if err != nil {
				return ConnectionStringProperties{}, errors.New("failed parsing connection string due to an incorrectly formatted Endpoint value")
			}
			csp.Endpoint = value
			csp.FullyQualifiedNamespace = u.Host
		case strings.EqualFold(sharedAccessKeyNameKey, key):
			csp.SharedAccessKeyName = &value
		case strings.EqualFold(sharedAccessKeyKey, key):
			csp.SharedAccessKey = &value
		case strings.EqualFold(entityPathKey, key):
			csp.EntityPath = &value
		case strings.EqualFold(sharedAccessSignatureKey, key):
			csp.SharedAccessSignature = &value
		case strings.EqualFold(useEmulator, key):
			v, err := strconv.ParseBool(value)

			if err != nil {
				return ConnectionStringProperties{}, err
			}

			csp.Emulator = v
		}
	}

	if csp.Emulator {
		// check that they're only connecting to localhost
		endpointParts := strings.SplitN(csp.Endpoint, ":", 3) // allow for a port, if it exists.

		if len(endpointParts) < 2 || endpointParts[0] != "sb" || endpointParts[1] != "//localhost" {
			// there should always be at least two parts "sb:" and "//localhost"
			// with an optional 3rd piece that's the port "1111".
			// (we don't need to validate it's a valid host since it's been through url.Parse() above)
			return ConnectionStringProperties{}, fmt.Errorf("UseDevelopmentEmulator=true can only be used with sb://localhost or sb://localhost:<port number>, not %s", csp.Endpoint)
		}
	}

	if csp.FullyQualifiedNamespace == "" {
		return ConnectionStringProperties{}, fmt.Errorf("key %q must not be empty", endpointKey)
	}

	if csp.SharedAccessSignature == nil && csp.SharedAccessKeyName == nil {
		return ConnectionStringProperties{}, fmt.Errorf("key %q must not be empty", sharedAccessKeyNameKey)
	}

	if csp.SharedAccessKey == nil && csp.SharedAccessSignature == nil {
		return ConnectionStringProperties{}, fmt.Errorf("key %q or %q cannot both be empty", sharedAccessKeyKey, sharedAccessSignatureKey)
	}

	return csp, nil
}
