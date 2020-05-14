package iot1click

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Project_PlacementTemplate AWS CloudFormation Resource (AWS::IoT1Click::Project.PlacementTemplate)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot1click-project-placementtemplate.html
type Project_PlacementTemplate struct {

	// DefaultAttributes AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot1click-project-placementtemplate.html#cfn-iot1click-project-placementtemplate-defaultattributes
	DefaultAttributes interface{} `json:"DefaultAttributes,omitempty"`

	// DeviceTemplates AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-iot1click-project-placementtemplate.html#cfn-iot1click-project-placementtemplate-devicetemplates
	DeviceTemplates interface{} `json:"DeviceTemplates,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Project_PlacementTemplate) AWSCloudFormationType() string {
	return "AWS::IoT1Click::Project.PlacementTemplate"
}
