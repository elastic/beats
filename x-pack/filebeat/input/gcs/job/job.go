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

	"cloud.google.com/go/storage"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/gcs/types"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Job interface {
	Do(ctx context.Context, id string) error
	Name() string
	Timestamp() *time.Time
	Source() *types.Source
}

type GcsInputJob struct {
	bucket    *storage.BucketHandle
	object    *storage.ObjectAttrs
	objectURI string
	state     *state.State
	src       *types.Source
	publisher cursor.Publisher
}

// NewGcsInputJob, returns an instance of a job , which is a unit of work that can be assigned to a go routine
func NewGcsInputJob(bucket *storage.BucketHandle, object *storage.ObjectAttrs, objectURI string,
	state *state.State, src *types.Source, publisher cursor.Publisher,
) Job {
	return &GcsInputJob{
		bucket:    bucket,
		object:    object,
		objectURI: objectURI,
		state:     state,
		src:       src,
		publisher: publisher,
	}
}

func (gcsij *GcsInputJob) Do(ctx context.Context, id string) error {
	var fields mapstr.M

	if types.AllowedContentTypes[gcsij.object.ContentType] {
		data, err := gcsij.extractData(ctx)
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

		var blobData []mapstr.M
		switch gcsij.object.ContentType {
		case types.Json:
			blobData, _, _, err = httpReadJSON(reader)
			if err != nil {
				return err
			}
			// Support for more types will be added here, in the future.
		default:
			return fmt.Errorf("job with jobId %s encountered an unexpected error", id)
		}

		fields = gcsij.createEventFields(data.String(), blobData)

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error : content-type %s not supported", id, gcsij.object.ContentType)
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

	gcsij.state.Save(gcsij.object.Name, &gcsij.object.Updated)
	if err := gcsij.publisher.Publish(event, gcsij.state.Checkpoint()); err != nil {
		return err
	}

	return nil
}

func (gcsij *GcsInputJob) Name() string {
	return gcsij.object.Name
}

func (gcsij *GcsInputJob) Source() *types.Source {
	return gcsij.src
}

func (gcsij *GcsInputJob) Timestamp() *time.Time {
	return &gcsij.object.Updated
}

func (gcsij *GcsInputJob) extractData(ctx context.Context) (*bytes.Buffer, error) {
	var err error
	ctxWithTimeout, cancel := context.WithTimeout(ctx, gcsij.src.BucketTimeOut)
	defer cancel()

	obj := gcsij.bucket.Object(gcsij.object.Name)
	reader, err := obj.NewReader(ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from object with error : %w", err)
	}
	defer func() {
		err = reader.Close()
		if err != nil {
			err = fmt.Errorf("failed to close object reader with error : %w", err)
		}
	}()

	data := &bytes.Buffer{}

	_, err = data.ReadFrom(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from object reader with error : %w", err)
	}

	return data, err
}

func (gcsij *GcsInputJob) createEventFields(message string, data []mapstr.M) mapstr.M {
	fields := mapstr.M{
		"message": message, // original stringified data
		"log": mapstr.M{
			"file": mapstr.M{
				"path": gcsij.objectURI,
			},
		},
		"gcs": mapstr.M{
			"object": mapstr.M{
				"container": mapstr.M{
					"name": gcsij.src.BucketName,
				},
				"object": mapstr.M{
					"name":         gcsij.object.Name,
					"content_type": gcsij.object.ContentType,
					"data":         data, // objectified data
				},
			},
		},
		"cloud": mapstr.M{
			"provider": "goole cloud",
		},
		"event": mapstr.M{
			"kind": "publish_data",
		},
	}

	return fields
}
