package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// FunctionDefinitionVersion_ResourceAccessPolicy AWS CloudFormation Resource (AWS::Greengrass::FunctionDefinitionVersion.ResourceAccessPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinitionversion-resourceaccesspolicy.html
type FunctionDefinitionVersion_ResourceAccessPolicy struct {

	// Permission AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinitionversion-resourceaccesspolicy.html#cfn-greengrass-functiondefinitionversion-resourceaccesspolicy-permission
	Permission string `json:"Permission,omitempty"`

	// ResourceId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-functiondefinitionversion-resourceaccesspolicy.html#cfn-greengrass-functiondefinitionversion-resourceaccesspolicy-resourceid
	ResourceId string `json:"ResourceId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *FunctionDefinitionVersion_ResourceAccessPolicy) AWSCloudFormationType() string {
	return "AWS::Greengrass::FunctionDefinitionVersion.ResourceAccessPolicy"
}
