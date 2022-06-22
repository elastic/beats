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
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Job interface {
	Do(ctx context.Context, jobID string, marker *string, wg *sync.WaitGroup, errChan chan<- error)
	Name() string
}

type azureInputJob struct {
	client    *azblob.BlockBlobClient
	blob      *azblob.BlobItemInternal
	state     *state.State
	src       *source
	publisher cursor.Publisher
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

func (aij *azureInputJob) Do(ctx context.Context, jobID string, marker *string, wg *sync.WaitGroup, errChan chan<- error) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		errChan <- ctx.Err()
		return
	default:
		data, err := aij.extractData(ctx)
		if err != nil {
			errChan <- fmt.Errorf("Job with jobID %s encountered an error : %v", jobID, err)
			return
		}
		msg := mapstr.M{}
		msg.Put("container.name", aij.src.containerName)
		msg.Put("container.blob.name", aij.blob.Name)
		msg.Put("container.blob.content_type", aij.blob.Properties.ContentType)
		msg.Put("container.blob.data", data.String())
		msg.Put("event.kind", "publish_data")
		event := beat.Event{
			Timestamp: time.Now(),
			Fields:    msg,
		}
		aij.state.Save(*aij.blob.Name, marker, aij.blob.Properties.LastModified)
		aij.publisher.Publish(event, aij.state.Checkpoint())
	}

}

func (aij *azureInputJob) Name() string {
	return *aij.blob.Name
}

func (aij *azureInputJob) extractData(ctx context.Context) (*bytes.Buffer, error) {
	get, err := aij.client.Download(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download data from blob with error : %v", err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from blob with error : %v", err)
	}

	err = reader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close blob reader with error : %v", err)
	}

	return downloadedData, nil
}
