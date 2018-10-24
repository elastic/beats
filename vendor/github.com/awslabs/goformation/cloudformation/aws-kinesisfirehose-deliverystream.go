package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSKinesisFirehoseDeliveryStream AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html
type AWSKinesisFirehoseDeliveryStream struct {

	// DeliveryStreamName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-deliverystreamname
	DeliveryStreamName string `json:"DeliveryStreamName,omitempty"`

	// DeliveryStreamType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-deliverystreamtype
	DeliveryStreamType string `json:"DeliveryStreamType,omitempty"`

	// ElasticsearchDestinationConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-elasticsearchdestinationconfiguration
	ElasticsearchDestinationConfiguration *AWSKinesisFirehoseDeliveryStream_ElasticsearchDestinationConfiguration `json:"ElasticsearchDestinationConfiguration,omitempty"`

	// ExtendedS3DestinationConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-extendeds3destinationconfiguration
	ExtendedS3DestinationConfiguration *AWSKinesisFirehoseDeliveryStream_ExtendedS3DestinationConfiguration `json:"ExtendedS3DestinationConfiguration,omitempty"`

	// KinesisStreamSourceConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-kinesisstreamsourceconfiguration
	KinesisStreamSourceConfiguration *AWSKinesisFirehoseDeliveryStream_KinesisStreamSourceConfiguration `json:"KinesisStreamSourceConfiguration,omitempty"`

	// RedshiftDestinationConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-redshiftdestinationconfiguration
	RedshiftDestinationConfiguration *AWSKinesisFirehoseDeliveryStream_RedshiftDestinationConfiguration `json:"RedshiftDestinationConfiguration,omitempty"`

	// S3DestinationConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-s3destinationconfiguration
	S3DestinationConfiguration *AWSKinesisFirehoseDeliveryStream_S3DestinationConfiguration `json:"S3DestinationConfiguration,omitempty"`

	// SplunkDestinationConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-kinesisfirehose-deliverystream.html#cfn-kinesisfirehose-deliverystream-splunkdestinationconfiguration
	SplunkDestinationConfiguration *AWSKinesisFirehoseDeliveryStream_SplunkDestinationConfiguration `json:"SplunkDestinationConfiguration,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisFirehoseDeliveryStream) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisFirehoseDeliveryStream) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSKinesisFirehoseDeliveryStream) MarshalJSON() ([]byte, error) {
	type Properties AWSKinesisFirehoseDeliveryStream
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSKinesisFirehoseDeliveryStream) UnmarshalJSON(b []byte) error {
	type Properties AWSKinesisFirehoseDeliveryStream
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSKinesisFirehoseDeliveryStream(*res.Properties)
	}

	return nil
}

// GetAllAWSKinesisFirehoseDeliveryStreamResources retrieves all AWSKinesisFirehoseDeliveryStream items from an AWS CloudFormation template
func (t *Template) GetAllAWSKinesisFirehoseDeliveryStreamResources() map[string]AWSKinesisFirehoseDeliveryStream {
	results := map[string]AWSKinesisFirehoseDeliveryStream{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSKinesisFirehoseDeliveryStream:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::KinesisFirehose::DeliveryStream" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSKinesisFirehoseDeliveryStream
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSKinesisFirehoseDeliveryStreamWithName retrieves all AWSKinesisFirehoseDeliveryStream items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSKinesisFirehoseDeliveryStreamWithName(name string) (AWSKinesisFirehoseDeliveryStream, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSKinesisFirehoseDeliveryStream:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::KinesisFirehose::DeliveryStream" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSKinesisFirehoseDeliveryStream
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSKinesisFirehoseDeliveryStream{}, errors.New("resource not found")
}
