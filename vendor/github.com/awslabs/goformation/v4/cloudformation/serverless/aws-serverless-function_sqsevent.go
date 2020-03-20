package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_SQSEvent AWS CloudFormation Resource (AWS::Serverless::Function.SQSEvent)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sqs
type Function_SQSEvent struct {

	// BatchSize AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sqs
	BatchSize int `json:"BatchSize,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sqs
	Enabled bool `json:"Enabled,omitempty"`

	// Queue AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#sqs
	Queue string `json:"Queue,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_SQSEvent) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.SQSEvent"
}
