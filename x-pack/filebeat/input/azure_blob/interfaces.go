// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure_blob

import (
	"context"
	"fmt"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/elastic/elastic-agent-libs/logp"
)

// ------
// S3 interfaces
// ------

type blobAPI interface {
	blobGetter
	blobLister
}

type blobGetter interface {
	GetObject(ctx context.Context, blob string) (*azblob.DownloadResponse, error)
}

type blobLister interface {
	ListObjectsPaginator(ctx context.Context, prefix string, marker azblob.Marker) (*azblob.ListBlobsFlatSegmentResponse, error)
}

// type blobPager interface {
// 	Next(ctx context.Context) bool
// 	CurrentPage() *s3.ListObjectsOutput
// 	Err() error
// }

type blobObjectHandlerFactory interface {
	// Create returns a new s3ObjectHandler that can be used to process the
	// specified S3 object. If the handler is not configured to process the
	// given S3 object (based on key name) then it will return nil.
	Create(ctx context.Context, log *logp.Logger, acker *EventACKTracker, container, blob string) blobObjectHandler
}

type blobObjectHandler interface {
	// ProcessS3Object downloads the S3 object, parses it, creates events, and
	// publishes them. It returns when processing finishes or when it encounters
	// an unrecoverable error. It does not wait for the events to be ACKed by
	// the publisher before returning (use eventACKTracker's Wait() method to
	// determine this).
	ProcessBlobObject() error

	// Wait waits for every event published by ProcessS3Object() to be ACKed
	// by the publisher before returning. Internally it uses the
	// s3ObjectHandler eventACKTracker's Wait() method
	Wait()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ------
// Azure Blob implementation
// ------

type azureBlobAPI struct {
	client azblob.ContainerURL
}

func (a *azureBlobAPI) GetObject(ctx context.Context, blob_name string) (*azblob.DownloadResponse, error) {
	blobURL := a.client.NewBlockBlobURL(blob_name)
	resp, err := blobURL.Download(ctx, 0, 0, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, fmt.Errorf("Azure Blob GetObject failed: %w", err)
	}
	return resp, nil
}

func (a *azureBlobAPI) ListObjectsPaginator(ctx context.Context, prefix string, marker azblob.Marker) (*azblob.ListBlobsFlatSegmentResponse, error) {
	req, err := a.client.ListBlobsFlatSegment(ctx, marker, azblob.ListBlobsSegmentOptions{Prefix: prefix})
	if err != nil {
		return nil, fmt.Errorf("Azure Blob GetObjects failed: %w", err)
	}
	return req, nil
}
