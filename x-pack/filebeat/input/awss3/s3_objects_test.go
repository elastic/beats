// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func newS3Object(t testing.TB, filename, contentType string) (s3EventV2, *s3.GetObjectOutput) {
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	return newS3Event(filename), newS3GetObjectResponse(filename, data, contentType)
}

func newS3GetObjectResponse(filename string, data []byte, contentType string) *s3.GetObjectOutput {
	r := bytes.NewReader(data)
	contentLength := int64(r.Len())

	getObjectOutput := s3.GetObjectOutput{}
	getObjectOutput.ContentLength = &contentLength
	getObjectOutput.Body = io.NopCloser(r)
	if contentType != "" {
		getObjectOutput.ContentType = &contentType
	}

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gz":
		gzipEncoding := "gzip"
		getObjectOutput.ContentEncoding = &gzipEncoding
	}
	return &getObjectOutput
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
		cfg := conf.MustNewConfigFrom(map[string]interface{}{
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

	t.Run("split array with expand_event_list_from_field equals .[]", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{ExpandEventListFromField: ".[]"}}
		testProcessS3Object(t, "testdata/array.json", "application/json", 2, sel)
	})

	t.Run("split array without expand_event_list_from_field", func(t *testing.T) {
		sel := fileSelectorConfig{ReaderConfig: readerConfig{ExpandEventListFromField: ""}}
		testProcessS3Object(t, "testdata/array.json", "application/json", 1, sel)
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

		s3Event := newS3Event("log.txt")

		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(nil, errFakeConnectivityFailure)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupConfig{})
		err := s3ObjProc.Create(ctx, s3Event).ProcessS3Object(logp.NewLogger(inputName), func(_ beat.Event) {})
		require.Error(t, err)
		assert.True(t, errors.Is(err, errS3DownloadFailed), "expected errS3DownloadFailed")
	})

	t.Run("no error empty result in download", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)

		s3Event := newS3Event("log.txt")

		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(nil, nil)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupConfig{})
		err := s3ObjProc.Create(ctx, s3Event).ProcessS3Object(logp.NewLogger(inputName), func(_ beat.Event) {})
		require.Error(t, err)
	})

	t.Run("no content type in GetObject response", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		s3Event, s3Resp := newS3Object(t, "testdata/log.txt", "")

		gomock.InOrder(
			mockS3API.EXPECT().
				GetObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
				Return(s3Resp, nil),
		)

		var events []beat.Event
		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupConfig{})
		err := s3ObjProc.Create(ctx, s3Event).ProcessS3Object(logp.NewLogger(inputName), func(event beat.Event) {
			events = append(events, event)
		})
		assert.Equal(t, 2, len(events))
		require.NoError(t, err)
	})

	t.Run("backups objects on finalize call", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		s3Event, _ := newS3Object(t, "testdata/log.txt", "")

		backupCfg := backupConfig{
			BackupToBucketArn: "arn:aws:s3:::backup",
		}

		gomock.InOrder(
			mockS3API.EXPECT().
				CopyObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq("backup"), gomock.Eq(s3Event.S3.Object.Key), gomock.Eq(s3Event.S3.Object.Key)).
				Return(nil, nil),
		)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupCfg)
		err := s3ObjProc.Create(ctx, s3Event).FinalizeS3Object()
		require.NoError(t, err)
	})

	t.Run("deletes objects after backing up", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		s3Event, _ := newS3Object(t, "testdata/log.txt", "")

		backupCfg := backupConfig{
			BackupToBucketArn: "arn:aws:s3:::backup",
			Delete:            true,
		}

		gomock.InOrder(
			mockS3API.EXPECT().
				CopyObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq("backup"), gomock.Eq(s3Event.S3.Object.Key), gomock.Eq(s3Event.S3.Object.Key)).
				Return(nil, nil),
			mockS3API.EXPECT().
				DeleteObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
				Return(nil, nil),
		)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupCfg)
		err := s3ObjProc.Create(ctx, s3Event).FinalizeS3Object()
		require.NoError(t, err)
	})

	t.Run("prefixes objects when backing up", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		ctrl, ctx := gomock.WithContext(ctx, t)
		defer ctrl.Finish()
		mockS3API := NewMockS3API(ctrl)
		s3Event, _ := newS3Object(t, "testdata/log.txt", "")

		backupCfg := backupConfig{
			BackupToBucketArn:    s3Event.S3.Bucket.ARN,
			BackupToBucketPrefix: "backup/",
		}

		gomock.InOrder(
			mockS3API.EXPECT().
				CopyObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key), gomock.Eq("backup/testdata/log.txt")).
				Return(nil, nil),
		)

		s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, nil, backupCfg)
		err := s3ObjProc.Create(ctx, s3Event).FinalizeS3Object()
		require.NoError(t, err)
	})

	t.Run("text file without end of line marker", func(t *testing.T) {
		testProcessS3Object(t, "testdata/no-eol.txt", "text/plain", 1)
	})

	t.Run("text file without end of line marker but with newline", func(t *testing.T) {
		testProcessS3Object(t, "testdata/no-eol-twolines.txt", "text/plain", 2)
	})
}

func TestProcessObjectMetricCollection(t *testing.T) {
	logger := logp.NewLogger("testing-s3-processor-metrics")

	tests := []struct {
		name        string
		filename    string
		contentType string
		objectSize  int64
	}{
		{
			name:        "simple text - octet-stream",
			filename:    "testdata/log.txt",
			contentType: "application/octet-stream",
			objectSize:  18,
		},
		{
			name:        "json text",
			filename:    "testdata/log.json",
			contentType: "application/json",
			objectSize:  199,
		},
		{
			name:        "gzip with json text",
			filename:    "testdata/multiline.json.gz",
			contentType: "application/x-gzip",
			objectSize:  175,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()

			ctrl, ctx := gomock.WithContext(ctx, t)
			defer ctrl.Finish()

			s3Event, s3Resp := newS3Object(t, test.filename, test.contentType)
			mockS3API := NewMockS3API(ctrl)
			gomock.InOrder(
				mockS3API.EXPECT().
					GetObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
					Return(s3Resp, nil),
			)

			// metric recorder with zero workers
			metricRecorder := newInputMetrics(v2.Context{MetricsRegistry: monitoring.NewRegistry()}, 0)
			objFactory := newS3ObjectProcessorFactory(metricRecorder, mockS3API, nil, backupConfig{})
			objHandler := objFactory.Create(ctx, s3Event)

			// when
			err := objHandler.ProcessS3Object(logger, func(_ beat.Event) {})

			// then
			require.NoError(t, err)

			require.Equal(t, uint64(1), metricRecorder.s3ObjectsRequestedTotal.Get())
			require.Equal(t, uint64(0), metricRecorder.s3ObjectsInflight.Get())

			values := metricRecorder.s3ObjectSizeInBytes.Values()
			require.Equal(t, 1, len(values))

			// since we processed a single object, total and current process size is same
			require.Equal(t, test.objectSize, values[0])
			require.Equal(t, uint64(test.objectSize), metricRecorder.s3BytesProcessedTotal.Get())
		})
	}
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

	s3Event, s3Resp := newS3Object(t, file, contentType)
	var events []beat.Event
	gomock.InOrder(
		mockS3API.EXPECT().
			GetObject(gomock.Any(), gomock.Eq("us-east-1"), gomock.Eq(s3Event.S3.Bucket.Name), gomock.Eq(s3Event.S3.Object.Key)).
			Return(s3Resp, nil),
	)

	s3ObjProc := newS3ObjectProcessorFactory(nil, mockS3API, selectors, backupConfig{})
	err := s3ObjProc.Create(ctx, s3Event).ProcessS3Object(
		logp.NewLogger(inputName),
		func(event beat.Event) { events = append(events, event) })

	if !expectErr {
		require.NoError(t, err)
		assert.Equal(t, numEvents, len(events))
	} else {
		require.Error(t, err)
	}

	return events
}

// TestNewMockS3Pager verifies that newMockS3Pager is behaving similar to
// the AWS S3 Pager.
func TestNewMockS3Pager(t *testing.T) {
	fakeObjects := []types.Object{
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
	for pager.HasMorePages() {
		listObjectsV2Output, err := pager.NextPage(ctx)
		if err != nil {
			t.Fatal(err)
		}

		for _, s3Obj := range listObjectsV2Output.Contents {
			keys = append(keys, *s3Obj.Key)
		}
	}

	assert.Equal(t, []string{"foo", "bar", "baz"}, keys)
}

func Test_objectID(t *testing.T) {
	lastModified, _ := time.Parse("2006-01-02 15:04:05 -0700", "2024-11-07 12:44:22 +0000")
	objId := objectID(lastModified, "fe8a230c26", 42)

	assert.Equal(t, "1730983462000000000-fe8a230c26-000000000042", objId)
}

// newMockS3Pager returns a s3Pager that paginates the given s3Objects based on
// the specified page size. It never returns an error.
func newMockS3Pager(ctrl *gomock.Controller, pageSize int, s3Objects []types.Object) *MockS3Pager {
	mockS3Pager := NewMockS3Pager(ctrl)

	currentPage := -1
	numPages := len(s3Objects) / pageSize
	if len(s3Objects)%pageSize != 0 {
		numPages++
	}

	mockS3Pager.EXPECT().HasMorePages().Times(numPages + 1).DoAndReturn(func() bool {
		currentPage++
		next := currentPage*pageSize < len(s3Objects)
		return next
	})
	mockS3Pager.EXPECT().NextPage(gomock.Any()).AnyTimes().DoAndReturn(func(_ context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		startIdx := currentPage * pageSize
		endIdx := currentPage + 1*pageSize
		if endIdx > len(s3Objects) {
			endIdx = len(s3Objects)
		}
		return &s3.ListObjectsV2Output{
			Contents: s3Objects[startIdx:endIdx],
		}, nil
	})

	return mockS3Pager
}
