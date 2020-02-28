package kinesisfirehose

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeliveryStream_OutputFormatConfiguration AWS CloudFormation Resource (AWS::KinesisFirehose::DeliveryStream.OutputFormatConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-outputformatconfiguration.html
type DeliveryStream_OutputFormatConfiguration struct {

	// Serializer AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisfirehose-deliverystream-outputformatconfiguration.html#cfn-kinesisfirehose-deliverystream-outputformatconfiguration-serializer
	Serializer *DeliveryStream_Serializer `json:"Serializer,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeliveryStream_OutputFormatConfiguration) AWSCloudFormationType() string {
	return "AWS::KinesisFirehose::DeliveryStream.OutputFormatConfiguration"
}
