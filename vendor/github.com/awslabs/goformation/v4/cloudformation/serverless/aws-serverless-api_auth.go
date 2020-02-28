package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Api_Auth AWS CloudFormation Resource (AWS::Serverless::Api.Auth)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#api-auth-object
type Api_Auth struct {

	// Authorizers AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#api-auth-object
	Authorizers interface{} `json:"Authorizers,omitempty"`

	// DefaultAuthorizer AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#api-auth-object
	DefaultAuthorizer string `json:"DefaultAuthorizer,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Api_Auth) AWSCloudFormationType() string {
	return "AWS::Serverless::Api.Auth"
}
