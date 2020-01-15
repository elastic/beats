package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_EventSource AWS CloudFormation Resource (AWS::Serverless::Function.EventSource)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#event-source-object
type Function_EventSource struct {

	// Properties AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#event-source-types
	Properties *Function_Properties `json:"Properties,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#event-source-object
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_EventSource) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.EventSource"
}
