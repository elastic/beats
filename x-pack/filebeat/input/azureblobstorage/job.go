// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Job interface {
	Do(ctx context.Context, id string) error
	Name() string
	Timestamp() *time.Time
	Source() *source
}

type azureInputJob struct {
	marker    *string
	client    *azblob.BlockBlobClient
	blob      *azblob.BlobItemInternal
	state     *state.State
	src       *source
	publisher cursor.Publisher
}

func newAzureInputJobV2(client *azblob.BlockBlobClient, blob *azblob.BlobItemInternal,
	marker *string, state *state.State, src *source, publisher cursor.Publisher) Job {

	return &azureInputJob{
		marker:    marker,
		client:    client,
		blob:      blob,
		state:     state,
		src:       src,
		publisher: publisher,
	}
}

func newAzureInputJob(client *azblob.BlockBlobClient, blob *azblob.BlobItemInternal, state *state.State, src *source, publisher cursor.Publisher) Job {

	return &azureInputJob{
		client:    client,
		blob:      blob,
		state:     state,
		src:       src,
		publisher: publisher,
	}
}

// func (aij *azureInputJob) Do(ctx context.Context, jobID string, marker *string, wg *sync.WaitGroup, errChan chan<- error) {
// 	defer aij.wg.Done()

// 	select {
// 	case <-aij.ctx.Done():
// 		aij.errChan <- aij.ctx.Err()
// 		return
// 	default:
// 		var event beat.Event
// 		msg := mapstr.M{}
// 		if allowedContentTypes[*aij.blob.Properties.ContentType] {
// 			data, err := aij.extractData(ctx)
// 			if err != nil {
// 				errChan <- fmt.Errorf("Job with jobID %s encountered an error : %v", aij.id, err)
// 				return
// 			}
// 			msg.Put("message.containerName", aij.src.containerName)
// 			msg.Put("message.blobName", aij.blob.Name)
// 			msg.Put("message.content_type", aij.blob.Properties.ContentType)
// 			msg.Put("message.data", data.String())
// 			msg.Put("event.kind", "publish_data")

// 		} else {
// 			err := fmt.Errorf("Job with jobID %s encountered an error : content-type %s not supported", aij.id, *aij.blob.Properties.ContentType)
// 			msg.Put("message.error", err)
// 			msg.Put("event.kind", "publish_error")
// 		}

// 		event = beat.Event{
// 			Timestamp: time.Now(),
// 			Fields:    msg,
// 		}
// 		aij.state.Save(*aij.blob.Name, aij.marker, aij.blob.Properties.LastModified)
// 		aij.publisher.Publish(event, aij.state.Checkpoint())

// 	}

// }

func (aij *azureInputJob) Do(ctx context.Context, id string) error {

	var event beat.Event
	msg := mapstr.M{}
	if allowedContentTypes[*aij.blob.Properties.ContentType] {
		data, err := aij.extractData(ctx)
		if err != nil {
			return fmt.Errorf("Job with jobID %s encountered an error : %w", id, err)
		}
		msg.Put("message.containerName", aij.src.containerName)
		msg.Put("message.blobName", aij.blob.Name)
		msg.Put("message.content_type", aij.blob.Properties.ContentType)
		msg.Put("message.data", data.String())
		msg.Put("event.kind", "publish_data")

	} else {
		err := fmt.Errorf("Job with jobID %s encountered an error : content-type %s not supported", id, *aij.blob.Properties.ContentType)
		msg.Put("message.error", err)
		msg.Put("event.kind", "publish_error")
	}

	event = beat.Event{
		Timestamp: time.Now(),
		Fields:    msg,
	}
	aij.state.Save(*aij.blob.Name, aij.marker, aij.blob.Properties.LastModified)
	aij.publisher.Publish(event, aij.state.Checkpoint())

	return nil
}

func (aij *azureInputJob) Name() string {
	return *aij.blob.Name
}

func (aij *azureInputJob) Source() *source {
	return aij.src
}
func (aij *azureInputJob) Timestamp() *time.Time {
	return aij.blob.Properties.LastModified
}

func (aij *azureInputJob) extractData(ctx context.Context) (*bytes.Buffer, error) {
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
