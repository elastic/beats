package transformer

import (
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/beat"
)

func TestCloudWatchLogs(t *testing.T) {
	atime := time.Now().UTC()
	atime = atime.Add(-time.Duration(atime.Nanosecond())) // skip nanosecond precision
	e := events.CloudwatchLogsData{
		LogEvents: []events.CloudwatchLogsLogEvent{
			events.CloudwatchLogsLogEvent{
				ID:        "id1",
				Timestamp: atime.Unix() * 1000,
				Message:   "hello world",
			},
		},
	}

	bE := CloudwatchLogs(e)
	assert.Equal(t, len(e.LogEvents), len(bE))

	for idx, event := range e.LogEvents {
		assertValue(t, bE[idx], "event.id", event.ID)
		assertValue(t, bE[idx], "@timestamp", atime)
		assertValue(t, bE[idx], "message", event.Message)

		v, _ := bE[idx].GetValue("read_timestamp")
		assert.NotNil(t, v)
	}
}

func TestAPIGatewayProxy(t *testing.T) {
	request := events.APIGatewayProxyRequest{
		RequestContext: events.APIGatewayProxyRequestContext{RequestID: "id1"},
		Body:           "hello world",
		Resource:       "http://localhost/resource",
		Path:           "/resource",
		HTTPMethod:     "POST",
		Headers: map[string]string{
			"Content-Type": "application.json",
		},
		QueryStringParameters: map[string]string{
			"q": "_all",
		},
		PathParameters: map[string]string{
			"abc": "def",
		},
		IsBase64Encoded: false,
	}

	bE := APIGatewayProxy(request)

	assertValue(t, bE, "event.id", request.RequestContext.RequestID)
	assertValue(t, bE, "message", request.Body)
	assertValue(t, bE, "aws.api_gateway_proxy.resource", request.Resource)
	assertValue(t, bE, "aws.api_gateway_proxy.path", request.Path)
	assertValue(t, bE, "aws.api_gateway_proxy.method", request.HTTPMethod)
	assertValue(t, bE, "aws.api_gateway_proxy.headers", request.Headers)
	assertValue(t, bE, "aws.api_gateway_proxy.query_string", request.QueryStringParameters)
	assertValue(t, bE, "aws.api_gateway_proxy.path_parameters", request.PathParameters)
	assertValue(t, bE, "aws.api_gateway_proxy.is_base64_encoded", request.IsBase64Encoded)

	v, _ := bE.GetValue("read_timestamp")
	assert.NotNil(t, v)
}

func TestKinesis(t *testing.T) {
	e := events.KinesisEvent{
		Records: []events.KinesisEventRecord{
			events.KinesisEventRecord{
				EventID:        "abc123",
				Kinesis:        events.KinesisRecord{Data: []byte("Hello world")},
				EventName:      "event_name",
				EventSource:    "asource",
				EventSourceArn: "arn...",
				AwsRegion:      "us-east-1",
			},
		},
	}

	bE := Kinesis(e)
	assert.Equal(t, len(e.Records), len(bE))

	for idx, event := range e.Records {
		assertValue(t, bE[idx], "event.id", event.EventID)
		assertValue(t, bE[idx], "message", event.Kinesis.Data)
		assertValue(t, bE[idx], "cloud.region", event.AwsRegion)
		assertValue(t, bE[idx], "aws.event_name", event.EventName)
		assertValue(t, bE[idx], "aws.event_source", event.EventSource)
		assertValue(t, bE[idx], "aws.event_source_arn", event.EventSourceArn)

		v, _ := bE[idx].GetValue("read_timestamp")
		assert.NotNil(t, v)

		v, _ = bE[idx].GetValue("@timestamp")
		assert.NotNil(t, v)
	}
}

func TestSQS(t *testing.T) {
	e := events.SQSEvent{
		Records: []events.SQSMessage{
			events.SQSMessage{
				MessageId:      "abc123",
				Body:           "Hello world",
				EventSource:    "asource",
				EventSourceARN: "arn...",
				AWSRegion:      "us-east-1",
				ReceiptHandle:  "abc123",
				Attributes: map[string]string{
					"abc": "def",
				},
			},
		},
	}

	bE := SQS(e)
	assert.Equal(t, len(e.Records), len(bE))

	for idx, event := range e.Records {
		assertValue(t, bE[idx], "event.id", event.MessageId)
		assertValue(t, bE[idx], "message", event.Body)
		assertValue(t, bE[idx], "cloud.region", event.AWSRegion)
		assertValue(t, bE[idx], "aws.event_source", event.EventSource)
		assertValue(t, bE[idx], "aws.event_source_arn", event.EventSourceARN)
		assertValue(t, bE[idx], "aws.sqs.receipt_handle", event.ReceiptHandle)
		assertValue(t, bE[idx], "aws.sqs.attributes", event.Attributes)

		v, _ := bE[idx].GetValue("read_timestamp")
		assert.NotNil(t, v)

		v, _ = bE[idx].GetValue("@timestamp")
		assert.NotNil(t, v)
	}
}

func assertValue(t *testing.T, event beat.Event, field string, expected interface{}) {
	v, err := event.GetValue(field)
	if err != nil {
		t.Fatalf("Could not retrieve field: '%s', error: '%+v'", field, err)
	}

	assert.Equal(t, expected, v)
}
