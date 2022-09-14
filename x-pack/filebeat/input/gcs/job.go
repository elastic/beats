// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"

	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type job struct {
	bucket    *storage.BucketHandle
	object    *storage.ObjectAttrs
	objectURI string
	state     *state
	src       *Source
	publisher cursor.Publisher
	log       *logp.Logger
	isFailed  bool
}

// newJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func newJob(bucket *storage.BucketHandle, object *storage.ObjectAttrs, objectURI string,
	state *state, src *Source, publisher cursor.Publisher, log *logp.Logger, isFailed bool,
) *job {
	return &job{
		bucket:    bucket,
		object:    object,
		objectURI: objectURI,
		state:     state,
		src:       src,
		publisher: publisher,
		log:       log,
		isFailed:  isFailed,
	}
}

const jobErrString = "job with jobId %s encountered an error : %w"

func (j *job) do(ctx context.Context, id string) {
	var fields mapstr.M

	if allowedContentTypes[j.object.ContentType] {
		data, err := j.extractData(ctx)
		if err != nil {
			j.state.updateFailedJobs(j.object.Name)
			j.log.Errorf(jobErrString, id, err)
			return
		}

		reader := io.NopCloser(bytes.NewReader(data.Bytes()))
		defer func() {
			err = reader.Close()
			if err != nil {
				j.log.Errorf("failed to close json reader with error : %w", err)
			}
		}()

		var objectData []mapstr.M
		switch j.object.ContentType {
		case contentTypeJSON:
			if j.src.ParseJSON {
				objectData, _, _, err = httpReadJSON(reader)
				if err != nil {
					j.log.Errorf(jobErrString, id, err)
					return
				}
			}
			// Support for more types will be added here, in the future.
		default:
			j.log.Errorf("job with jobId %s encountered an unexpected error", id)
			return
		}

		fields = j.createEventFields(data.String(), objectData)

	} else {
		err := fmt.Errorf("job with jobId %s encountered an error : content-type %s not supported", id, j.object.ContentType)
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
	j.state.save(j.object.Name, &j.object.Updated)

	if err := j.publisher.Publish(event, j.state.checkpoint()); err != nil {
		j.state.updateFailedJobs(j.object.Name)
		j.log.Errorf(jobErrString, id, err)
	}

}

func (j *job) Name() string {
	return j.object.Name
}

func (j *job) Source() *Source {
	return j.src
}

func (j *job) Timestamp() time.Time {
	return j.object.Updated
}

func (j *job) extractData(ctx context.Context) (*bytes.Buffer, error) {
	var err error
	ctxWithTimeout, cancel := context.WithTimeout(ctx, j.src.BucketTimeOut)
	defer cancel()
	obj := j.bucket.Object(j.object.Name)
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

func (j *job) createEventFields(message string, data []mapstr.M) mapstr.M {
	fields := mapstr.M{
		"message": message, // original stringified data
		"log": mapstr.M{
			"file": mapstr.M{
				"path": j.objectURI,
			},
		},
		"gcs": mapstr.M{
			"storage": mapstr.M{
				"bucket": mapstr.M{
					"name": j.src.BucketName,
				},
				"object": mapstr.M{
					"name":         j.object.Name,
					"content_type": j.object.ContentType,
					"json_data":    data, // objectified data
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
