package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_FunctionEnvironment AWS CloudFormation Resource (AWS::Serverless::Function.FunctionEnvironment)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#environment-object
type Function_FunctionEnvironment struct {

	// Variables AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#environment-object
	Variables map[string]string `json:"Variables,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_FunctionEnvironment) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.FunctionEnvironment"
}
