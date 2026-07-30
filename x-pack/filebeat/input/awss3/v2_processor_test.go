// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestV2Processor_EventFieldsMatchContract(t *testing.T) {
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "my-bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::my-bucket"
	obj.S3.Object.Key = "logs/data.txt"
	obj.S3.Object.LastModified = time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	obj.AWSRegion = "eu-west-1"
	obj.Provider = "aws"
	obj.EventSource = "aws:s3"
	obj.EventName = "ObjectCreated:Put"

	fakeS3 := &fakeS3GetterV2{
		body:        "line1\nline2\n",
		contentType: "text/plain",
		requestURL:  "https://my-bucket.s3.eu-west-1.amazonaws.com/logs/data.txt",
	}

	log := logptest.NewTestingLogger(t, t.Name())
	metrics := newInputMetrics(monitoring.NewRegistry(), 1, log)
	proc := newObjectProcessorV2(fakeS3, metrics, nil, backupConfig{})

	var events []beat.Event
	n, err := proc.ProcessObject(t.Context(), log, obj, func(e beat.Event) {
		events = append(events, e)
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.Len(t, events, 2)

	e := events[0]

	// Verify field paths match the contract.
	v, _ := e.Fields.GetValue("message")
	assert.Equal(t, "line1", v)

	v, _ = e.Fields.GetValue("log.file.path")
	assert.Equal(t, "https://my-bucket.s3.eu-west-1.amazonaws.com/logs/data.txt", v)

	v, _ = e.Fields.GetValue("log.offset")
	assert.Equal(t, int64(0), v)

	v, _ = e.Fields.GetValue("aws.s3.bucket.name")
	assert.Equal(t, "my-bucket", v)

	v, _ = e.Fields.GetValue("aws.s3.bucket.arn")
	assert.Equal(t, "arn:aws:s3:::my-bucket", v)

	v, _ = e.Fields.GetValue("aws.s3.object.key")
	assert.Equal(t, "logs/data.txt", v)

	v, _ = e.Fields.GetValue("cloud.provider")
	assert.Equal(t, "aws", v)

	v, _ = e.Fields.GetValue("cloud.region")
	assert.Equal(t, "eu-west-1", v)

	// @metadata._id matches the legacy formula.
	expectedID := objectID(obj.S3.Object.LastModified, objectHashV2(obj), 0)
	gotID, err := e.GetValue("@metadata._id")
	require.NoError(t, err)
	assert.Equal(t, expectedID, gotID)

	// Second event has offset > 0.
	v, _ = events[1].Fields.GetValue("log.offset")
	assert.Equal(t, int64(6), v, "second line offset = len('line1\\n')")
}

func TestV2Processor_FileSelectorsSkip(t *testing.T) {
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "skip-me.csv"
	obj.AWSRegion = "us-east-1"

	fakeS3 := &fakeS3GetterV2{body: "data", contentType: "text/csv"}
	metrics := newInputMetrics(monitoring.NewRegistry(), 1, logptest.NewTestingLogger(t, t.Name()+"-metrics"))

	sel := mustFileSelector(t, `\.json$`)
	proc := newObjectProcessorV2(fakeS3, metrics, []fileSelectorConfig{sel}, backupConfig{})

	var events []beat.Event
	n, err := proc.ProcessObject(t.Context(), logptest.NewTestingLogger(t, t.Name()), obj, func(e beat.Event) {
		events = append(events, e)
	})
	require.NoError(t, err)
	assert.Equal(t, 0, n, "non-matching selector must produce zero events")
	assert.Empty(t, events)
}

func TestV2Processor_ExpandEventList(t *testing.T) {
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "events.json"
	obj.AWSRegion = "us-east-1"

	body := `{"Records":[{"action":"create"},{"action":"delete"}]}`
	fakeS3 := &fakeS3GetterV2{body: body, contentType: "application/json"}
	metrics := newInputMetrics(monitoring.NewRegistry(), 1, logptest.NewTestingLogger(t, t.Name()+"-metrics"))

	rc := defaultConfig().ReaderConfig
	rc.ExpandEventListFromField = "Records"
	rc.ContentType = "application/json"

	proc := newObjectProcessorV2(fakeS3, metrics, []fileSelectorConfig{{ReaderConfig: rc}}, backupConfig{})

	var events []beat.Event
	n, err := proc.ProcessObject(t.Context(), logptest.NewTestingLogger(t, t.Name()), obj, func(e beat.Event) {
		events = append(events, e)
	})
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	require.Len(t, events, 2)

	msg0, _ := events[0].Fields.GetValue("message")
	assert.Contains(t, msg0, `"action":"create"`)

	msg1, _ := events[1].Fields.GetValue("message")
	assert.Contains(t, msg1, `"action":"delete"`)
}

func TestV2Processor_DownloadError(t *testing.T) {
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "key"
	obj.AWSRegion = "us-east-1"

	fakeS3 := &fakeS3GetterV2{err: io.ErrUnexpectedEOF}
	metrics := newInputMetrics(monitoring.NewRegistry(), 1, logptest.NewTestingLogger(t, t.Name()+"-metrics"))
	proc := newObjectProcessorV2(fakeS3, metrics, nil, backupConfig{})

	_, err := proc.ProcessObject(t.Context(), logptest.NewTestingLogger(t, t.Name()), obj, func(beat.Event) {})
	require.Error(t, err)
	assert.ErrorIs(t, err, errS3DownloadFailed, "download errors must be wrapped as transient")
}

// --- test helpers ---

type fakeS3GetterV2 struct {
	body        string
	contentType string
	requestURL  string
	err         error
}

func (f *fakeS3GetterV2) GetObject(_ context.Context, _, _, _ string) (*s3.GetObjectOutput, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := &s3.GetObjectOutput{
		Body:        io.NopCloser(strings.NewReader(f.body)),
		ContentType: &f.contentType,
	}
	out.ResultMetadata = middleware.Metadata{}
	if f.requestURL != "" {
		out.ResultMetadata.Set(s3RequestURLMetadataKey, f.requestURL)
	}
	return out, nil
}

func mustFileSelector(t *testing.T, pattern string) fileSelectorConfig {
	t.Helper()
	var sel fileSelectorConfig
	sel.ReaderConfig = defaultConfig().ReaderConfig
	sel.Regex = &match.Matcher{}
	err := sel.Regex.Unpack(pattern)
	require.NoError(t, err)
	return sel
}
