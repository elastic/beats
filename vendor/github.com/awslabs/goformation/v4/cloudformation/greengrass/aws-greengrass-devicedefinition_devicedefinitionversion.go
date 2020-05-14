package greengrass

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeviceDefinition_DeviceDefinitionVersion AWS CloudFormation Resource (AWS::Greengrass::DeviceDefinition.DeviceDefinitionVersion)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-devicedefinition-devicedefinitionversion.html
type DeviceDefinition_DeviceDefinitionVersion struct {

	// Devices AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-greengrass-devicedefinition-devicedefinitionversion.html#cfn-greengrass-devicedefinition-devicedefinitionversion-devices
	Devices []DeviceDefinition_Device `json:"Devices,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeviceDefinition_DeviceDefinitionVersion) AWSCloudFormationType() string {
	return "AWS::Greengrass::DeviceDefinition.DeviceDefinitionVersion"
}
