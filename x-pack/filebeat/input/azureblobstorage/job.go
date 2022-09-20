// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const jobErrString = "job with jobId %s encountered an error: %w"

type job struct {
	client    *azblob.BlobClient
	blob      *azblob.BlobItemInternal
	blobURL   string
	state     *state
	src       *Source
	publisher cursor.Publisher
	log       *logp.Logger
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(client *azblob.BlobClient, blob *azblob.BlobItemInternal, blobURL string,
	state *state, src *Source, publisher cursor.Publisher, log *logp.Logger,
) *job {
	return &job{
		client:    client,
		blob:      blob,
		blobURL:   blobURL,
		state:     state,
		src:       src,
		publisher: publisher,
		log:       log,
	}
}

func (j *job) do(ctx context.Context, id string) {
	var fields mapstr.M

	if allowedContentTypes[*j.blob.Properties.ContentType] {
		data, err := j.extractData(ctx)
		if err != nil {
			j.log.Errorf(jobErrString, id, err)
			return
		}

		reader := io.NopCloser(bytes.NewReader(data.Bytes()))
		defer func() {
			err = reader.Close()
			if err != nil {
				j.log.Errorf("failed to close io reader with error: %w", err)
			}
		}()

		fields = j.createEventFields(data.String())

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error: content-type %s not supported", id, *j.blob.Properties.ContentType)
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

	j.state.save(*j.blob.Name, j.blob.Properties.LastModified)
	if err := j.publisher.Publish(event, j.state.checkpoint()); err != nil {
		j.log.Errorf(jobErrString, id, err)
	}

}

func (j *job) name() string {
	return *j.blob.Name
}

func (j *job) timestamp() *time.Time {
	return j.blob.Properties.LastModified
}

func (j *job) extractData(ctx context.Context) (*bytes.Buffer, error) {
	var err error

	get, err := j.client.Download(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download data from blob with error: %w", err)
	}

	downloadedData := &bytes.Buffer{}
	reader := get.Body(&azblob.RetryReaderOptions{})
	defer func() {
		err = reader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close blob reader with error: %w", err)
		}
	}()

	_, err = downloadedData.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from blob with error: %w", err)
	}

	return downloadedData, err
}

func (j *job) createEventFields(message string) mapstr.M {
	return mapstr.M{
		"message": message, // original stringified data
		"log": mapstr.M{
			"file": mapstr.M{
				"path": j.blobURL,
			},
		},
		"azure": mapstr.M{
			"storage": mapstr.M{
				"container": mapstr.M{
					"name": j.src.ContainerName,
				},
				"blob": mapstr.M{
					"name":         j.blob.Name,
					"content_type": j.blob.Properties.ContentType,
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
