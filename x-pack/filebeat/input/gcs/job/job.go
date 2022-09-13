// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type Job interface {
	Do(ctx context.Context, id string)
	Name() string
	Timestamp() time.Time
	Source() *types.Source
}

type GcsInputJob struct {
	bucket    *storage.BucketHandle
	object    *storage.ObjectAttrs
	objectURI string
	state     *state.State
	src       *types.Source
	publisher cursor.Publisher
	log       *logp.Logger
	isFailed  bool
}

// NewGcsInputJob, returns an instance of a job, which is a unit of work that can be assigned to a go routine
func NewGcsInputJob(bucket *storage.BucketHandle, object *storage.ObjectAttrs, objectURI string,
	state *state.State, src *types.Source, publisher cursor.Publisher, log *logp.Logger, isFailed bool,
) Job {
	return &GcsInputJob{
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

func (j *GcsInputJob) Do(ctx context.Context, id string) {
	var fields mapstr.M

	if types.AllowedContentTypes[j.object.ContentType] {
		data, err := j.extractData(ctx)
		if err != nil {
			j.state.UpdateFailedJobs(j.object.Name)
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
		case types.ContentTypeJSON:
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
	j.state.Save(j.object.Name, &j.object.Updated)

	if err := j.publisher.Publish(event, j.state.Checkpoint()); err != nil {
		j.state.UpdateFailedJobs(j.object.Name)
		j.log.Errorf(jobErrString, id, err)
	}

}

func (j *GcsInputJob) Name() string {
	return j.object.Name
}

func (j *GcsInputJob) Source() *types.Source {
	return j.src
}

func (j *GcsInputJob) Timestamp() time.Time {
	return j.object.Updated
}

func (j *GcsInputJob) extractData(ctx context.Context) (*bytes.Buffer, error) {
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

func (j *GcsInputJob) createEventFields(message string, data []mapstr.M) mapstr.M {
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
