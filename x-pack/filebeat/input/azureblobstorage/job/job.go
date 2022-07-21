// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package job

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/types"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Job interface {
	Do(ctx context.Context, id string) error
	Name() string
	Timestamp() *time.Time
	Source() *types.Source
}

type AzureInputJob struct {
	marker    *string
	client    *azblob.BlobClient
	blob      *azblob.BlobItemInternal
	state     *state.State
	src       *types.Source
	publisher cursor.Publisher
}

// NewAzureInputJob, returns an instance of a job , which is a unit of work that can be assigned to a go routine
func NewAzureInputJob(client *azblob.BlobClient, blob *azblob.BlobItemInternal,
	marker *string, state *state.State, src *types.Source, publisher cursor.Publisher) Job {

	return &AzureInputJob{
		marker:    marker,
		client:    client,
		blob:      blob,
		state:     state,
		src:       src,
		publisher: publisher,
	}
}

func (aij *AzureInputJob) Do(ctx context.Context, id string) error {

	var event beat.Event
	msg := mapstr.M{}
	if types.AllowedContentTypes[*aij.blob.Properties.ContentType] {
		data, err := aij.extractData(ctx)
		if err != nil {
			return fmt.Errorf("job with jobId %s encountered an error : %w", id, err)
		}

		if _, err := msg.Put("message.containerName", aij.src.ContainerName); err != nil {
			return err
		}
		if _, err := msg.Put("message.blobName", aij.blob.Name); err != nil {
			return err
		}
		if _, err := msg.Put("message.content_type", aij.blob.Properties.ContentType); err != nil {
			return err
		}
		if _, err := msg.Put("message.data", data.String()); err != nil {
			return err
		}
		if _, err := msg.Put("event.kind", "publish_data"); err != nil {
			return err
		}

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error : content-type %s not supported", id, *aij.blob.Properties.ContentType)
		if _, err := msg.Put("message.error", err); err != nil {
			return err
		}
		if _, err := msg.Put("event.kind", "publish_error"); err != nil {
			return err
		}
	}

	event = beat.Event{
		Timestamp: time.Now(),
		Fields:    msg,
	}
	aij.state.Save(*aij.blob.Name, aij.marker, aij.blob.Properties.LastModified)
	if err := aij.publisher.Publish(event, aij.state.Checkpoint()); err != nil {
		return err
	}

	return nil
}

func (aij *AzureInputJob) Name() string {
	return *aij.blob.Name
}

func (aij *AzureInputJob) Source() *types.Source {
	return aij.src
}
func (aij *AzureInputJob) Timestamp() *time.Time {
	return aij.blob.Properties.LastModified
}

func (aij *AzureInputJob) extractData(ctx context.Context) (*bytes.Buffer, error) {
	var err error

	get, err := aij.client.Download(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download data from blob with error : %w", err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	defer func() {
		err = reader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close blob reader with error : %w", err)
		}
	}()

	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from blob with error : %w", err)
	}

	return downloadedData, err
}
