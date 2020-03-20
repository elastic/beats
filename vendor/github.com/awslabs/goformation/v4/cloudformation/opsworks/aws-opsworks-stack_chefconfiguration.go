package opsworks

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Stack_ChefConfiguration AWS CloudFormation Resource (AWS::OpsWorks::Stack.ChefConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-stack-chefconfiguration.html
type Stack_ChefConfiguration struct {

	// BerkshelfVersion AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-stack-chefconfiguration.html#cfn-opsworks-chefconfiguration-berkshelfversion
	BerkshelfVersion string `json:"BerkshelfVersion,omitempty"`

	// ManageBerkshelf AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-opsworks-stack-chefconfiguration.html#cfn-opsworks-chefconfiguration-berkshelfversion
	ManageBerkshelf bool `json:"ManageBerkshelf,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Stack_ChefConfiguration) AWSCloudFormationType() string {
	return "AWS::OpsWorks::Stack.ChefConfiguration"
}
