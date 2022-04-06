// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	awscommon "github.com/elastic/beats/v7/x-pack/libbeat/common/aws"
)

func newS3Object(t testing.TB, filename, contentType string) (s3EventV2, *s3.GetObjectResponse) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	return newS3Event(filename), newS3GetObjectResponse(filename, data, contentType)
}

func newS3GetObjectResponse(filename string, data []byte, contentType string) *s3.GetObjectResponse {
	r := bytes.NewReader(data)
	contentLen := int64(r.Len())

	req := &s3.GetObjectRequest{
		Request: &awssdk.Request{
			HTTPRequest: &http.Request{
				URL: &url.URL{Path: filename},
			},
			Retryer: awssdk.NoOpRetryer{},
			Data: &s3.GetObjectOutput{
				Body:          ioutil.NopCloser(r),
				ContentLength: &contentLen,
			},
		},
		Input: &s3.GetObjectInput{
			Bucket: awssdk.String("dummy_bucket"),
			Key:    awssdk.String(filename),
		},
	}

	resp, _ := req.Send(context.Background())

	if contentType != "" {
		resp.ContentType = &contentType
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gz":
		gzipEncoding := "gzip"
		resp.ContentEncoding = &gzipEncoding
	}
	return resp
}

func TestS3ObjectProcessor(t *testing.T) {
	logp.TestingSetup()

	t.Run("download text/plain file", func(t *testing.T) {
		testProcessS3Object(t, "testdata/log.txt", "text/plain", 2)
	})

	t.Run("multiline content", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{}}
		sel.ReaderConfig.InitDefaults()

		// Unfortunately the config structs for the parser package are not
		// exported to use config parsing.
		cfg := common.MustNewConfigFrom(map[string]interface{}{
			"parsers": []map[string]interface{}{
				{
					"multiline": map[string]interface{}{
						"pattern": "^<Event",
						"negate":  true,
						"match":   "after",
					},
				},
			},
		})
		require.NoError(t, cfg.Unpack(&sel.ReaderConfig.Parsers))

		testProcessS3Object(t, "testdata/multiline.txt", "text/plain", 2, sel)
	})

	t.Run("application/json content-type", func(t *testing.T) {
		testProcessS3Object(t, "testdata/log.json", "application/json", 2)
	})

	t.Run("application/x-ndjson content-type", func(t *testing.T) {
		testProcessS3Object(t, "testdata/log.ndjson", "application/x-ndjson", 2)
	})

	t.Run("configured content-type", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{ContentType: contentTypeJSON}}
		testProcessS3Object(t, "testdata/multiline.json", "application/octet-stream", 2, sel)
	})

	t.Run("uncompress application/zip content", func(t *testing.T) {
		testProcessS3Object(t, "testdata/multiline.json.gz", "application/json", 2)
	})

	t.Run("unparsable json", func(t *testing.T) {
		testProcessS3ObjectError(t, "testdata/invalid.json", "application/json", 0)
	})

	t.Run("split array", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{ExpandEventListFromField: "Events"}}
		testProcessS3Object(t, "testdata/events-array.json", "application/json", 2, sel)
	})

	t.Run("split array error missing key", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{ExpandEventListFromField: "Records"}}
		testProcessS3ObjectError(t, "testdata/events-array.json", "application/json", 0, sel)
	})

	t.Run("events have a unique repeatable _id", func(t *testing.T) {
		// Hash of bucket ARN, object key, object versionId, and log offset.
		events := testProcessS3Object(t, "testdata/log.txt", "text/plain", 2)

		const idFieldName = "@metadata._id"
		for _, event := range events {
			v, _ := event.GetValue(idFieldName)
			if assert.NotNil(t, v, idFieldName+" is nil") {
				_id, ok := v.(string)
				if assert.True(t, ok, idFieldName+" is not a string") {
					assert.NotEmpty(t, _id, idFieldName+" is empty")
				}
			}
		}
	})

	t.Run("download error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)

		s3Event := newS3Event("log.txt")

		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockS3API, mockPublisher, nil)
		ack := awscommon.NewEventACKTracker(ctx)
		err := s3ObjProc.Create(ctx, logp.NewLogger(inputName), ack, s3Event).ProcessS3Object()
		require.Error(t, err)
		assert.True(t, errors.Is(err, errFakeConnectivityFailure), "expected errFakeConnectivityFailure error")
	})

	t.Run("no error empty result in download", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)

		s3Event := newS3Event("log.txt")

		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(nil, nil)

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockS3API, mockPublisher, nil)
		ack := awscommon.NewEventACKTracker(ctx)
		err := s3ObjProc.Create(ctx, logp.NewLogger(inputName), ack, s3Event).ProcessS3Object()
		require.Error(t, err)
	})

	t.Run("no content type in GetObject response", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		mockPublisher := NewMockBeatClient(ctrl)
		s3Event, s3Resp := newS3Object(t, "testdata/log.txt", "")

		var events []beat.Event
		gomock.InOrder(
			mockS3API.EXPECT().
				GetObject(gomock.Any(), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
				Return(s3Resp, nil),
			mockPublisher.EXPECT().
				Publish(gomock.Any()).
				Do(func(event beat.Event) { events = append(events, event) }).
				Times(2),
		)

		s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockS3API, mockPublisher, nil)
		ack := awscommon.NewEventACKTracker(ctx)
		err := s3ObjProc.Create(ctx, logp.NewLogger(inputName), ack, s3Event).ProcessS3Object()
		require.NoError(t, err)
	})
}

func testProcessS3Object(t testing.TB, file, contentType string, numEvents int, selectors ...fileSelectorConfig) []beat.Event {
	return _testProcessS3Object(t, file, contentType, numEvents, false, selectors)
}

func testProcessS3ObjectError(t testing.TB, file, contentType string, numEvents int, selectors ...fileSelectorConfig) []beat.Event {
	return _testProcessS3Object(t, file, contentType, numEvents, true, selectors)
}

func _testProcessS3Object(t testing.TB, file, contentType string, numEvents int, expectErr bool, selectors []fileSelectorConfig) []beat.Event {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ctrl, ctx := gomock.WithContext(ctx, t)
	defer ctrl.Finish()
	mockS3API := NewMockS3API(ctrl)
	mockPublisher := NewMockBeatClient(ctrl)

	s3Event, s3Resp := newS3Object(t, file, contentType)
	var events []beat.Event
	gomock.InOrder(
		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(s3Resp, nil),
		mockPublisher.EXPECT().
			Publish(gomock.Any()).
			Do(func(event beat.Event) { events = append(events, event) }).
			Times(numEvents),
	)

	s3ObjProc := newS3ObjectProcessorFactory(logp.NewLogger(inputName), nil, mockS3API, mockPublisher, selectors)
	ack := awscommon.NewEventACKTracker(ctx)
	err := s3ObjProc.Create(ctx, logp.NewLogger(inputName), ack, s3Event).ProcessS3Object()

	if !expectErr {
		require.NoError(t, err)
		assert.Equal(t, numEvents, len(events))
		assert.EqualValues(t, numEvents, ack.PendingACKs)
	} else {
		require.Error(t, err)
	}

	return events
}

// TestNewMockS3Pager verifies that newMockS3Pager is behaving similar to
// the AWS S3 Pager.
func TestNewMockS3Pager(t *testing.T) {
	fakeObjects := []s3.Object{
		{Key: awssdk.String("foo")},
		{Key: awssdk.String("bar")},
		{Key: awssdk.String("baz")},
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ctrl, ctx := gomock.WithContext(ctx, t)
	defer ctrl.Finish()
	mockS3Pager := newMockS3Pager(ctrl, 1, fakeObjects)
	mockS3API := NewMockS3API(ctrl)
	mockS3API.EXPECT().ListObjectsPaginator(gomock.Any(), "").Return(mockS3Pager)

	// Test the mock.
	var keys []string
	pager := mockS3API.ListObjectsPaginator("nombre", "")
	for pager.Next(ctx) {
		for _, s3Obj := range pager.CurrentPage().Contents {
			keys = append(keys, *s3Obj.Key)
		}
	}
	if err := pager.Err(); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []string{"foo", "bar", "baz"}, keys)
}

// newMockS3Pager returns a s3Pager that paginates the given s3Objects based on
// the specified page size. It never returns an error.
func newMockS3Pager(ctrl *gomock.Controller, pageSize int, s3Objects []s3.Object) *MockS3Pager {
	mockS3Pager := NewMockS3Pager(ctrl)

	currentPage := -1
	numPages := len(s3Objects) / pageSize
	if len(s3Objects)%pageSize != 0 {
		numPages++
	}

	mockS3Pager.EXPECT().Next(gomock.Any()).Times(numPages + 1).DoAndReturn(func(_ context.Context) interface{} {
		currentPage++
		next := currentPage*pageSize < len(s3Objects)
		return next
	})
	mockS3Pager.EXPECT().CurrentPage().AnyTimes().DoAndReturn(func() *s3.ListObjectsOutput {
		startIdx := currentPage * pageSize
		endIdx := currentPage + 1*pageSize
		if endIdx > len(s3Objects) {
			endIdx = len(s3Objects)
		}
		return &s3.ListObjectsOutput{
			Contents: s3Objects[startIdx:endIdx],
		}
	})
	mockS3Pager.EXPECT().Err().Return(nil)

	return mockS3Pager
}
