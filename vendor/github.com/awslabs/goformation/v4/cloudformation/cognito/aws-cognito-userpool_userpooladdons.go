package cognito

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// UserPool_UserPoolAddOns AWS CloudFormation Resource (AWS::Cognito::UserPool.UserPoolAddOns)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-userpooladdons.html
type UserPool_UserPoolAddOns struct {

	// AdvancedSecurityMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cognito-userpool-userpooladdons.html#cfn-cognito-userpool-userpooladdons-advancedsecuritymode
	AdvancedSecurityMode string `json:"AdvancedSecurityMode,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *UserPool_UserPoolAddOns) AWSCloudFormationType() string {
	return "AWS::Cognito::UserPool.UserPoolAddOns"
}
