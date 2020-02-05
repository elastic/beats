package cognito

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// UserPoolResourceServer_ResourceServerScopeType AWS CloudFormation Resource (AWS::Cognito::UserPoolResourceServer.ResourceServerScopeType)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolresourceserver-resourceserverscopetype.html
type UserPoolResourceServer_ResourceServerScopeType struct {

	// ScopeDescription AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolresourceserver-resourceserverscopetype.html#cfn-cognito-userpoolresourceserver-resourceserverscopetype-scopedescription
	ScopeDescription string `json:"ScopeDescription,omitempty"`

	// ScopeName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpoolresourceserver-resourceserverscopetype.html#cfn-cognito-userpoolresourceserver-resourceserverscopetype-scopename
	ScopeName string `json:"ScopeName,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *UserPoolResourceServer_ResourceServerScopeType) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPoolResourceServer.ResourceServerScopeType"
}
