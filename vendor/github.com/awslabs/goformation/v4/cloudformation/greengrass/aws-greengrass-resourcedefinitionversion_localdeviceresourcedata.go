package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ResourceDefinitionVersion_LocalDeviceResourceData AWS CloudFormation Resource (AWS::Greengrass::ResourceDefinitionVersion.LocalDeviceResourceData)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-localdeviceresourcedata.html
type ResourceDefinitionVersion_LocalDeviceResourceData struct {

	// GroupOwnerSetting AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-localdeviceresourcedata.html#cfn-greengrass-resourcedefinitionversion-localdeviceresourcedata-groupownersetting
	GroupOwnerSetting *ResourceDefinitionVersion_GroupOwnerSetting `json:"GroupOwnerSetting,omitempty"`

	// SourcePath AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinitionversion-localdeviceresourcedata.html#cfn-greengrass-resourcedefinitionversion-localdeviceresourcedata-sourcepath
	SourcePath string `json:"SourcePath,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ResourceDefinitionVersion_LocalDeviceResourceData) AWSCloudFormationType() string {
	return "AWS::Greengrass::ResourceDefinitionVersion.LocalDeviceResourceData"
}
