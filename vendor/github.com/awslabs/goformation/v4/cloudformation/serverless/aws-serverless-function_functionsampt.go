package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_FunctionSAMPT AWS CloudFormation Resource (AWS::Serverless::Function.FunctionSAMPT)
// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
type Function_FunctionSAMPT struct {

	// FunctionName AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/docs/policy_templates.rst
	FunctionName string `json:"FunctionName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_FunctionSAMPT) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.FunctionSAMPT"
}
