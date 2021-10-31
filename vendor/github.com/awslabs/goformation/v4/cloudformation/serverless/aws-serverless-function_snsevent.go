package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_SNSEvent AWS CloudFormation Resource (AWS::Serverless::Function.SNSEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sns
type Function_SNSEvent struct {

	// Topic AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sns
	Topic string `json:"Topic,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_SNSEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.SNSEvent"
}
