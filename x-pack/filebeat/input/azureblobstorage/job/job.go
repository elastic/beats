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
	"io"
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
	blobURL   string
	state     *state.State
	src       *types.Source
	publisher cursor.Publisher
}

// NewAzureInputJob, returns an instance of a job , which is a unit of work that can be assigned to a go routine
func NewAzureInputJob(client *azblob.BlobClient, blob *azblob.BlobItemInternal, blobURL string,
	marker *string, state *state.State, src *types.Source, publisher cursor.Publisher,
) Job {
	return &AzureInputJob{
		marker:    marker,
		client:    client,
		blob:      blob,
		blobURL:   blobURL,
		state:     state,
		src:       src,
		publisher: publisher,
	}
}

func (aij *AzureInputJob) Do(ctx context.Context, id string) error {
	var fields mapstr.M

	if types.AllowedContentTypes[*aij.blob.Properties.ContentType] {
		data, err := aij.extractData(ctx)
		if err != nil {
			return fmt.Errorf("job with jobId %s encountered an error : %w", id, err)
		}

		reader := io.NopCloser(bytes.NewReader(data.Bytes()))
		defer func() {
			err = reader.Close()
			if err != nil {
				err = fmt.Errorf("failed to close json reader with error : %w", err)
			}
		}()

		fields = aij.createEventFields(data.String())

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error : content-type %s not supported", id, *aij.blob.Properties.ContentType)
		fields = mapstr.M{
			"message": err.Error(),
			"event": mapstr.M{
				"kind": "publish_error",
			},
		}
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}
	event.SetID(id)

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

func (aij *AzureInputJob) createEventFields(message string) mapstr.M {
	return mapstr.M{
		"message": message, // original stringified data
		"log": mapstr.M{
			"file": mapstr.M{
				"path": aij.blobURL,
			},
		},
		"azure": mapstr.M{
			"storage": mapstr.M{
				"container": mapstr.M{
					"name": aij.src.ContainerName,
				},
				"blob": mapstr.M{
					"name":         aij.blob.Name,
					"content_type": aij.blob.Properties.ContentType,
				},
			},
		},
		"cloud": mapstr.M{
			"provider": "azure",
		},
		"event": mapstr.M{
			"kind": "publish_data",
		},
	}
}
