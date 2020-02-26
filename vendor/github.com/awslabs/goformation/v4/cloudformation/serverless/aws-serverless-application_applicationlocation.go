package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Application_ApplicationLocation AWS CloudFormation Resource (AWS::Serverless::Application.ApplicationLocation)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessapplication
type Application_ApplicationLocation struct {

	// ApplicationId AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessapplication
	ApplicationId string `json:"ApplicationId,omitempty"`

	// SemanticVersion AWS CloudFormation Property
	// Required: true
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#awsserverlessapplication
	SemanticVersion string `json:"SemanticVersion,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Application_ApplicationLocation) AWSCloudFormationType() string {
	return "AWS::Serverless::Application.ApplicationLocation"
}
