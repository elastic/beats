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

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs/checkpoints"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/elastic/elastic-agent-libs/logp"
)

// migrationAssistant assists the input in migrating
// checkpoint data from v1 to v2.
type migrationAssistant struct {
	log                 *logp.Logger
	consumerClient      *azeventhubs.ConsumerClient
	blobContainerClient *container.Client
	checkpointStore     *checkpoints.BlobStore
}

func newMigrationAssistant(log *logp.Logger, consumerClient *azeventhubs.ConsumerClient, blobContainerClient *container.Client, checkpointStore *checkpoints.BlobStore) *migrationAssistant {
	return &migrationAssistant{
		log:                 log,
		consumerClient:      consumerClient,
		blobContainerClient: blobContainerClient,
		checkpointStore:     checkpointStore,
	}
}

func (m *migrationAssistant) checkAndMigrate(ctx context.Context, eventHubConnectionString, eventHubName, consumerGroup string) error {

	// Fetching event hub information
	eventHubProperties, err := m.consumerClient.GetEventHubProperties(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to get event hub properties: %w", err)
	}

	m.log.Infow(
		"Event Hub properties",
		"name", eventHubProperties.Name,
		"created_on", eventHubProperties.CreatedOn,
		"partition_ids", eventHubProperties.PartitionIDs,
	)

	// Parse the connection string to get FQDN.
	props, err := parseConnectionString(eventHubConnectionString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	err = m.checkAndMigratePartition(ctx, eventHubProperties, props, eventHubName, consumerGroup)
	if err != nil {
		return fmt.Errorf("failed to check and migrate partition: %w", err)
	}

	// blobClient := m.blobContainerClient.NewBlobClient("")
	// blobClient.BlobExists(ctx)

	// blobPager := m.blobContainerClient.NewListBlobsFlatPager(nil)

	// for blobPager.More() {
	// 	page, err := blobPager.NextPage(ctx)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to list blobs: %w", err)
	// 	}

	// }

	// Fetching the list of blobs in the container.

	// Search for the checkpoint blobs in the container.
	// The blobs are named as <fullyQualifiedNamespace>/<eventHubName>/<consumerGroup>/checkpoint/<partitionID>

	// blobPager := m.blobContainerClient.NewListBlobsFlatPager(nil)

	// r, err := blobPager.NextPage(ctx)
	// if err != nil {
	// 	return fmt.Errorf("failed to list blobs: %w", err)
	// }

	// props.FullyQualifiedNamespace

	// // Fetching event hub information
	// eventHubProperties, err := m.consumerClient.GetEventHubProperties(ctx, nil)
	// if err != nil {
	// 	return fmt.Errorf("failed to get event hub properties: %w", err)
	// }

	// // v2 checkpoint information path
	// // mbranca-general.servicebus.windows.net/sdh4552/$Default/checkpoint/0

	// eventHubProperties.PartitionIDs

	return nil
}

func (m *migrationAssistant) checkAndMigratePartition(
	ctx context.Context,
	eventHubProperties azeventhubs.EventHubProperties,
	props ConnectionStringProperties,
	eventHubName,
	consumerGroup string) error {

	blobs := map[string]bool{}

	c := m.blobContainerClient.NewListBlobsFlatPager(nil)

	for c.More() {
		page, err := c.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list blobs: %w", err)
		}

		for _, blob := range page.Segment.BlobItems {
			blobs[*blob.Name] = true
		}
	}

	for _, partitionID := range eventHubProperties.PartitionIDs {
		// v2 checkpoint information path
		// mbranca-general.servicebus.windows.net/sdh4552/$Default/checkpoint/0
		blob := fmt.Sprintf("%s/%s/%s/checkpoint/%s", props.FullyQualifiedNamespace, eventHubName, consumerGroup, partitionID)

		if _, ok := blobs[blob]; ok {
			m.log.Infow(
				"checkpoint v2 information for partition already exists, no migration needed",
				"partitionID", partitionID,
			)
			continue
		}

		// try downloading the checkpoint v1 information for the partition
		if _, ok := blobs[partitionID]; !ok {
			m.log.Infow(
				"checkpoint v1 information for partition doesn't exist, no migration needed",
				"partitionID", partitionID,
			)
			continue
		}

		// v1 checkpoint information path is the partition ID itself
		cln := m.blobContainerClient.NewBlobClient(partitionID)

		buff := [4000]byte{}
		size, err := cln.DownloadBuffer(ctx, buff[:], nil)
		if err != nil {
			return fmt.Errorf("failed to download checkpoint v1 information for partition %s: %w", partitionID, err)
		}

		m.log.Infow("downloaded checkpoint v1 information for partition", "partitionID", partitionID, "size", size)

		var checkpointV1 *LegacyCheckpoint

		if err := json.Unmarshal(buff[0:size], &checkpointV1); err != nil {
			return fmt.Errorf("failed to unmarshal checkpoint v1 information for partition %s: %w", partitionID, err)
		}

		// migrate the checkpoint v1 information to v2
		m.log.Infow("migrating checkpoint v1 information to v2", "partitionID", partitionID)

		checkpointV2 := azeventhubs.Checkpoint{
			ConsumerGroup:           consumerGroup,
			EventHubName:            eventHubName,
			FullyQualifiedNamespace: props.FullyQualifiedNamespace,
			PartitionID:             partitionID,
		}

		offset, err := strconv.ParseInt(checkpointV1.Checkpoint.Offset, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse offset: %w", err)
		}

		checkpointV2.Offset = &offset
		checkpointV2.SequenceNumber = &checkpointV1.Checkpoint.SequenceNumber

		if err := m.checkpointStore.SetCheckpoint(ctx, checkpointV2, nil); err != nil {
			return fmt.Errorf("failed to update checkpoint v2 information for partition %s: %w", partitionID, err)
		}

		m.log.Infow("migrated checkpoint v1 information to v2", "partitionID", partitionID)
	}

	return nil
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
