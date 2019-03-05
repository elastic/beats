// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

// KinesisFirehoseEvent represents the input event from Amazon Kinesis Firehose. It is used as the input parameter.
type KinesisFirehoseEvent struct {
	InvocationID      string                       `json:"invocationId"`
	DeliveryStreamArn string                       `json:"deliveryStreamArn"`
	Region            string                       `json:"region"`
	Records           []KinesisFirehoseEventRecord `json:"records"`
}

type KinesisFirehoseEventRecord struct {
	RecordID                    string                `json:"recordId"`
	ApproximateArrivalTimestamp MilliSecondsEpochTime `json:"approximateArrivalTimestamp"`
	Data                        []byte                `json:"data"`
}

// Constants used for describing the transformation result
const (
	KinesisFirehoseTransformedStateOk               = "Ok"
	KinesisFirehoseTransformedStateDropped          = "Dropped"
	KinesisFirehoseTransformedStateProcessingFailed = "ProcessingFailed"
)

type KinesisFirehoseResponse struct {
	Records []KinesisFirehoseResponseRecord `json:"records"`
}

type KinesisFirehoseResponseRecord struct {
	RecordID string `json:"recordId"`
	Result   string `json:"result"` // The status of the transformation. May be TransformedStateOk, TransformedStateDropped or TransformedStateProcessingFailed
	Data     []byte `json:"data"`
}
