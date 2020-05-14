package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_TopicSAMPT AWS CloudFormation Resource (AWS::Serverless::Function.TopicSAMPT)
// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
type Function_TopicSAMPT struct {

	// TopicName AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
	TopicName string `json:"TopicName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_TopicSAMPT) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.TopicSAMPT"
}
