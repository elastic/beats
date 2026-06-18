// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

// Contract tests pin the behavioral contracts that a drop-in replacement must
// preserve. These tests exist to detect regressions in the current
// implementation and to serve as a specification for the V2 rewrite.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// --- Event ID contract ---

func TestContract_EventID_Format(t *testing.T) {
	// @metadata._id = "{lastModifiedUnixNano}-{sha256(bucketARN+key)[:10]}-{offset:012d}"
	lastModified := time.Date(2024, 11, 7, 12, 44, 22, 0, time.UTC)
	bucketARN := "arn:aws:s3:::my-bucket"
	key := "path/to/object.json"
	offset := int64(1234)

	h := sha256.New()
	h.Write([]byte(bucketARN))
	h.Write([]byte(key))
	expectedHash := hex.EncodeToString(h.Sum(nil))[:10]
	expectedID := fmt.Sprintf("%d-%s-%012d", lastModified.UnixNano(), expectedHash, offset)

	// Verify via the production functions.
	obj := s3EventV2{}
	obj.S3.Bucket.ARN = bucketARN
	obj.S3.Object.Key = key
	gotHash := s3ObjectHash(obj)
	gotID := objectID(lastModified, gotHash, offset)

	assert.Equal(t, expectedHash, gotHash, "s3ObjectHash must produce first 10 hex chars of sha256(ARN+key)")
	assert.Equal(t, expectedID, gotID, "objectID must format as {unixNano}-{hash10}-{offset:012d}")
	assert.Equal(t, "1730983462000000000-"+expectedHash+"-000000001234", gotID)
}

func TestContract_EventID_ZeroLastModified(t *testing.T) {
	// In SQS mode, LastModified is often zero (bug #45566).
	zeroNano := time.Time{}.UnixNano()
	expected := fmt.Sprintf("%d-fe8a230c26-000000000000", zeroNano)
	gotID := objectID(time.Time{}, "fe8a230c26", 0)
	assert.Equal(t, expected, gotID,
		"zero time produces time.Time{}.UnixNano() prefix")
}

func TestContract_EventID_NegativeOffset_Omitted(t *testing.T) {
	// When offset < 0, createEvent does not set @metadata._id or log.offset.
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "key"
	obj.AWSRegion = "us-east-1"

	p := &s3ObjectProcessor{
		s3Obj:        obj,
		s3ObjHash:    s3ObjectHash(obj),
		s3RequestURL: "https://bucket.s3.us-east-1.amazonaws.com/key",
		ctx:          t.Context(),
	}
	event := p.createEvent("test message", -1)

	_, err := event.GetValue("@metadata._id")
	assert.Error(t, err, "negative offset must not set @metadata._id")

	_, err = event.GetValue("log.offset")
	assert.Error(t, err, "negative offset must not set log.offset")
}

// --- Event field paths contract ---

func TestContract_EventFieldPaths(t *testing.T) {
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "my-bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::my-bucket"
	obj.S3.Object.Key = "logs/2024/data.json"
	obj.S3.Object.LastModified = time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	obj.AWSRegion = "eu-west-1"
	obj.Provider = "aws"

	p := &s3ObjectProcessor{
		s3Obj:        obj,
		s3ObjHash:    s3ObjectHash(obj),
		s3RequestURL: "https://my-bucket.s3.eu-west-1.amazonaws.com/logs/2024/data.json",
		s3Metadata:   mapstr.M{"x-amz-server-side-encryption": "AES256"},
		ctx:          t.Context(),
	}
	event := p.createEvent("hello world", 42)

	// message
	v, _ := event.Fields.GetValue("message")
	assert.Equal(t, "hello world", v)

	// log.file.path
	v, _ = event.Fields.GetValue("log.file.path")
	assert.Equal(t, "https://my-bucket.s3.eu-west-1.amazonaws.com/logs/2024/data.json", v)

	// log.offset
	v, _ = event.Fields.GetValue("log.offset")
	assert.Equal(t, int64(42), v)

	// aws.s3.bucket.name
	v, _ = event.Fields.GetValue("aws.s3.bucket.name")
	assert.Equal(t, "my-bucket", v)

	// aws.s3.bucket.arn
	v, _ = event.Fields.GetValue("aws.s3.bucket.arn")
	assert.Equal(t, "arn:aws:s3:::my-bucket", v)

	// aws.s3.object.key
	v, _ = event.Fields.GetValue("aws.s3.object.key")
	assert.Equal(t, "logs/2024/data.json", v)

	// cloud.provider
	v, _ = event.Fields.GetValue("cloud.provider")
	assert.Equal(t, "aws", v)

	// cloud.region
	v, _ = event.Fields.GetValue("cloud.region")
	assert.Equal(t, "eu-west-1", v)

	// aws.s3.metadata (stored as map[string]interface{}, which is the underlying type of mapstr.M)
	v, _ = event.Fields.GetValue("aws.s3.metadata")
	assert.Equal(t, map[string]interface{}{"x-amz-server-side-encryption": "AES256"}, v)

	// @metadata._id
	expectedID := objectID(obj.S3.Object.LastModified, s3ObjectHash(obj), 42)
	gotID, err := event.GetValue("@metadata._id")
	require.NoError(t, err)
	assert.Equal(t, expectedID, gotID)
}

func TestContract_EventFieldPaths_NoMetadata(t *testing.T) {
	// When include_s3_metadata is not configured, aws.s3.metadata is absent.
	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "key"
	obj.AWSRegion = "us-east-1"

	p := &s3ObjectProcessor{
		s3Obj:        obj,
		s3ObjHash:    s3ObjectHash(obj),
		s3RequestURL: "https://bucket.s3.us-east-1.amazonaws.com/key",
		s3Metadata:   nil,
		ctx:          t.Context(),
	}
	event := p.createEvent("msg", 0)

	_, err := event.GetValue("aws.s3.metadata")
	assert.Error(t, err, "aws.s3.metadata must be absent when no metadata is configured")
}

// --- State store contract ---

func TestContract_StateStoreKeyFormat(t *testing.T) {
	bucket := "my-bucket"
	key := "path/to/file.log"
	etag := "abc123"
	lastModified := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)

	// Normal mode state ID
	normalID := stateID(bucket, key, etag, lastModified, false)
	expectedNormalID := bucket + key + etag + lastModified.String()
	assert.Equal(t, expectedNormalID, normalID, "normal state ID = bucket+key+etag+lastModified.String()")

	// Lexicographical mode state ID
	lexID := stateID(bucket, key, etag, lastModified, true)
	assert.Equal(t, expectedNormalID+"::lexicographical", lexID, "lex state ID appends ::lexicographical")

	// Full store key
	fullKey := awsS3ObjectStatePrefix + normalID
	assert.Equal(t, "filebeat::aws-s3::state::"+expectedNormalID, fullKey)
}

func TestContract_StateStoreKeyPrefix(t *testing.T) {
	assert.Equal(t, "filebeat::aws-s3::state::", awsS3ObjectStatePrefix)
	assert.Equal(t, "filebeat::aws-s3::tail", awsS3TailKey)
}

func TestContract_StateJSON_RoundTrip(t *testing.T) {
	original := state{
		Bucket:       "my-bucket",
		Key:          "path/to/file.log",
		Etag:         "abc123",
		LastModified: time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
		Stored:       true,
		Failed:       false,
	}

	data, err := json.Marshal(original)
	require.NoError(t, err, "state must be JSON-serializable")

	// Verify exact JSON field names.
	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &raw))

	expectedKeys := []string{"bucket", "key", "etag", "last_modified", "stored", "failed"}
	for _, k := range expectedKeys {
		_, ok := raw[k]
		assert.True(t, ok, "JSON must contain field %q", k)
	}

	// Round-trip.
	var decoded state
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, original.Bucket, decoded.Bucket)
	assert.Equal(t, original.Key, decoded.Key)
	assert.Equal(t, original.Etag, decoded.Etag)
	assert.True(t, original.LastModified.Equal(decoded.LastModified), "LastModified must round-trip")
	assert.Equal(t, original.Stored, decoded.Stored)
	assert.Equal(t, original.Failed, decoded.Failed)
}

func TestContract_StateJSON_FailedFieldName(t *testing.T) {
	// The field was renamed from "error" to "failed" in 8.14. The JSON tag
	// must be "failed" so that old entries (with "error") are not incorrectly
	// loaded as failed=true.
	s := state{Failed: true}
	data, _ := json.Marshal(s)
	assert.Contains(t, string(data), `"failed":true`)
	assert.NotContains(t, string(data), `"error"`)
}

func TestContract_StateRegistry_Persistence(t *testing.T) {
	log := logptest.NewTestingLogger(t, t.Name())
	store := openTestStatestore()

	bucket := "test-bucket"
	key := "test-key"
	etag := "test-etag"
	lastModified := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create registry, add a state, close.
	reg, err := newStateRegistry(log, store, "", false, 100)
	require.NoError(t, err)

	st := newState(bucket, key, etag, lastModified)
	st.Stored = true
	require.NoError(t, reg.AddState(st))
	reg.Close()

	// Reopen and verify persistence.
	reg2, err := newStateRegistry(log, store, "", false, 100)
	require.NoError(t, err)
	defer reg2.Close()

	id := stateID(bucket, key, etag, lastModified, false)
	assert.True(t, reg2.IsProcessed(id), "stored state must survive registry reload")
}

func TestContract_StateRegistry_FailedIsPermanent(t *testing.T) {
	log := logptest.NewTestingLogger(t, t.Name())
	store := openTestStatestore()

	bucket := "test-bucket"
	key := "test-key"
	etag := "test-etag"
	lastModified := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	reg, err := newStateRegistry(log, store, "", false, 100)
	require.NoError(t, err)

	st := newState(bucket, key, etag, lastModified)
	st.Failed = true
	require.NoError(t, reg.AddState(st))

	id := stateID(bucket, key, etag, lastModified, false)
	assert.True(t, reg.IsProcessed(id), "failed state must be treated as permanently processed")
	reg.Close()

	// Reload — failed state must persist.
	reg2, err := newStateRegistry(log, store, "", false, 100)
	require.NoError(t, err)
	defer reg2.Close()
	assert.True(t, reg2.IsProcessed(id), "failed state must survive reload")
}

// --- Notification parsing contract ---

func TestContract_NotificationParsing_S3Direct(t *testing.T) {
	body := `{
		"Records": [{
			"eventVersion": "2.2",
			"eventSource": "aws:s3",
			"awsRegion": "us-east-1",
			"eventName": "ObjectCreated:Put",
			"s3": {
				"bucket": {
					"name": "my-bucket",
					"arn": "arn:aws:s3:::my-bucket"
				},
				"object": {
					"key": "logs/2024/data.json.gz"
				}
			}
		}]
	}`

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, "us-east-1", events[0].AWSRegion)
	assert.Equal(t, "aws:s3", events[0].EventSource)
	assert.Equal(t, "ObjectCreated:Put", events[0].EventName)
	assert.Equal(t, "my-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "arn:aws:s3:::my-bucket", events[0].S3.Bucket.ARN)
	assert.Equal(t, "logs/2024/data.json.gz", events[0].S3.Object.Key)
}

func TestContract_NotificationParsing_URLDecode(t *testing.T) {
	// Object keys are URL-encoded in notifications (+ → space, %XX → char).
	body := `{
		"Records": [{
			"eventSource": "aws:s3",
			"awsRegion": "us-east-1",
			"eventName": "ObjectCreated:Put",
			"s3": {
				"bucket": {"name": "b", "arn": "arn:aws:s3:::b"},
				"object": {"key": "path/file+name%3D1.log"}
			}
		}]
	}`

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "path/file name=1.log", events[0].S3.Object.Key,
		"keys must be URL-decoded (+ → space, %%3D → =)")
}

func TestContract_NotificationParsing_SNSWrapped(t *testing.T) {
	inner := `{"Records":[{"eventSource":"aws:s3","awsRegion":"us-west-2","eventName":"ObjectCreated:Put","s3":{"bucket":{"name":"sns-bucket","arn":"arn:aws:s3:::sns-bucket"},"object":{"key":"data.log"}}}]}`
	body := fmt.Sprintf(`{"TopicArn":"arn:aws:sns:us-west-2:123456789012:my-topic","Message":%s}`, jsonString(inner))

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "sns-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "data.log", events[0].S3.Object.Key)
	assert.Equal(t, "us-west-2", events[0].AWSRegion)
}

func TestContract_NotificationParsing_EventBridge(t *testing.T) {
	body := `{
		"version": "0",
		"id": "example-id",
		"detail-type": "Object Created",
		"source": "aws.s3",
		"account": "123456789012",
		"time": "2024-01-01T00:00:00Z",
		"region": "eu-central-1",
		"resources": ["arn:aws:s3:::eb-bucket"],
		"detail": {
			"version": "0",
			"bucket": {"name": "eb-bucket"},
			"object": {"key": "events/file.json", "size": 1024, "etag": "abc"}
		}
	}`

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1)

	assert.Equal(t, "eb-bucket", events[0].S3.Bucket.Name)
	assert.Equal(t, "arn:aws:s3:::eb-bucket", events[0].S3.Bucket.ARN)
	assert.Equal(t, "events/file.json", events[0].S3.Object.Key)
	assert.Equal(t, "eu-central-1", events[0].AWSRegion)
	assert.Equal(t, "aws:s3", events[0].EventSource)
	assert.Equal(t, "ObjectCreated:Put", events[0].EventName)
}

func TestContract_NotificationParsing_TestEvent(t *testing.T) {
	body := `{"Event":"s3:TestEvent","Service":"Amazon S3","Time":"2024-01-01T00:00:00.000Z","Bucket":"test-bucket"}`

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	assert.NoError(t, err)
	assert.Empty(t, events, "s3:TestEvent must be silently skipped (nil events, no error)")
}

func TestContract_NotificationParsing_TestEventViaSNS(t *testing.T) {
	inner := `{"Event":"s3:TestEvent","Service":"Amazon S3","Time":"2024-01-01T00:00:00.000Z","Bucket":"test-bucket"}`
	body := fmt.Sprintf(`{"TopicArn":"arn:aws:sns:us-east-1:123456789012:topic","Message":%s}`, jsonString(inner))

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	assert.NoError(t, err)
	assert.Empty(t, events, "s3:TestEvent via SNS must be silently skipped")
}

func TestContract_NotificationParsing_MissingRecords(t *testing.T) {
	body := `{"someField": "value"}`

	proc := newTestNotificationProcessor(t)
	_, err := proc.getS3Notifications(body)
	assert.Error(t, err, "missing Records field must return an error")
	assert.Contains(t, err.Error(), "missing Records field")
}

func TestContract_NotificationParsing_NonObjectCreatedFiltered(t *testing.T) {
	body := `{
		"Records": [
			{"eventSource":"aws:s3","eventName":"ObjectCreated:Put","awsRegion":"us-east-1","s3":{"bucket":{"name":"b","arn":"a"},"object":{"key":"keep.log"}}},
			{"eventSource":"aws:s3","eventName":"ObjectRemoved:Delete","awsRegion":"us-east-1","s3":{"bucket":{"name":"b","arn":"a"},"object":{"key":"removed.log"}}}
		]
	}`

	proc := newTestNotificationProcessor(t)
	events, err := proc.getS3Notifications(body)
	require.NoError(t, err)
	require.Len(t, events, 1, "only ObjectCreated:* events are retained")
	assert.Equal(t, "keep.log", events[0].S3.Object.Key)
}

// --- expand_event_list_from_field contract ---

func TestContract_ExpandEventListFromField_NamedField(t *testing.T) {
	// {"Events": [{...}, {...}]} with expand field "Events"
	input := `{"Events":[{"time":"2021-05-25","msg":"hello"},{"time":"2021-05-26","msg":"world"}]}`

	events := runExpandTest(t, input, "Events")
	require.Len(t, events, 2)

	// Each event's message is the individual JSON object.
	assert.Contains(t, events[0], `"msg":"hello"`)
	assert.Contains(t, events[1], `"msg":"world"`)
}

func TestContract_ExpandEventListFromField_RootArray(t *testing.T) {
	// ".[]" means the root value is a JSON array.
	input := `[{"a":1},{"a":2}]`

	events := runExpandTest(t, input, ".[]")
	require.Len(t, events, 2)
	assert.Contains(t, events[0], `"a":1`)
	assert.Contains(t, events[1], `"a":2`)
}

func TestContract_ExpandEventListFromField_EmptyArray(t *testing.T) {
	input := `{"Records":[]}`

	events := runExpandTest(t, input, "Records")
	assert.Empty(t, events, "empty array produces zero events")
}

// --- Config validation contract ---

func TestContract_ConfigValidation_MutualExclusion(t *testing.T) {
	cases := []struct {
		name string
		cfg  map[string]interface{}
	}{
		{
			name: "queue_url_and_bucket_arn",
			cfg:  map[string]interface{}{"queue_url": "https://sqs.us-east-1.amazonaws.com/1234/queue", "bucket_arn": "arn:aws:s3:::b"},
		},
		{
			name: "bucket_arn_and_non_aws_bucket_name",
			cfg:  map[string]interface{}{"bucket_arn": "arn:aws:s3:::b", "non_aws_bucket_name": "nb", "region": "us-east-1"},
		},
		{
			name: "none_set",
			cfg:  map[string]interface{}{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := makeConfig(tc.cfg)
			err := c.Validate()
			assert.Error(t, err, "mutually exclusive source keys must fail validation")
		})
	}
}

func TestContract_ConfigValidation_SQS(t *testing.T) {
	t.Run("visibility_timeout_zero", func(t *testing.T) {
		c := validSQSConfig()
		c.VisibilityTimeout = 0
		assert.Error(t, c.Validate())
	})

	t.Run("visibility_timeout_over_12h", func(t *testing.T) {
		c := validSQSConfig()
		c.VisibilityTimeout = 13 * time.Hour
		assert.Error(t, c.Validate())
	})

	t.Run("api_timeout_less_than_sqs.wait_time", func(t *testing.T) {
		c := validSQSConfig()
		c.APITimeout = c.SQSWaitTime - time.Second
		assert.Error(t, c.Validate())
	})
}

func TestContract_ConfigValidation_Polling(t *testing.T) {
	t.Run("bucket_list_interval_zero", func(t *testing.T) {
		c := validPollingConfig()
		c.BucketListInterval = 0
		assert.Error(t, c.Validate())
	})

	t.Run("number_of_workers_zero", func(t *testing.T) {
		c := validPollingConfig()
		c.NumberOfWorkers = 0
		assert.Error(t, c.Validate())
	})
}

func TestContract_ConfigValidation_Lexicographical(t *testing.T) {
	t.Run("lexicographical_with_queue_url", func(t *testing.T) {
		c := validSQSConfig()
		c.LexicographicalOrdering = true
		assert.Error(t, c.Validate(), "lexicographical ordering requires polling mode")
	})

	t.Run("lookback_keys_zero", func(t *testing.T) {
		c := validPollingConfig()
		c.LexicographicalOrdering = true
		c.LexicographicalLookbackKeys = 0
		assert.Error(t, c.Validate())
	})
}

func TestContract_ConfigValidation_NonAWS(t *testing.T) {
	t.Run("fips_with_non_aws", func(t *testing.T) {
		c := config{}
		c.NonAWSBucketName = "bucket"
		c.AWSConfig.FIPSEnabled = true
		c.RegionName = "us-east-1"
		c.NumberOfWorkers = 1
		c.BucketListInterval = time.Minute
		assert.Error(t, c.Validate(), "FIPS not allowed with non-AWS")
	})

	t.Run("provider_requires_non_aws_bucket_name", func(t *testing.T) {
		c := validPollingConfig()
		c.ProviderOverride = "minio"
		assert.Error(t, c.Validate(), "provider requires non_aws_bucket_name")
	})
}

// --- Helpers ---

func newTestNotificationProcessor(t *testing.T) *sqsS3EventProcessor {
	t.Helper()
	return newSQSS3EventProcessor(logptest.NewTestingLogger(t, t.Name()), nil, nil, nil, time.Minute, 5, nil, nil)
}

func jsonString(s string) string {
	data, _ := json.Marshal(s)
	return string(data)
}

func runExpandTest(t *testing.T, input, field string) []string {
	t.Helper()

	obj := s3EventV2{}
	obj.S3.Bucket.Name = "bucket"
	obj.S3.Bucket.ARN = "arn:aws:s3:::bucket"
	obj.S3.Object.Key = "key"

	rc := &readerConfig{}
	rc.ExpandEventListFromField = field

	var events []string
	p := &s3ObjectProcessor{
		s3Obj:        obj,
		s3ObjHash:    s3ObjectHash(obj),
		readerConfig: rc,
		s3RequestURL: "https://bucket.s3.us-east-1.amazonaws.com/key",
		ctx:          t.Context(),
		eventCallback: func(e beat.Event) {
			msg, _ := e.Fields.GetValue("message")
			events = append(events, msg.(string)) //nolint:errcheck // Statically known to be a string.
		},
	}

	err := p.readJSON(strings.NewReader(input))
	require.NoError(t, err)
	return events
}

func makeConfig(overrides map[string]interface{}) config {
	c := defaultConfig()
	if v, ok := overrides["queue_url"]; ok {
		c.QueueURL = v.(string) //nolint:errcheck // Statically known to be a string.
	}
	if v, ok := overrides["bucket_arn"]; ok {
		c.BucketARN = v.(string) //nolint:errcheck // Statically known to be a string.
	}
	if v, ok := overrides["non_aws_bucket_name"]; ok {
		c.NonAWSBucketName = v.(string) //nolint:errcheck // Statically known to be a string.
	}
	if v, ok := overrides["region"]; ok {
		c.RegionName = v.(string) //nolint:errcheck // Statically known to be a string.
	}
	return c
}

func validSQSConfig() config {
	c := defaultConfig()
	c.QueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue"
	return c
}

func validPollingConfig() config {
	c := defaultConfig()
	c.BucketARN = "arn:aws:s3:::my-bucket"
	c.NumberOfWorkers = 5
	c.BucketListInterval = 2 * time.Minute
	return c
}
