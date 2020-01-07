package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_KinesisEvent AWS CloudFormation Resource (AWS::Serverless::Function.KinesisEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
type Function_KinesisEvent struct {

	// BatchSize AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	BatchSize int `json:"BatchSize,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	Enabled bool `json:"Enabled,omitempty"`

	// StartingPosition AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	StartingPosition string `json:"StartingPosition,omitempty"`

	// Stream AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#kinesis
	Stream string `json:"Stream,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_KinesisEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.KinesisEvent"
}
