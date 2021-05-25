// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3iface"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
)

// MockS3Client struct is used for unit tests.
type MockS3Client struct {
	s3iface.ClientAPI
}

var (
	s3LogString1        = "36c1f test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5141F REST.HEAD.OBJECT Screen1.png\n"
	s3LogString1Trimmed = "36c1f test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5141F REST.HEAD.OBJECT Screen1.png"
	s3LogString2        = "28kdg test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5A070 REST.HEAD.OBJECT Screen2.png\n"
	s3LogString2Trimmed = "28kdg test-s3-ks [20/Jun/2019] 1.2.3.4 arn:aws:iam::1234:user/test@elastic.co 5A070 REST.HEAD.OBJECT Screen2.png"
	mockSvc             = &MockS3Client{}
	info                = s3Info{
		name:   "test-s3-ks",
		key:    "log2019-06-21-16-16-54",
		region: "us-west-1",
	}
)

func (m *MockS3Client) GetObjectRequest(input *s3.GetObjectInput) s3.GetObjectRequest {
	logBody := ioutil.NopCloser(bytes.NewReader([]byte(s3LogString1 + s3LogString2)))
	httpReq, _ := http.NewRequest("", "", nil)
	return s3.GetObjectRequest{
		Request: &awssdk.Request{
			Data: &s3.GetObjectOutput{
				Body: logBody,
			},
			HTTPRequest: httpReq,
		},
	}
}

func TestGetRegionFromQueueURL(t *testing.T) {
	casesPositive := []struct {
		title          string
		queueURL       string
		endpoint       string
		expectedRegion string
	}{
		{
			"QueueURL using amazonaws.com domain with blank Endpoint",
			"https://sqs.us-east-1.amazonaws.com/627959692251/test-s3-logs",
			"",
			"us-east-1",
		},
		{
			"QueueURL using abc.xyz and domain with matching Endpoint",
			"https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			"abc.xyz",
			"us-east-1",
		},
	}

	for _, c := range casesPositive {
		t.Run(c.title, func(t *testing.T) {
			regionName, err := getRegionFromQueueURL(c.queueURL, c.endpoint)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedRegion, regionName)
		})
	}

	casesNegative := []struct {
		title          string
		queueURL       string
		endpoint       string
		expectedRegion string
	}{
		{
			"QueueURL using abc.xyz and domain with blank Endpoint",
			"https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			"",
			"",
		},
		{
			"QueueURL using abc.xyz and domain with different Endpoint",
			"https://sqs.us-east-1.abc.xyz/627959692251/test-s3-logs",
			"googlecloud.com",
			"",
		},
		{
			"QueueURL is an invalid URL",
			":foo",
			"",
			"",
		},
	}

	for _, c := range casesNegative {
		t.Run(c.title, func(t *testing.T) {
			regionName, err := getRegionFromQueueURL(c.queueURL, c.endpoint)
			assert.Error(t, err)
			assert.Empty(t, regionName)
		})
	}

}

func TestHandleMessage(t *testing.T) {
	casesPositive := []struct {
		title           string
		message         sqs.Message
		expectedS3Infos []s3Info
	}{
		{
			"sqs message with event source aws:s3 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA",
				},
			},
		},
		{
			"sqs message with event source aws:s3 and event name ObjectCreated:CompleteMultipartUpload",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:CompleteMultipartUpload\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA",
				},
			},
		},
		{
			"sqs message with event source aws:s3, event name ObjectCreated:Put and encoded filename",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"year%3D2020/month%3D05/test1.txt\"}}}]}"),
			},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "year=2020/month=05/test1.txt",
				},
			},
		},
		{
			"sqs message with event source aws:s3, event name ObjectCreated:Put and gzip filename",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"428152502467_CloudTrail_us-east-2_20191219T1655Z_WXCas1PVnOaTpABD.json.gz\"}}}]}"),
			},
			[]s3Info{
				{
					name: "test-s3-ks-2",
					key:  "428152502467_CloudTrail_us-east-2_20191219T1655Z_WXCas1PVnOaTpABD.json.gz",
				},
			},
		},
	}

	p := &s3Collector{config: &config{}}
	for _, c := range casesPositive {
		t.Run(c.title, func(t *testing.T) {
			s3Info, err := p.handleSQSMessage(c.message)
			assert.NoError(t, err)
			assert.Equal(t, len(c.expectedS3Infos), len(s3Info))
			if len(s3Info) > 0 {
				assert.Equal(t, c.expectedS3Infos[0].key, s3Info[0].key)
				assert.Equal(t, c.expectedS3Infos[0].name, s3Info[0].name)
			}
		})
	}

	casesNegative := []struct {
		title           string
		message         sqs.Message
		expectedS3Infos []s3Info
	}{
		{
			"sqs message with event source aws:s3 and event name ObjectRemoved:Delete",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:s3\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectRemoved:Delete\",\"s3\":{\"configurationId\":\"object-removed-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{},
		},
		{
			"sqs message with event source aws:ec2 and event name ObjectCreated:Put",
			sqs.Message{
				Body: awssdk.String("{\"Records\":[{\"eventSource\":\"aws:ec2\",\"awsRegion\":\"ap-southeast-1\",\"eventTime\":\"2019-06-21T16:16:54.629Z\",\"eventName\":\"ObjectCreated:Put\",\"s3\":{\"configurationId\":\"object-created-event\",\"bucket\":{\"name\":\"test-s3-ks-2\",\"arn\":\"arn:aws:s3:::test-s3-ks-2\"},\"object\":{\"key\":\"server-access-logging2019-06-21-16-16-54-E68E4316CEB285AA\"}}}]}"),
			},
			[]s3Info{},
		},
	}

	for _, c := range casesNegative {
		t.Run(c.title, func(t *testing.T) {
			s3Info, err := p.handleSQSMessage(c.message)
			assert.Error(t, err)
			assert.Nil(t, s3Info)
		})
	}

}

func TestNewS3BucketReader(t *testing.T) {
	config := defaultConfig()
	p := &s3Collector{cancellation: context.TODO(), config: &config}
	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(info.name),
		Key:    awssdk.String(info.key),
	}
	req := mockSvc.GetObjectRequest(s3GetObjectInput)

	// The Context will interrupt the request if the timeout expires.
	var cancelFn func()
	ctx, cancelFn := context.WithTimeout(p.cancellation, p.config.APITimeout)
	defer cancelFn()

	resp, err := req.Send(ctx)
	assert.NoError(t, err)
	bodyReader := bufio.NewReader(resp.Body)
	defer resp.Body.Close()

	encFactory, ok := encoding.FindEncoding("plain")
	if !ok {
		t.Fatalf("unable to find 'plain' encoding")
	}

	enc, err := encFactory(bodyReader)
	if err != nil {
		t.Fatalf("failed to initialize encoding: %v", err)
	}

	var r reader.Reader
	r, err = readfile.NewEncodeReader(ioutil.NopCloser(bodyReader), readfile.Config{
		Codec:      enc,
		BufferSize: 4096,
		Terminator: readfile.LineFeed,
	})
	if err != nil {
		t.Fatalf("Failed to initialize line reader: %v", err)
	}

	r = readfile.NewStripNewline(r, readfile.LineFeed)

	for i := 0; i < 3; i++ {
		msg, err := r.Next()
		switch i {
		case 0:
			assert.NoError(t, err)
			assert.Equal(t, s3LogString1Trimmed, string(msg.Content))
		case 1:
			assert.NoError(t, err)
			assert.Equal(t, s3LogString2Trimmed, string(msg.Content))
		case 2:
			assert.Error(t, io.EOF, err)
			assert.Equal(t, "", string(msg.Content))
		}
	}
}

func TestCreateEvent(t *testing.T) {
	config := defaultConfig()
	p := &s3Collector{cancellation: context.TODO(), config: &config}
	errC := make(chan error)
	s3Context := &s3Context{
		refs: 1,
		errC: errC,
	}

	mockSvc := &MockS3Client{}
	s3Info := s3Info{
		name:   "test-s3-ks",
		key:    "log2019-06-21-16-16-54",
		region: "us-west-1",
		arn:    "arn:aws:s3:::test-s3-ks",
	}
	s3ObjectHash := s3ObjectHash(s3Info)

	s3GetObjectInput := &s3.GetObjectInput{
		Bucket: awssdk.String(info.name),
		Key:    awssdk.String(info.key),
	}
	req := mockSvc.GetObjectRequest(s3GetObjectInput)

	// The Context will interrupt the request if the timeout expires.
	var cancelFn func()
	ctx, cancelFn := context.WithTimeout(p.cancellation, p.config.APITimeout)
	defer cancelFn()

	resp, err := req.Send(ctx)
	assert.NoError(t, err)
	reader := bufio.NewReader(resp.Body)
	defer resp.Body.Close()

	var events []beat.Event
	for {
		log, err := reader.ReadString('\n')
		if log == "" {
			break
		}
		if err == io.EOF {
			event := createEvent(log, int64(len(log)), s3Info, s3ObjectHash, s3Context)
			events = append(events, event)
			break
		}

		event := createEvent(log, int64(len(log)), s3Info, s3ObjectHash, s3Context)
		events = append(events, event)
	}

	assert.Equal(t, 2, len(events))

	bucketName, err := events[0].Fields.GetValue("aws.s3.bucket.name")
	assert.NoError(t, err)
	assert.Equal(t, "test-s3-ks", bucketName.(string))

	objectKey, err := events[0].Fields.GetValue("aws.s3.object.key")
	assert.NoError(t, err)
	assert.Equal(t, "log2019-06-21-16-16-54", objectKey.(string))

	cloudProvider, err := events[0].Fields.GetValue("cloud.provider")
	assert.NoError(t, err)
	assert.Equal(t, "aws", cloudProvider)

	region, err := events[0].Fields.GetValue("cloud.region")
	assert.NoError(t, err)
	assert.Equal(t, "us-west-1", region)

	message1, err := events[0].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, s3LogString1, message1.(string))

	message2, err := events[1].Fields.GetValue("message")
	assert.NoError(t, err)
	assert.Equal(t, s3LogString2, message2.(string))

	s3Context.done()
}

func TestConstructObjectURL(t *testing.T) {
	cases := []struct {
		title             string
		s3Info            s3Info
		expectedObjectURL string
	}{
		{"construct with object in s3",
			s3Info{
				name:   "test-1",
				key:    "log2019-06-21-16-16-54",
				region: "us-west-1",
			},
			"https://test-1.s3-us-west-1.amazonaws.com/log2019-06-21-16-16-54",
		},
		{"construct with object in a folder of s3",
			s3Info{
				name:   "test-2",
				key:    "test-folder-1/test-log-1.txt",
				region: "us-east-1",
			},
			"https://test-2.s3-us-east-1.amazonaws.com/test-folder-1/test-log-1.txt",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			objectURL := constructObjectURL(c.s3Info)
			assert.Equal(t, c.expectedObjectURL, objectURL)
		})
	}
}

func TestCreateObjectID(t *testing.T) {
	cases := map[string]struct {
		offset int64
		want   string
	}{
		"object1": {
			123,
			"object1-000000000123",
		},
		"object2": {
			123456,
			"object2-000000123456",
		},
		"object3": {
			123456789123,
			"object3-123456789123",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			id := objectID(name, c.offset)
			assert.Equal(t, c.want, id)
		})
	}

}

func TestIsStreamGzipped(t *testing.T) {
	logBytes := []byte(`May 28 03:00:52 Shaunaks-MacBook-Pro-Work syslogd[119]: ASL Sender Statistics
May 28 03:03:29 Shaunaks-MacBook-Pro-Work VTDecoderXPCService[57953]: DEPRECATED USE in libdispatch client: Changing the target of a source after it has been activated; set a breakpoint on _dispatch_bug_deprecated to debug
May 28 03:03:29 Shaunaks-MacBook-Pro-Work VTDecoderXPCService[57953]: DEPRECATED USE in libdispatch client: Changing target queue hierarchy after xpc connection was activated; set a breakpoint on _dispatch_bug_deprecated to debug
`)

	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write(logBytes)
	require.NoError(t, err)

	err = gz.Close()
	require.NoError(t, err)

	tests := map[string]struct {
		contents []byte
		expected bool
	}{
		"not_gzipped": {
			logBytes,
			false,
		},
		"gzipped": {
			b.Bytes(),
			true,
		},
		"empty": {
			[]byte{},
			false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewReader(test.contents))
			actual, err := isStreamGzipped(r)

			require.NoError(t, err)
			require.Equal(t, test.expected, actual)
		})
	}
}

func TestTrimLogDelimiter(t *testing.T) {
	cases := []struct {
		title       string
		logOriginal string
		expectedLog string
	}{
		{"string with delimiter",
			`test
`,
			"test",
		},
		{"string without delimiter",
			"test",
			"test",
		},
		{"string just with delimiter",
			`
`,
			"",
		},
	}
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			log := trimLogDelimiter(c.logOriginal)
			assert.Equal(t, c.expectedLog, log)
		})
	}
}
