package kinesisfirehose

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeliveryStream_HiveJsonSerDe AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.HiveJsonSerDe)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-hivejsonserde.html
type DeliveryStream_HiveJsonSerDe struct {

	// TimestampFormats AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-hivejsonserde.html#cfn-kinesisfirehose-deliverystream-hivejsonserde-timestampformats
	TimestampFormats []string `json:"TimestampFormats,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeliveryStream_HiveJsonSerDe) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.HiveJsonSerDe"
}
