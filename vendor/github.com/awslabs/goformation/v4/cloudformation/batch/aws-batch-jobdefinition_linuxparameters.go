package batch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// JobDefinition_LinuxParameters AWS CloudFormation Resource (AWS::Batch::JobDefinition.LinuxParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-containerproperties-linuxparameters.html
type JobDefinition_LinuxParameters struct {

	// Devices AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-containerproperties-linuxparameters.html#cfn-batch-jobdefinition-containerproperties-linuxparameters-devices
	Devices []JobDefinition_Device `json:"Devices,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *JobDefinition_LinuxParameters) AWSCloudFormationType() string {
	return "AWS::Batch::JobDefinition.LinuxParameters"
}
