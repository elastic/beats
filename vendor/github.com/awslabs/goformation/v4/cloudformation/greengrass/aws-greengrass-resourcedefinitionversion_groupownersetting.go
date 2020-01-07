package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ResourceDefinitionVersion_GroupOwnerSetting AWS CloudFormation Resource (AWS::Greengrass::ResourceDefinitionVersion.GroupOwnerSetting)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-groupownersetting.html
type ResourceDefinitionVersion_GroupOwnerSetting struct {

	// AutoAddGroupOwner AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-groupownersetting.html#cfn-greengrass-resourcedefinitionversion-groupownersetting-autoaddgroupowner
	AutoAddGroupOwner bool `json:"AutoAddGroupOwner"`

	// GroupOwner AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-groupownersetting.html#cfn-greengrass-resourcedefinitionversion-groupownersetting-groupowner
	GroupOwner string `json:"GroupOwner,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ResourceDefinitionVersion_GroupOwnerSetting) AWSCloudFormationType() string {
	return "AWS::Greengrass::ResourceDefinitionVersion.GroupOwnerSetting"
}
