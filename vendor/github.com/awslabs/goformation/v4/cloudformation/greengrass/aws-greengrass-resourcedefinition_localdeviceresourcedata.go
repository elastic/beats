package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// ResourceDefinition_LocalDeviceResourceData AWS CloudFormation Resource (AWS::Greengrass::ResourceDefinition.LocalDeviceResourceData)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-localdeviceresourcedata.html
type ResourceDefinition_LocalDeviceResourceData struct {

	// GroupOwnerSetting AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-localdeviceresourcedata.html#cfn-greengrass-resourcedefinition-localdeviceresourcedata-groupownersetting
	GroupOwnerSetting *ResourceDefinition_GroupOwnerSetting `json:"GroupOwnerSetting,omitempty"`

	// SourcePath AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-resourcedefinition-localdeviceresourcedata.html#cfn-greengrass-resourcedefinition-localdeviceresourcedata-sourcepath
	SourcePath string `json:"SourcePath,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *ResourceDefinition_LocalDeviceResourceData) AWSCloudFormationType() string {
	return "AWS::Greengrass::ResourceDefinition.LocalDeviceResourceData"
}
