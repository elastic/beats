// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func TestPollingDiscoveryV2_listObjects(t *testing.T) {
	store := openTestStatestore()
	log := logptest.NewTestingLogger(t, t.Name())

	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:   log,
		Store: store,
	})
	require.NoError(t, err)
	defer reg.Close()

	now := time.Now().UTC()
	key1 := "logs/file1.log"
	key2 := "logs/file2.log"
	etag1 := "etag1"
	etag2 := "etag2"

	fakePager := &fakeS3Pager{
		pages: []s3.ListObjectsV2Output{
			{
				Contents: []s3types.Object{
					{Key: &key1, ETag: &etag1, LastModified: &now},
					{Key: &key2, ETag: &etag2, LastModified: &now},
				},
			},
		},
	}
	fakeS3 := &fakeS3API{pager: fakePager}

	metrics := newInputMetrics(monitoring.NewRegistry(), 0, log)
	defer metrics.Close()

	p := newPollingDiscoveryV2(pollingDiscoveryV2Config{
		S3:             fakeS3,
		Registry:       reg,
		Metrics:        metrics,
		Log:            log,
		Status:         &noopStatusReporter{},
		BucketARN:      "arn:aws:s3:::test-bucket",
		ListPrefix:     "logs/",
		NumWorkers:     1,
		Region:         "us-east-1",
		FilterProvider: newFilterProvider(&config{}),
	})

	workChan := make(chan state, 10)
	ids, numListed, ok := p.listObjects(t.Context(), workChan)

	assert.True(t, ok, "listing should succeed")
	assert.Equal(t, 2, numListed)
	assert.Len(t, ids, 2, "both objects should be tracked")
}

func TestPollingDiscoveryV2_listObjects_skips_processed(t *testing.T) {
	store := openTestStatestore()
	log := logptest.NewTestingLogger(t, t.Name())

	reg, err := newStateRegistryV2(stateRegistryV2Config{
		Log:   log,
		Store: store,
	})
	require.NoError(t, err)
	defer reg.Close()

	now := time.Now().UTC()
	key1 := "logs/file1.log"
	etag1 := "etag1"

	// Pre-mark file1 as processed.
	obj := s3EventV2{}
	obj.S3.Object.LastModified = now
	require.NoError(t, reg.MarkProcessed("test-bucket", key1, etag1, obj))

	key2 := "logs/file2.log"
	etag2 := "etag2"

	fakePager := &fakeS3Pager{
		pages: []s3.ListObjectsV2Output{
			{
				Contents: []s3types.Object{
					{Key: &key1, ETag: &etag1, LastModified: &now},
					{Key: &key2, ETag: &etag2, LastModified: &now},
				},
			},
		},
	}
	fakeS3 := &fakeS3API{pager: fakePager}

	metrics := newInputMetrics(monitoring.NewRegistry(), 0, log)
	defer metrics.Close()

	p := newPollingDiscoveryV2(pollingDiscoveryV2Config{
		S3:             fakeS3,
		Registry:       reg,
		Metrics:        metrics,
		Log:            log,
		Status:         &noopStatusReporter{},
		BucketARN:      "arn:aws:s3:::test-bucket",
		ListPrefix:     "logs/",
		NumWorkers:     1,
		Region:         "us-east-1",
		FilterProvider: newFilterProvider(&config{}),
	})

	workChan := make(chan state, 10)
	ids, _, ok := p.listObjects(t.Context(), workChan)

	assert.True(t, ok)
	// Both IDs are tracked for cleanup, but only file2 is sent to work channel.
	assert.Len(t, ids, 2)

	// workChan is closed by listObjects; drain and count dispatched items.
	var sent []state
	for st := range workChan {
		sent = append(sent, st)
	}
	require.Len(t, sent, 1, "only unprocessed object should be dispatched")
	assert.Equal(t, key2, sent[0].Key)
}

func TestPollingDiscoveryV2_stateToEvent(t *testing.T) {
	p := &pollingDiscoveryV2{
		bucketARN: "arn:aws:s3:::my-bucket",
		region:    "eu-west-1",
		provider:  "custom",
	}

	now := time.Now().UTC()
	st := state{Bucket: "my-bucket", Key: "path/obj.log", LastModified: now}
	evt := p.stateToEvent(st)

	assert.Equal(t, "eu-west-1", evt.AWSRegion)
	assert.Equal(t, "custom", evt.Provider)
	assert.Equal(t, "my-bucket", evt.S3.Bucket.Name)
	assert.Equal(t, "arn:aws:s3:::my-bucket", evt.S3.Bucket.ARN)
	assert.Equal(t, "path/obj.log", evt.S3.Object.Key)
	assert.Equal(t, now, evt.S3.Object.LastModified)
}

// fakeS3API is a minimal s3API implementation for testing listing.
type fakeS3API struct {
	pager *fakeS3Pager
}

func (f *fakeS3API) GetObject(ctx context.Context, region, bucket, key string) (*s3.GetObjectOutput, error) {
	return nil, nil
}

func (f *fakeS3API) CopyObject(ctx context.Context, region, from_bucket, to_bucket, from_key, to_key string) (*s3.CopyObjectOutput, error) {
	return nil, nil
}

func (f *fakeS3API) DeleteObject(ctx context.Context, region, bucket, key string) (*s3.DeleteObjectOutput, error) {
	return nil, nil
}

func (f *fakeS3API) ListObjectsPaginator(bucket, prefix, startAfterKey string) s3Pager {
	return f.pager
}

// fakeS3Pager is a minimal s3Pager for testing.
type fakeS3Pager struct {
	pages []s3.ListObjectsV2Output
	idx   int
}

func (p *fakeS3Pager) HasMorePages() bool {
	return p.idx < len(p.pages)
}

func (p *fakeS3Pager) NextPage(ctx context.Context, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if p.idx >= len(p.pages) {
		return &s3.ListObjectsV2Output{}, nil
	}
	page := p.pages[p.idx]
	p.idx++
	return &page, nil
}

// noopStatusReporter implements status.StatusReporter with no-ops.
type noopStatusReporter struct{}

func (*noopStatusReporter) UpdateStatus(_ status.Status, _ string) {}
